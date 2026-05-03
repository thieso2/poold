package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func main() {
	baseURL := envString("POOLCTL_URL", "http://127.0.0.1:8090")
	token := envString("POOLCTL_TOKEN", envString("POOLD_TOKEN", "dev-token"))

	fs := flag.NewFlagSet("poolctl", flag.ExitOnError)
	fs.StringVar(&baseURL, "url", baseURL, "poold base URL")
	fs.StringVar(&token, "token", token, "poold bearer token")
	fs.Parse(os.Args[1:])

	args := fs.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	c := client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: 30 * time.Second}}
	if err := run(c, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(c client, args []string) error {
	switch args[0] {
	case "status":
		var status pool.Status
		if err := c.doJSON(http.MethodGet, "/status", nil, &status); err != nil {
			return err
		}
		printJSON(status)
	case "watch":
		return c.watch()
	case "set":
		return runSet(c, args[1:])
	case "plans":
		return runPlans(c, args[1:])
	case "ready-by":
		return runReadyBy(c, args[1:])
	case "filter":
		return runFilter(c, args[1:])
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
	return nil
}

func runSet(c client, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: poolctl set temp 36 | heater on|off | filter on|off")
	}
	capability := args[0]
	if capability == "temp" || capability == "temperature" {
		temp, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		return postCommand(c, pool.CommandRequest{
			Capability: "target_temp",
			Value:      json.RawMessage(strconv.Itoa(temp)),
			Source:     "poolctl",
		})
	}
	state, err := parseOnOff(args[1])
	if err != nil {
		return err
	}
	return postCommand(c, pool.CommandRequest{
		Capability: capability,
		State:      pool.BoolPtr(state),
		Source:     "poolctl",
	})
}

func runPlans(c client, args []string) error {
	if len(args) == 1 && args[0] == "list" {
		var response struct {
			Plans []pool.Plan `json:"plans"`
		}
		if err := c.doJSON(http.MethodGet, "/plans", nil, &response); err != nil {
			return err
		}
		printJSON(response.Plans)
		return nil
	}
	if len(args) == 2 && args[0] == "apply" {
		body, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		var response any
		if err := c.doJSON(http.MethodPut, "/plans", body, &response); err != nil {
			return err
		}
		printJSON(response)
		return nil
	}
	return fmt.Errorf("usage: poolctl plans list | plans apply <file>")
}

func runReadyBy(c client, args []string) error {
	fs := flag.NewFlagSet("ready-by", flag.ContinueOnError)
	temp := fs.Int("temp", 36, "target temperature")
	at := fs.String("at", "", "ready time, e.g. Sat 08:30")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *at == "" {
		return fmt.Errorf("ready-by requires --at")
	}
	readyAt, err := parseLocalTime(*at)
	if err != nil {
		return err
	}
	plans, err := getPlans(c)
	if err != nil {
		return err
	}
	plans = append(plans, pool.Plan{
		ID:         fmt.Sprintf("ready-by-%d", time.Now().Unix()),
		Type:       pool.PlanReadyBy,
		Name:       fmt.Sprintf("Ready by %s", readyAt.Format("Mon 15:04")),
		Enabled:    true,
		TargetTemp: pool.IntPtr(*temp),
		At:         &readyAt,
	})
	return putPlans(c, plans)
}

func runFilter(c client, args []string) error {
	fs := flag.NewFlagSet("filter", flag.ContinueOnError)
	from := fs.String("from", "", "start time HH:MM")
	to := fs.String("to", "", "end time HH:MM")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		return fmt.Errorf("filter requires --from and --to")
	}
	if _, err := pool.ParseClock(*from); err != nil {
		return err
	}
	if _, err := pool.ParseClock(*to); err != nil {
		return err
	}
	plans, err := getPlans(c)
	if err != nil {
		return err
	}
	plans = append(plans, pool.Plan{
		ID:         fmt.Sprintf("filter-%d", time.Now().Unix()),
		Type:       pool.PlanTimeWindow,
		Name:       fmt.Sprintf("Filter %s-%s", *from, *to),
		Enabled:    true,
		Capability: "filter",
		From:       *from,
		To:         *to,
	})
	return putPlans(c, plans)
}

func postCommand(c client, command pool.CommandRequest) error {
	var response pool.CommandRecord
	if err := c.doJSON(http.MethodPost, "/commands", command, &response); err != nil {
		return err
	}
	printJSON(response)
	return nil
}

func getPlans(c client) ([]pool.Plan, error) {
	var response struct {
		Plans []pool.Plan `json:"plans"`
	}
	if err := c.doJSON(http.MethodGet, "/plans", nil, &response); err != nil {
		return nil, err
	}
	return response.Plans, nil
}

func putPlans(c client, plans []pool.Plan) error {
	var response any
	if err := c.doJSON(http.MethodPut, "/plans", map[string]any{"plans": plans}, &response); err != nil {
		return err
	}
	printJSON(response)
	return nil
}

func (c client) doJSON(method, path string, body any, out any) error {
	var reader io.Reader
	switch value := body.(type) {
	case nil:
	case []byte:
		reader = bytes.NewReader(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s", method, path, strings.TrimSpace(string(respBody)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

func (c client) watch() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/events/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("watch: %s", strings.TrimSpace(string(body)))
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			fmt.Println(strings.TrimPrefix(line, "data: "))
		}
	}
	return scanner.Err()
}

func parseOnOff(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "on", "true", "1", "yes":
		return true, nil
	case "off", "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected on or off")
	}
}

func parseLocalTime(value string) (time.Time, error) {
	location, err := time.LoadLocation(envString("POOLCTL_TIMEZONE", "Europe/Berlin"))
	if err != nil {
		return time.Time{}, err
	}
	now := time.Now().In(location)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04", "2006-01-02T15:04"} {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			return parsed, nil
		}
	}
	parts := strings.Fields(value)
	if len(parts) == 2 {
		weekday, ok := parseWeekday(parts[0])
		if !ok {
			return time.Time{}, fmt.Errorf("unknown weekday %q", parts[0])
		}
		clock, err := pool.ParseClock(parts[1])
		if err != nil {
			return time.Time{}, err
		}
		daysAhead := (int(weekday) - int(now.Weekday()) + 7) % 7
		candidate := time.Date(now.Year(), now.Month(), now.Day()+daysAhead, clock.Hour, clock.Minute, 0, 0, location)
		if !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, 7)
		}
		return candidate, nil
	}
	return time.Time{}, fmt.Errorf("unsupported time %q", value)
}

func parseWeekday(value string) (time.Weekday, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sun", "sunday":
		return time.Sunday, true
	case "mon", "monday":
		return time.Monday, true
	case "tue", "tues", "tuesday":
		return time.Tuesday, true
	case "wed", "wednesday":
		return time.Wednesday, true
	case "thu", "thur", "thurs", "thursday":
		return time.Thursday, true
	case "fri", "friday":
		return time.Friday, true
	case "sat", "saturday":
		return time.Saturday, true
	default:
		return 0, false
	}
}

func printJSON(value any) {
	encoded, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(encoded))
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  poolctl status
  poolctl watch
  poolctl set temp 36
  poolctl set heater on|off
  poolctl set filter on|off
  poolctl plans list
  poolctl plans apply <file>
  poolctl ready-by --temp 36 --at "Sat 08:30"
  poolctl filter --from "02:00" --to "04:00"`)
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
