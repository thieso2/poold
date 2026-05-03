package intex

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientStatusRequest(t *testing.T) {
	addr, requests := fakeSpa(t, func(req request) string {
		if req.Type != 1 {
			t.Fatalf("request type = %d, want 1", req.Type)
		}
		wantData, _ := AppendChecksum(StatusCommand)
		if req.Data != wantData {
			t.Fatalf("request data = %s, want %s", req.Data, wantData)
		}
		return spaStatusResponse(statusData(statusParts{flags: 0b11, current: 31, target: 36}))
	})
	client := testClient(addr)

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.CurrentTemp == nil || *status.CurrentTemp != 31 || status.TargetTemp != 36 || !status.Power || !status.Filter {
		t.Fatalf("unexpected status: %+v", status)
	}
	if got := len(requests()); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestClientSetCommandFetchesUpdatedStatus(t *testing.T) {
	addr, requests := fakeSpa(t,
		func(req request) string {
			wantData, _ := AppendChecksum(CommandHeater)
			if req.Data != wantData {
				t.Fatalf("heater command data = %s, want %s", req.Data, wantData)
			}
			return `{"result":"ok","type":2,"data":"FF"}`
		},
		func(req request) string {
			wantData, _ := AppendChecksum(StatusCommand)
			if req.Data != wantData {
				t.Fatalf("status command data = %s, want %s", req.Data, wantData)
			}
			return spaStatusResponse(statusData(statusParts{flags: 0b111, current: 32, target: 36}))
		},
	)
	client := testClient(addr)

	status, err := client.Set(context.Background(), "heater", true)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Heater {
		t.Fatalf("heater = false, want true: %+v", status)
	}
	if got := len(requests()); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func TestClientRetriesBadStatusChecksum(t *testing.T) {
	addr, requests := fakeSpa(t,
		func(req request) string {
			return spaStatusResponse("00000000000000000000000000AA")
		},
		func(req request) string {
			return spaStatusResponse(statusData(statusParts{current: 30, target: 36}))
		},
	)
	client := testClient(addr)
	client.ProtocolRetries = 1

	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.CurrentTemp == nil || *status.CurrentTemp != 30 {
		t.Fatalf("unexpected status: %+v", status)
	}
	if got := len(requests()); got != 2 {
		t.Fatalf("requests = %d, want 2", got)
	}
}

func TestClientMalformedJSONReturnsError(t *testing.T) {
	addr, requests := fakeSpa(t, func(req request) string {
		return `{"result":`
	})
	client := testClient(addr)
	client.MaxRetries = 3

	if _, err := client.Status(context.Background()); err == nil {
		t.Fatal("expected malformed JSON error")
	}
	if got := len(requests()); got != 1 {
		t.Fatalf("requests = %d, want no retry after malformed JSON", got)
	}
}

func TestClientTimeout(t *testing.T) {
	var accepted atomic.Int64
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		accepted.Add(1)
		defer conn.Close()
		time.Sleep(500 * time.Millisecond)
	}()

	client := testClient(listener.Addr().String())
	client.Timeout = 30 * time.Millisecond
	client.MaxRetries = 0
	if _, err := client.Status(context.Background()); err == nil {
		t.Fatal("expected timeout")
	}
	if accepted.Load() != 1 {
		t.Fatalf("accepted connections = %d, want 1", accepted.Load())
	}
}

func TestCommandHexTemperature(t *testing.T) {
	command, err := CommandHex("target_temp", 36)
	if err != nil {
		t.Fatal(err)
	}
	if command != "8888050F0C24" {
		t.Fatalf("command = %s, want 8888050F0C24", command)
	}
}

func fakeSpa(t *testing.T, handlers ...func(request) string) (string, func() []request) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })

	var handled atomic.Int64
	requests := make(chan request, len(handlers)+5)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(time.Second))
				var req request
				if err := json.NewDecoder(conn).Decode(&req); err != nil {
					return
				}
				requests <- req
				index := int(handled.Add(1)) - 1
				if index >= len(handlers) {
					return
				}
				response := handlers[index](req)
				if strings.TrimSpace(response) == "" {
					return
				}
				_, _ = conn.Write([]byte(response + "\n"))
			}(conn)
		}
	}()

	return listener.Addr().String(), func() []request {
		var out []request
		for {
			select {
			case req := <-requests:
				out = append(out, req)
			default:
				return out
			}
		}
	}
}

func testClient(addr string) *Client {
	return &Client{
		Address:         addr,
		Timeout:         time.Second,
		MaxRetries:      0,
		RetryDelays:     nil,
		ProtocolRetries: 0,
		ProtocolDelays:  nil,
	}
}

func spaStatusResponse(data string) string {
	return `{"result":"ok","type":2,"data":"` + data + `"}`
}
