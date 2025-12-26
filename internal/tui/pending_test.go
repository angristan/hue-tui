package tui

import (
	"testing"
	"time"
)

func TestPendingTracker_ExactMatch(t *testing.T) {
	tracker := NewPendingTracker()

	// Add a pending op for on/off (exact match)
	tracker.Add("light1", "on", true)

	// Should ignore matching value
	if !tracker.ShouldIgnore("light1", "on", true) {
		t.Error("Expected to ignore matching on=true")
	}

	// After match, pending op should be cleared
	if tracker.ShouldIgnore("light1", "on", true) {
		t.Error("Expected pending op to be cleared after match")
	}
}

func TestPendingTracker_ExactMatch_NoMatch(t *testing.T) {
	tracker := NewPendingTracker()

	tracker.Add("light1", "on", true)

	// Should not ignore non-matching value
	if tracker.ShouldIgnore("light1", "on", false) {
		t.Error("Expected not to ignore non-matching on=false")
	}

	// Pending op should still exist (not cleared on non-match for exact)
	// Actually with current impl it doesn't clear - let's verify behavior
}

func TestPendingTracker_DirUp_IntermediateValues(t *testing.T) {
	tracker := NewPendingTracker()

	// Simulating brightness increase from 50 to 80
	tracker.AddWithDirection("light1", "brightness", 80, DirUp)

	// Intermediate values should be ignored
	if !tracker.ShouldIgnore("light1", "brightness", 55) {
		t.Error("Expected to ignore intermediate value 55 (< 80)")
	}

	if !tracker.ShouldIgnore("light1", "brightness", 70) {
		t.Error("Expected to ignore intermediate value 70 (< 80)")
	}

	// Target value should be ignored and clear the op
	if !tracker.ShouldIgnore("light1", "brightness", 80) {
		t.Error("Expected to ignore target value 80")
	}

	// After reaching target, should not ignore anymore
	if tracker.ShouldIgnore("light1", "brightness", 85) {
		t.Error("Expected not to ignore after target reached")
	}
}

func TestPendingTracker_DirUp_ExternalIncrease(t *testing.T) {
	tracker := NewPendingTracker()

	// User sets brightness to 60
	tracker.AddWithDirection("light1", "brightness", 60, DirUp)

	// External source sets it higher than our target
	if tracker.ShouldIgnore("light1", "brightness", 75) {
		t.Error("Expected not to ignore external value 75 (> 60 target)")
	}
}

func TestPendingTracker_DirDown_IntermediateValues(t *testing.T) {
	tracker := NewPendingTracker()

	// Simulating brightness decrease from 80 to 40
	tracker.AddWithDirection("light1", "brightness", 40, DirDown)

	// Intermediate values should be ignored
	if !tracker.ShouldIgnore("light1", "brightness", 70) {
		t.Error("Expected to ignore intermediate value 70 (> 40)")
	}

	if !tracker.ShouldIgnore("light1", "brightness", 50) {
		t.Error("Expected to ignore intermediate value 50 (> 40)")
	}

	// Target value should be ignored and clear the op
	if !tracker.ShouldIgnore("light1", "brightness", 40) {
		t.Error("Expected to ignore target value 40")
	}

	// After reaching target, should not ignore anymore
	if tracker.ShouldIgnore("light1", "brightness", 35) {
		t.Error("Expected not to ignore after target reached")
	}
}

func TestPendingTracker_DirDown_ExternalDecrease(t *testing.T) {
	tracker := NewPendingTracker()

	// User sets brightness to 40
	tracker.AddWithDirection("light1", "brightness", 40, DirDown)

	// External source sets it lower than our target
	if tracker.ShouldIgnore("light1", "brightness", 30) {
		t.Error("Expected not to ignore external value 30 (< 40 target)")
	}
}

func TestPendingTracker_RapidChanges(t *testing.T) {
	tracker := NewPendingTracker()

	// Simulate rapid brightness increases: 50 -> 60 -> 70 -> 80
	tracker.AddWithDirection("light1", "brightness", 60, DirUp)
	tracker.AddWithDirection("light1", "brightness", 70, DirUp)
	tracker.AddWithDirection("light1", "brightness", 80, DirUp)

	// Only the last target (80) should matter
	// All values up to 80 should be ignored
	if !tracker.ShouldIgnore("light1", "brightness", 55) {
		t.Error("Expected to ignore 55")
	}
	if !tracker.ShouldIgnore("light1", "brightness", 65) {
		t.Error("Expected to ignore 65")
	}
	if !tracker.ShouldIgnore("light1", "brightness", 75) {
		t.Error("Expected to ignore 75")
	}
	if !tracker.ShouldIgnore("light1", "brightness", 80) {
		t.Error("Expected to ignore 80 (target)")
	}
}

