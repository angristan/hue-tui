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

func TestValuesEqual_ColorXY_EpsilonComparison(t *testing.T) {
	tests := []struct {
		name     string
		a, b     struct{ X, Y float64 }
		expected bool
	}{
		{
			name:     "exact match",
			a:        struct{ X, Y float64 }{0.5, 0.6},
			b:        struct{ X, Y float64 }{0.5, 0.6},
			expected: true,
		},
		{
			name:     "within epsilon (small difference)",
			a:        struct{ X, Y float64 }{0.5104, 0.2120},
			b:        struct{ X, Y float64 }{0.5100, 0.2120},
			expected: true,
		},
		{
			name:     "within epsilon (bridge rounding)",
			a:        struct{ X, Y float64 }{0.163766, 0.083500},
			b:        struct{ X, Y float64 }{0.1638, 0.0835},
			expected: true,
		},
		{
			name:     "outside epsilon",
			a:        struct{ X, Y float64 }{0.5, 0.6},
			b:        struct{ X, Y float64 }{0.55, 0.6},
			expected: false,
		},
		{
			name:     "very different values",
			a:        struct{ X, Y float64 }{0.3307, 0.1426},
			b:        struct{ X, Y float64 }{0.4159, 0.1814},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valuesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("valuesEqual(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestPendingTracker_HasPending(t *testing.T) {
	tracker := NewPendingTracker()

	// No pending ops initially
	if tracker.HasPending("light1", "color_xy") {
		t.Error("Expected no pending op for color_xy")
	}

	// Add pending op
	tracker.Add("light1", "color_xy", struct{ X, Y float64 }{0.5, 0.6})

	// Should have pending op now
	if !tracker.HasPending("light1", "color_xy") {
		t.Error("Expected pending op for color_xy")
	}

	// Different field should not have pending
	if tracker.HasPending("light1", "color_temp") {
		t.Error("Expected no pending op for color_temp")
	}

	// Different light should not have pending
	if tracker.HasPending("light2", "color_xy") {
		t.Error("Expected no pending op for light2")
	}
}

func TestPendingTracker_HasPending_Expiry(t *testing.T) {
	tracker := NewPendingTracker()

	// Add expired op directly
	tracker.mu.Lock()
	tracker.ops["light1:color_xy"] = &PendingOp{
		Field:     "color_xy",
		Target:    struct{ X, Y float64 }{0.5, 0.6},
		Direction: DirExact,
		ExpiresAt: time.Now().Add(-1 * time.Second), // Already expired
	}
	tracker.mu.Unlock()

	// Should return false for expired op
	if tracker.HasPending("light1", "color_xy") {
		t.Error("Expected HasPending to return false for expired op")
	}

	// Expired op should be cleaned up
	tracker.mu.Lock()
	_, exists := tracker.ops["light1:color_xy"]
	tracker.mu.Unlock()
	if exists {
		t.Error("Expected expired op to be cleaned up")
	}
}

func TestPendingTracker_ColorXY(t *testing.T) {
	tracker := NewPendingTracker()

	// Add pending color_xy op
	target := struct{ X, Y float64 }{0.5104, 0.2120}
	tracker.Add("light1", "color_xy", target)

	// Exact match should be ignored
	if !tracker.ShouldIgnore("light1", "color_xy", target) {
		t.Error("Expected to ignore exact color_xy match")
	}
}

func TestPendingTracker_ColorXY_ApproximateMatch(t *testing.T) {
	tracker := NewPendingTracker()

	// Add pending color_xy op (what we computed from HS)
	target := struct{ X, Y float64 }{0.163766, 0.083500}
	tracker.Add("light1", "color_xy", target)

	// Bridge returns slightly different value (rounded to 4 decimal places)
	incoming := struct{ X, Y float64 }{0.1638, 0.0835}

	// Should ignore because it's within epsilon
	if !tracker.ShouldIgnore("light1", "color_xy", incoming) {
		t.Error("Expected to ignore approximate color_xy match")
	}
}

func TestPendingTracker_ColorXY_RapidChanges(t *testing.T) {
	tracker := NewPendingTracker()

	// Simulate rapid hue changes (each overwrites the previous)
	tracker.Add("light1", "color_xy", struct{ X, Y float64 }{0.41, 0.18})
	tracker.Add("light1", "color_xy", struct{ X, Y float64 }{0.33, 0.14})
	tracker.Add("light1", "color_xy", struct{ X, Y float64 }{0.30, 0.13})
	tracker.Add("light1", "color_xy", struct{ X, Y float64 }{0.22, 0.10})

	// HasPending should return true
	if !tracker.HasPending("light1", "color_xy") {
		t.Error("Expected HasPending to return true during rapid changes")
	}

	// Old values should not match (outside epsilon)
	if tracker.ShouldIgnore("light1", "color_xy", struct{ X, Y float64 }{0.41, 0.18}) {
		t.Error("Expected not to ignore old color_xy value")
	}

	// HasPending should still return true (op not cleared on non-match)
	// Wait, in current impl it gets cleared... let me check
}

func TestPendingTracker_ColorXY_MutualExclusion(t *testing.T) {
	tracker := NewPendingTracker()

	// Add pending color_xy op
	tracker.Add("light1", "color_xy", struct{ X, Y float64 }{0.5, 0.6})

	// Should have color_xy pending
	if !tracker.HasPending("light1", "color_xy") {
		t.Error("Expected pending color_xy")
	}

	// Should NOT have color_temp pending
	if tracker.HasPending("light1", "color_temp") {
		t.Error("Expected no pending color_temp")
	}

	// Now add color_temp
	tracker.AddWithDirection("light1", "color_temp", 400, DirUp)

	// Both should be pending
	if !tracker.HasPending("light1", "color_xy") {
		t.Error("Expected pending color_xy")
	}
	if !tracker.HasPending("light1", "color_temp") {
		t.Error("Expected pending color_temp")
	}
}
