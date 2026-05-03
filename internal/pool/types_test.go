package pool

import "testing"

func TestPowerOffConstrainsEquipmentOff(t *testing.T) {
	desired := DesiredState{
		Power:     BoolPtr(false),
		Filter:    BoolPtr(true),
		Heater:    BoolPtr(true),
		Jets:      BoolPtr(true),
		Bubbles:   BoolPtr(true),
		Sanitizer: BoolPtr(true),
	}.WithHardwareConstraints()

	if desired.Power == nil || *desired.Power {
		t.Fatalf("power = %+v, want off", desired.Power)
	}
	for name, value := range map[string]*bool{
		"filter":    desired.Filter,
		"heater":    desired.Heater,
		"jets":      desired.Jets,
		"bubbles":   desired.Bubbles,
		"sanitizer": desired.Sanitizer,
	} {
		if value == nil || *value {
			t.Fatalf("%s = %+v, want off when power is off", name, value)
		}
	}
}
