package tui

import (
	"sync"
	"time"
)

const pendingOpExpiry = 5 * time.Second

// Direction represents the direction of a change
type Direction int

const (
	DirExact Direction = iota // Exact match required (for booleans)
	DirUp                     // Value is increasing
	DirDown                   // Value is decreasing
)

// PendingOp represents an in-flight operation that we're waiting for confirmation on
type PendingOp struct {
	Field     string      // "on", "brightness", "color_temp"
	Target    interface{} // target value we're moving toward
	Direction Direction   // direction of change
	ExpiresAt time.Time
}

// PendingTracker tracks pending operations to avoid flickering from event echoes
type PendingTracker struct {
	ops map[string]*PendingOp // keyed by lightID:field
	mu  sync.Mutex
}

// NewPendingTracker creates a new pending operations tracker
func NewPendingTracker() *PendingTracker {
	return &PendingTracker{
		ops: make(map[string]*PendingOp),
	}
}

// Add registers a pending operation for a light (exact match, for booleans)
func (t *PendingTracker) Add(lightID, field string, value interface{}) {
	t.AddWithDirection(lightID, field, value, DirExact)
}

// AddWithDirection registers a pending operation with a direction
func (t *PendingTracker) AddWithDirection(lightID, field string, target interface{}, dir Direction) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := lightID + ":" + field
	t.ops[key] = &PendingOp{
		Field:     field,
		Target:    target,
		Direction: dir,
		ExpiresAt: time.Now().Add(pendingOpExpiry),
	}
}

// ShouldIgnore checks if an incoming event should be ignored.
// Returns true if the event is "on the way" to our target or matches it.
// Clears the pending op if we've reached or passed the target.
func (t *PendingTracker) ShouldIgnore(lightID, field string, value interface{}) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := lightID + ":" + field
	op, exists := t.ops[key]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(op.ExpiresAt) {
		delete(t.ops, key)
		return false
	}

	switch op.Direction {
	case DirExact:
		// For exact matches (booleans), only ignore if value matches target
		if valuesEqual(op.Target, value) {
			delete(t.ops, key)
			return true
		}
		return false

	case DirUp:
		// We're increasing toward target
		// Ignore if value <= target (still on the way or reached)
		cmp := compareValues(value, op.Target)
		if cmp <= 0 {
			// If we reached or passed target, clear the op
			if cmp == 0 {
				delete(t.ops, key)
			}
			return true
		}
		// Value went higher than target - external change, don't ignore
		delete(t.ops, key)
		return false

	case DirDown:
		// We're decreasing toward target
		// Ignore if value >= target (still on the way or reached)
		cmp := compareValues(value, op.Target)
		if cmp >= 0 {
			// If we reached or passed target, clear the op
			if cmp == 0 {
				delete(t.ops, key)
			}
			return true
		}
		// Value went lower than target - external change, don't ignore
		delete(t.ops, key)
		return false
	}

	return false
}

// MatchesAndClear is the old API for backward compatibility - uses ShouldIgnore
func (t *PendingTracker) MatchesAndClear(lightID, field string, value interface{}) bool {
	return t.ShouldIgnore(lightID, field, value)
}

// Cleanup removes expired pending operations
func (t *PendingTracker) Cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for key, op := range t.ops {
		if now.After(op.ExpiresAt) {
			delete(t.ops, key)
		}
	}
}

// compareValues compares two numeric values
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func compareValues(a, b interface{}) int {
	af := toFloat64(a)
	bf := toFloat64(b)

	if af < bf {
		return -1
	} else if af > bf {
		return 1
	}
	return 0
}

// toFloat64 converts a numeric value to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	case uint16:
		return float64(val)
	case uint8:
		return float64(val)
	}
	return 0
}

// valuesEqual compares two values for equality (exact match)
func valuesEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
	case int:
		return toFloat64(a) == toFloat64(b)
	case float64:
		return toFloat64(a) == toFloat64(b)
	case struct{ X, Y float64 }:
		if bv, ok := b.(struct{ X, Y float64 }); ok {
			return av.X == bv.X && av.Y == bv.Y
		}
	}
	return false
}
