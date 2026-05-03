package intex

import (
	"testing"
)

func TestChecksumMatchesRubyClient(t *testing.T) {
	tests := map[string]string{
		StatusCommand:    "DA",
		CommandPower:     "98",
		CommandFilter:    "D4",
		CommandHeater:    "C8",
		CommandJets:      "C8",
		CommandBubbles:   "D4",
		CommandSanitizer: "D7",
		"8888050F0C24":   "AA",
		"":               "FF",
	}

	for data, want := range tests {
		got, err := Checksum(data)
		if err != nil {
			t.Fatalf("Checksum(%q): %v", data, err)
		}
		if got != want {
			t.Fatalf("Checksum(%q) = %s, want %s", data, got, want)
		}
	}
}

func TestDecodeStatusMatchesRubyBitLayout(t *testing.T) {
	data := statusData(statusParts{
		flags:   0b0010_0111,
		current: 32,
		target:  36,
	})
	status, err := DecodeStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Power || !status.Filter || !status.Heater || status.Jets || status.Bubbles || !status.Sanitizer {
		t.Fatalf("unexpected equipment flags: %+v", status)
	}
	if status.CurrentTemp == nil || *status.CurrentTemp != 32 {
		t.Fatalf("current temp = %v, want 32", status.CurrentTemp)
	}
	if status.TargetTemp != 36 {
		t.Fatalf("target temp = %d, want 36", status.TargetTemp)
	}
	if status.Unit != "\u00b0C" {
		t.Fatalf("unit = %q, want Celsius", status.Unit)
	}
}

func TestDecodeStatusErrorCode(t *testing.T) {
	data := statusData(statusParts{current: 181, target: 36})
	status, err := DecodeStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if status.CurrentTemp != nil {
		t.Fatalf("current temp = %v, want nil", status.CurrentTemp)
	}
	if status.ErrorCode != "E81" {
		t.Fatalf("error code = %q, want E81", status.ErrorCode)
	}
}

type statusParts struct {
	flags   byte
	current byte
	target  byte
}

func statusData(parts statusParts) string {
	payload := make([]byte, 13)
	payload[0] = parts.flags
	payload[2] = parts.current
	payload[10] = parts.target
	hexPayload := hexUpper(payload)
	withChecksum, err := AppendChecksum(hexPayload)
	if err != nil {
		panic(err)
	}
	return withChecksum
}

func hexUpper(bytes []byte) string {
	const table = "0123456789ABCDEF"
	out := make([]byte, len(bytes)*2)
	for i, b := range bytes {
		out[i*2] = table[b>>4]
		out[i*2+1] = table[b&0x0F]
	}
	return string(out)
}
