package intex

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

func DecodeStatus(data string) (pool.Status, error) {
	data = strings.ToUpper(strings.TrimSpace(data))
	if err := VerifyChecksum(data); err != nil {
		return pool.Status{}, err
	}

	raw, ok := new(big.Int).SetString(data, 16)
	if !ok {
		return pool.Status{}, fmt.Errorf("invalid status hex")
	}

	currentRaw := byteAtShift(raw, 88)
	presetTemp := int(byteAtShift(raw, 24))

	status := pool.Status{
		ObservedAt: time.Now().UTC(),
		Connected:  true,
		Power:      raw.Bit(104) == 1,
		Filter:     raw.Bit(105) == 1,
		Heater:     raw.Bit(106) == 1,
		Jets:       raw.Bit(107) == 1,
		Bubbles:    raw.Bit(108) == 1,
		Sanitizer:  raw.Bit(109) == 1,
		TargetTemp: presetTemp,
		RawData:    data,
	}
	if presetTemp <= 40 {
		status.Unit = "\u00b0C"
	} else {
		status.Unit = "\u00b0F"
	}
	if currentRaw < 181 {
		status.CurrentTemp = pool.IntPtr(int(currentRaw))
	} else {
		status.ErrorCode = fmt.Sprintf("E%d", int(currentRaw)-100)
	}
	return status, nil
}

func byteAtShift(raw *big.Int, shift uint) uint64 {
	shifted := new(big.Int).Rsh(raw, shift)
	return new(big.Int).And(shifted, big.NewInt(0xFF)).Uint64()
}
