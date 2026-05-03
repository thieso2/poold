package intex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

const (
	StatusCommand = "8888060FEE0F01"

	CommandPower     = "8888060F014000"
	CommandFilter    = "8888060F010004"
	CommandHeater    = "8888060F010010"
	CommandJets      = "8888060F011000"
	CommandBubbles   = "8888060F010400"
	CommandSanitizer = "8888060F010001"
)

type PoolClient interface {
	Status(context.Context) (pool.Status, error)
	Set(context.Context, string, any) (pool.Status, error)
}

type Client struct {
	Address         string
	Timeout         time.Duration
	MaxRetries      int
	RetryDelays     []time.Duration
	ProtocolRetries int
	ProtocolDelays  []time.Duration
	Dialer          func(context.Context, string, string) (net.Conn, error)
}

type request struct {
	Data string `json:"data"`
	SID  string `json:"sid"`
	Type int    `json:"type"`
}

type response struct {
	Result string `json:"result"`
	Type   int    `json:"type"`
	Data   string `json:"data"`
}

func New(address string) *Client {
	return &Client{
		Address:         address,
		Timeout:         15 * time.Second,
		MaxRetries:      3,
		RetryDelays:     []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond},
		ProtocolRetries: 2,
		ProtocolDelays:  []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second},
	}
}

func (c *Client) Status(ctx context.Context) (pool.Status, error) {
	var lastErr error
	for attempt := 0; attempt <= c.ProtocolRetries; attempt++ {
		resp, err := c.sendSpaRequest(ctx, StatusCommand, 1)
		if err != nil {
			return pool.Status{}, err
		}
		if resp.Result != "ok" || resp.Type != 2 {
			lastErr = fmt.Errorf("invalid status response: result=%q type=%d", resp.Result, resp.Type)
		} else if strings.TrimSpace(resp.Data) == "" {
			lastErr = fmt.Errorf("empty status data")
		} else {
			status, decodeErr := DecodeStatus(resp.Data)
			if decodeErr == nil {
				return status, nil
			}
			lastErr = decodeErr
		}

		if attempt < c.ProtocolRetries {
			sleep(ctx, c.protocolDelay(attempt))
		}
	}
	return pool.Status{}, lastErr
}

func (c *Client) Set(ctx context.Context, capability string, value any) (pool.Status, error) {
	command, err := CommandHex(capability, value)
	if err != nil {
		return pool.Status{}, err
	}
	if _, err := c.sendSpaRequest(ctx, command, 1); err != nil {
		return pool.Status{}, err
	}
	return c.Status(ctx)
}

func (c *Client) Info(ctx context.Context) (map[string]string, error) {
	resp, err := c.sendSpaRequest(ctx, "", 3)
	if err != nil {
		return nil, err
	}
	if resp.Result != "ok" || resp.Type != 3 {
		return nil, fmt.Errorf("invalid info response: result=%q type=%d", resp.Result, resp.Type)
	}
	var data struct {
		IP    string `json:"ip"`
		UID   string `json:"uid"`
		DType string `json:"dtype"`
	}
	if err := json.Unmarshal([]byte(resp.Data), &data); err != nil {
		return nil, err
	}
	if data.DType != "spa" {
		return nil, fmt.Errorf("unexpected device type %q", data.DType)
	}
	return map[string]string{"ip": data.IP, "uid": data.UID, "dtype": data.DType}, nil
}

func CommandHex(capability string, value any) (string, error) {
	switch strings.ToLower(strings.TrimSpace(capability)) {
	case "power":
		return CommandPower, nil
	case "filter":
		return CommandFilter, nil
	case "heater", "heating":
		return CommandHeater, nil
	case "jets":
		return CommandJets, nil
	case "bubbles":
		return CommandBubbles, nil
	case "sanitizer":
		return CommandSanitizer, nil
	case "target_temp", "temperature", "temp", "preset_temp":
		temp, err := intValue(value)
		if err != nil {
			return "", err
		}
		if temp < 0 || temp > 255 {
			return "", fmt.Errorf("temperature out of one-byte range: %d", temp)
		}
		return fmt.Sprintf("8888050F0C%02X", temp), nil
	default:
		return "", fmt.Errorf("unknown capability %q", capability)
	}
}

func (c *Client) sendSpaRequest(ctx context.Context, data string, requestType int) (response, error) {
	payload := "FF"
	var err error
	if requestType == 1 {
		payload, err = AppendChecksum(data)
		if err != nil {
			return response{}, err
		}
	}

	req := request{
		Data: payload,
		SID:  strconv.FormatInt(time.Now().UnixNano()/int64(100*time.Microsecond), 10),
		Type: requestType,
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return response{}, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		resp, err := c.roundTrip(ctx, encoded)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retryable(err) || attempt == c.MaxRetries {
			break
		}
		sleep(ctx, c.retryDelay(attempt))
	}
	return response{}, lastErr
}

func (c *Client) roundTrip(ctx context.Context, encoded []byte) (response, error) {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	dialer := c.Dialer
	if dialer == nil {
		nd := &net.Dialer{Timeout: timeout}
		dialer = nd.DialContext
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dialer(ctx, "tcp", c.Address)
	if err != nil {
		return response{}, err
	}
	defer conn.Close()

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)
	}

	if _, err := conn.Write(encoded); err != nil {
		return response{}, err
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && strings.TrimSpace(line) != "") {
		return response{}, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return response{}, fmt.Errorf("empty response from spa")
	}

	var resp response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return response{}, err
	}
	return resp, nil
}

func (c *Client) retryDelay(attempt int) time.Duration {
	if len(c.RetryDelays) == 0 {
		return 0
	}
	if attempt < len(c.RetryDelays) {
		return c.RetryDelays[attempt]
	}
	return c.RetryDelays[len(c.RetryDelays)-1]
}

func (c *Client) protocolDelay(attempt int) time.Duration {
	if len(c.ProtocolDelays) == 0 {
		return 0
	}
	if attempt < len(c.ProtocolDelays) {
		return c.ProtocolDelays[attempt]
	}
	return c.ProtocolDelays[len(c.ProtocolDelays)-1]
}

func intValue(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		i, err := v.Int64()
		return int(i), err
	case string:
		i, err := strconv.Atoi(v)
		return i, err
	case []byte:
		var n json.Number
		if err := json.Unmarshal(v, &n); err == nil {
			i, err := n.Int64()
			return int(i), err
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return 0, err
		}
		return strconv.Atoi(s)
	default:
		return 0, fmt.Errorf("expected integer temperature, got %T", value)
	}
}

func retryable(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "connection refused") ||
		strings.Contains(message, "connection reset") ||
		strings.Contains(message, "empty response") ||
		strings.Contains(message, "broken pipe")
}

func sleep(ctx context.Context, delay time.Duration) {
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
