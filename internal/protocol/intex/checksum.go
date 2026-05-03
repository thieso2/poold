package intex

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func Checksum(data string) (string, error) {
	bytes, err := hex.DecodeString(strings.TrimSpace(data))
	if err != nil {
		return "", err
	}

	checksum := 0xFF
	for _, b := range bytes {
		checksum -= int(b)
	}
	checksum %= 0xFF
	if checksum < 0 {
		checksum += 0xFF
	}
	if checksum == 0x00 {
		checksum = 0xFF
	}
	return fmt.Sprintf("%02X", checksum), nil
}

func AppendChecksum(data string) (string, error) {
	checksum, err := Checksum(data)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(data)) + checksum, nil
}

func VerifyChecksum(data string) error {
	data = strings.ToUpper(strings.TrimSpace(data))
	if len(data) < 2 {
		return fmt.Errorf("response data is too short")
	}
	payload := data[:len(data)-2]
	want := data[len(data)-2:]
	got, err := Checksum(payload)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("checksum mismatch: got %s want %s", got, want)
	}
	return nil
}