func TestPendingTracker_MultipleFields(t *testing.T) {
	tracker := NewPendingTracker()

	tracker.Add("light1", "on", true)
	tracker.AddWithDirection("light1", "brightness", 80, DirUp)

	// Both should work independently
	if !tracker.ShouldIgnore("light1", "on", true) {
		t.Error("Expected to ignore on=true")
	}
	if !tracker.ShouldIgnore("light1", "brightness", 70) {
		t.Error("Expected to ignore brightness 70")
	}
}

func TestPendingTracker_MultipleLights(t *testing.T) {
	tracker := NewPendingTracker()

	tracker.AddWithDirection("light1", "brightness", 80, DirUp)
	tracker.AddWithDirection("light2", "brightness", 40, DirDown)

	// Should handle each light independently
	if !tracker.ShouldIgnore("light1", "brightness", 70) {
		t.Error("Expected to ignore light1 brightness 70")
	}
	if !tracker.ShouldIgnore("light2", "brightness", 50) {
		t.Error("Expected to ignore light2 brightness 50")
	}

	// Wrong direction for each light
	if tracker.ShouldIgnore("light1", "brightness", 90) {
		t.Error("Expected not to ignore light1 brightness 90 (external increase)")
	}
	if tracker.ShouldIgnore("light2", "brightness", 30) {
		t.Error("Expected not to ignore light2 brightness 30 (external decrease)")
	}
}

func TestPendingTracker_Expiry(t *testing.T) {
	tracker := NewPendingTracker()

	// Add op with very short expiry for testing
	tracker.mu.Lock()
	tracker.ops["light1:brightness"] = &PendingOp{
		Field:     "brightness",
		Target:    80,
		Direction: DirUp,
		ExpiresAt: time.Now().Add(-1 * time.Second), // Already expired
	}
	tracker.mu.Unlock()

	// Should not ignore because op is expired
	if tracker.ShouldIgnore("light1", "brightness", 70) {
		t.Error("Expected not to ignore expired pending op")
	}
}

func TestPendingTracker_UnknownLight(t *testing.T) {
	tracker := NewPendingTracker()

	tracker.AddWithDirection("light1", "brightness", 80, DirUp)

	// Different light should not be ignored
	if tracker.ShouldIgnore("light2", "brightness", 70) {
		t.Error("Expected not to ignore unknown light")
	}
}

func TestPendingTracker_ColorTemp(t *testing.T) {
	tracker := NewPendingTracker()

	// Warmer = higher mirek
	tracker.AddWithDirection("light1", "color_temp", 400, DirUp)

	if !tracker.ShouldIgnore("light1", "color_temp", 350) {
		t.Error("Expected to ignore intermediate mirek 350")
	}
	if !tracker.ShouldIgnore("light1", "color_temp", 400) {
		t.Error("Expected to ignore target mirek 400")
	}

	// Cooler = lower mirek
	tracker.AddWithDirection("light1", "color_temp", 200, DirDown)

	if !tracker.ShouldIgnore("light1", "color_temp", 300) {
		t.Error("Expected to ignore intermediate mirek 300")
	}
	if !tracker.ShouldIgnore("light1", "color_temp", 200) {
		t.Error("Expected to ignore target mirek 200")
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		a, b     interface{}
		expected int
	}{
		{50, 60, -1},
		{60, 50, 1},
		{50, 50, 0},
		{50.0, 60.0, -1},
		{50, 50.0, 0},
		{int64(50), 60, -1},
		{uint16(50), 60, -1},
	}

	for _, tt := range tests {
		result := compareValues(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("compareValues(%v, %v) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		a, b     interface{}
		expected bool
	}{
		{true, true, true},
		{true, false, false},
		{false, false, true},
		{50, 50, true},
		{50, 60, false},
		{50.0, 50.0, true},
		{struct{ X, Y float64 }{0.5, 0.6}, struct{ X, Y float64 }{0.5, 0.6}, true},
		{struct{ X, Y float64 }{0.5, 0.6}, struct{ X, Y float64 }{0.5, 0.7}, false},
	}

	for _, tt := range tests {
		result := valuesEqual(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("valuesEqual(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
		}
	}
}
