package models

// Light represents a Philips Hue light
type Light struct {
	// Unique identifier from the bridge
	ID string
	// User-friendly name
	Name string
	// Current on/off state
	On bool
	// Brightness level (0-254)
	Brightness uint8
	// Whether the light is reachable on the network
	Reachable bool
	// Color state (may be nil for non-color lights)
	Color *Color
	// Whether the light supports color
	SupportsColor bool
	// Whether the light supports color temperature
	SupportsColorTemp bool
	// ID of the room this light belongs to (empty if ungrouped)
	RoomID string
	// Device ID that owns this light service
	DeviceID string
}

// BrightnessPct returns the brightness as a percentage (0-100)
func (l *Light) BrightnessPct() int {
	return int(float64(l.Brightness) / 254.0 * 100)
}

// SetBrightnessPct sets brightness from a percentage (0-100)
func (l *Light) SetBrightnessPct(pct int) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	l.Brightness = uint8(float64(pct) / 100.0 * 254)
}

// IsColorLight returns true if the light supports any color features
func (l *Light) IsColorLight() bool {
	return l.SupportsColor || l.SupportsColorTemp
}

// Clone creates a deep copy of the light
func (l *Light) Clone() *Light {
	clone := *l
	if l.Color != nil {
		colorCopy := *l.Color
		clone.Color = &colorCopy
	}
	return &clone
}
