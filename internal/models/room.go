package models

// Room represents a Philips Hue room/zone
type Room struct {
	// Unique identifier from the bridge
	ID string
	// User-friendly name
	Name string
	// Lights in this room
	Lights []*Light
	// GroupedLight service ID for room-level control
	GroupedLightID string
	// Device IDs that belong to this room
	DeviceIDs []string
	// Calculated state: all lights are on
	AllOn bool
	// Calculated state: at least one light is on
	AnyOn bool
}

// UpdateState recalculates AllOn and AnyOn based on light states
func (r *Room) UpdateState() {
	if len(r.Lights) == 0 {
		r.AllOn = false
		r.AnyOn = false
		return
	}

	r.AllOn = true
	r.AnyOn = false

	for _, light := range r.Lights {
		if light.On {
			r.AnyOn = true
		} else {
			r.AllOn = false
		}
	}
}

// LightByID finds a light in this room by ID
func (r *Room) LightByID(id string) *Light {
	for _, light := range r.Lights {
		if light.ID == id {
			return light
		}
	}
	return nil
}

// AverageBrightness returns the average brightness of all on lights
func (r *Room) AverageBrightness() int {
	if len(r.Lights) == 0 {
		return 0
	}

	var total int
	var count int
	for _, light := range r.Lights {
		if light.On {
			total += light.BrightnessPct()
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / count
}

// ReachableLights returns only the lights that are reachable
func (r *Room) ReachableLights() []*Light {
	var reachable []*Light
	for _, light := range r.Lights {
		if light.Reachable {
			reachable = append(reachable, light)
		}
	}
	return reachable
}
