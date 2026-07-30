package pool

import (
	"testing"
	"time"
)

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

func TestEquipmentOnConstrainsPowerOn(t *testing.T) {
	cases := map[string]DesiredState{
		"filter":    {Filter: BoolPtr(true)},
		"heater":    {Heater: BoolPtr(true)},
		"jets":      {Jets: BoolPtr(true)},
		"bubbles":   {Bubbles: BoolPtr(true)},
		"sanitizer": {Sanitizer: BoolPtr(true)},
	}
	for name, input := range cases {
		desired := input.WithHardwareConstraints()
		if desired.Power == nil || !*desired.Power {
			t.Fatalf("%s on should imply power on: %+v", name, desired)
		}
	}
}

func TestTimeWindowPlanValidation(t *testing.T) {
	valid := Plan{
		ID:              "morning-filter",
		Type:            PlanTimeWindow,
		Enabled:         true,
		Capability:      "filter",
		Start:           "06:00",
		DurationMinutes: 120,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	cases := map[string]func(Plan) Plan{
		"heater capability": func(p Plan) Plan { p.Capability = "heater"; return p },
		"heating alias":     func(p Plan) Plan { p.Capability = "Heating"; return p },
		"missing start":     func(p Plan) Plan { p.Start = ""; return p },
		"bad start":         func(p Plan) Plan { p.Start = "25:00"; return p },
		"zero duration":     func(p Plan) Plan { p.DurationMinutes = 0; return p },
		"over one day":      func(p Plan) Plan { p.DurationMinutes = 1441; return p },
	}
	for name, mutate := range cases {
		if err := mutate(valid).Validate(); err == nil {
			t.Fatalf("Validate() error = nil, want rejection for %s", name)
		}
	}
}

func TestReadyByPlanValidationAllowsCron(t *testing.T) {
	plan := Plan{
		ID:         "weekend-ready",
		Type:       PlanReadyBy,
		Enabled:    true,
		TargetTemp: IntPtr(36),
		Cron:       "30 8 * * sat,sun",
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestReadyByPlanValidationRequiresOneSchedule(t *testing.T) {
	at := time.Date(2026, 5, 9, 8, 30, 0, 0, time.UTC)
	plan := Plan{
		ID:         "ready",
		Type:       PlanReadyBy,
		Enabled:    true,
		TargetTemp: IntPtr(36),
		At:         &at,
		Cron:       "30 8 * * sat",
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want schedule conflict")
	}
}

func TestNextCronTime(t *testing.T) {
	loc := time.FixedZone("TEST", 2*60*60)
	next, ok, err := NextCronTime("30 8 * * sat", time.Date(2026, 5, 8, 23, 0, 0, 0, loc), loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 9, 8, 30, 0, 0, loc)
	if !ok || !next.Equal(want) {
		t.Fatalf("next = %s ok=%v, want %s", next, ok, want)
	}
}

func TestNextCronTimeIncludesCurrentMinute(t *testing.T) {
	loc := time.FixedZone("TEST", 2*60*60)
	next, ok, err := NextCronTime("30 8 * * sat", time.Date(2026, 5, 9, 8, 30, 45, 0, loc), loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 5, 9, 8, 30, 0, 0, loc)
	if !ok || !next.Equal(want) {
		t.Fatalf("next = %s ok=%v, want %s", next, ok, want)
	}
}
