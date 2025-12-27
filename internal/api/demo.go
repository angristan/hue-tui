package api

import (
	"context"
	"sync"
	"time"

	"github.com/angristan/hue-tui/internal/models"
)

// DemoBridge implements BridgeClient for demo mode without a real Hue bridge.
// All state changes are maintained in memory.
type DemoBridge struct {
	rooms  []*models.Room
	scenes []*models.Scene
	lights map[string]*models.Light // ID -> Light for quick lookup
	mu     sync.RWMutex
}

// NewDemoBridge creates a demo bridge with sample data
func NewDemoBridge() *DemoBridge {
	d := &DemoBridge{
		lights: make(map[string]*models.Light),
	}
	d.initializeDemoData()
	// Verify data was initialized (will panic if not, for debugging)
	if len(d.rooms) == 0 {
		panic("DemoBridge: initializeDemoData failed to create rooms")
	}
	return d
}

// Host returns the demo bridge host
func (d *DemoBridge) Host() string {
	return "demo-bridge.local"
}

// BridgeID returns the demo bridge identifier
func (d *DemoBridge) BridgeID() string {
	return "demo-bridge-001"
}

// FetchAll returns the demo rooms and scenes
func (d *DemoBridge) FetchAll(ctx context.Context) ([]*models.Room, []*models.Scene, error) {
	// Simulate network delay for realistic demo experience
	time.Sleep(1 * time.Second)

	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return copies to avoid external modification
	rooms := make([]*models.Room, len(d.rooms))
	for i, room := range d.rooms {
		rooms[i] = room
	}

	scenes := make([]*models.Scene, len(d.scenes))
	for i, scene := range d.scenes {
		scenes[i] = scene
	}

	return rooms, scenes, nil
}

// SetLightOn turns a demo light on or off
func (d *DemoBridge) SetLightOn(ctx context.Context, lightID string, on bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if light, ok := d.lights[lightID]; ok {
		light.On = on
		d.updateRoomStates()
	}
	return nil
}

// SetLightBrightness sets a demo light's brightness (0-100)
func (d *DemoBridge) SetLightBrightness(ctx context.Context, lightID string, brightness int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if light, ok := d.lights[lightID]; ok {
		light.SetBrightnessPct(brightness)
		if light.Color != nil {
			light.Color.Brightness = light.Brightness
			light.Color.InvalidateCache()
		}
	}
	return nil
}

// SetLightColorTemp sets a demo light's color temperature in mirek (153-500)
func (d *DemoBridge) SetLightColorTemp(ctx context.Context, lightID string, mirek int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if light, ok := d.lights[lightID]; ok && light.Color != nil {
		if mirek < 153 {
			mirek = 153
		}
		if mirek > 500 {
			mirek = 500
		}
		light.Color.Mirek = uint16(mirek)
		light.Color.Mode = models.ColorModeColorTemp
		light.Color.InvalidateCache()
	}
	return nil
}

// SetLightColorXY sets a demo light's color using XY coordinates
func (d *DemoBridge) SetLightColorXY(ctx context.Context, lightID string, x, y float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if light, ok := d.lights[lightID]; ok && light.Color != nil {
		light.Color.X = x
		light.Color.Y = y
		light.Color.Mode = models.ColorModeXY
		light.Color.InvalidateCache()
	}
	return nil
}

// SetLightColorHS sets a demo light's color using Hue/Saturation
func (d *DemoBridge) SetLightColorHS(ctx context.Context, lightID string, hue uint16, sat uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if light, ok := d.lights[lightID]; ok && light.Color != nil {
		// Convert to XY for consistency
		x, y := HSToXY(hue, sat)
		light.Color.X = x
		light.Color.Y = y
		light.Color.Hue = hue
		light.Color.Saturation = sat
		light.Color.Mode = models.ColorModeXY
		light.Color.InvalidateCache()
	}
	return nil
}

// SetGroupedLightOn turns all lights in a demo group on or off
func (d *DemoBridge) SetGroupedLightOn(ctx context.Context, groupedLightID string, on bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Find room by grouped light ID and update all lights
	for _, room := range d.rooms {
		if room.GroupedLightID == groupedLightID {
			for _, light := range room.Lights {
				light.On = on
			}
			room.UpdateState()
			break
		}
	}
	return nil
}

// ActivateScene activates a demo scene with preset light states
func (d *DemoBridge) ActivateScene(ctx context.Context, sceneID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	preset, ok := demoScenePresets[sceneID]
	if !ok {
		return nil
	}

	for lightID, state := range preset {
		if light, ok := d.lights[lightID]; ok {
			light.On = state.On
			if state.Brightness > 0 {
				light.Brightness = state.Brightness
			}
			if light.Color != nil {
				if state.Mirek > 0 {
					light.Color.Mirek = state.Mirek
					light.Color.Mode = models.ColorModeColorTemp
				} else if state.X > 0 || state.Y > 0 {
					light.Color.X = state.X
					light.Color.Y = state.Y
					light.Color.Mode = models.ColorModeXY
				}
				light.Color.Brightness = light.Brightness
				light.Color.InvalidateCache()
			}
		}
	}

	d.updateRoomStates()
	return nil
}

// updateRoomStates recalculates the state for all rooms
func (d *DemoBridge) updateRoomStates() {
	for _, room := range d.rooms {
		room.UpdateState()
	}
}

// lightState represents preset state for a light in a scene
type lightState struct {
	On         bool
	Brightness uint8
	Mirek      uint16
	X, Y       float64
}

// Demo scene presets
var demoScenePresets = map[string]map[string]lightState{
	// Living Room scenes
	"scene-movie-night": {
		"light-lr-ceiling":  {On: false, Brightness: 0},
		"light-lr-floor":    {On: true, Brightness: 64, Mirek: 500},   // Dim warm
		"light-lr-tv-bias":  {On: true, Brightness: 76, X: 0.15, Y: 0.06}, // Blue
		"light-lr-accent":   {On: true, Brightness: 38, X: 0.55, Y: 0.41}, // Purple
	},
	"scene-energize": {
		"light-lr-ceiling":  {On: true, Brightness: 254, Mirek: 200},  // Cool bright
		"light-lr-floor":    {On: true, Brightness: 254, Mirek: 200},
		"light-lr-tv-bias":  {On: true, Brightness: 254, X: 0.31, Y: 0.32}, // White
		"light-lr-accent":   {On: true, Brightness: 254, X: 0.31, Y: 0.32},
	},
	"scene-relax": {
		"light-lr-ceiling":  {On: true, Brightness: 150, Mirek: 400},  // Warm
		"light-lr-floor":    {On: true, Brightness: 127, Mirek: 450},
		"light-lr-tv-bias":  {On: false, Brightness: 0},
		"light-lr-accent":   {On: true, Brightness: 76, X: 0.56, Y: 0.35}, // Soft orange
	},
	// Bedroom scenes
	"scene-sleep": {
		"light-br-left":    {On: true, Brightness: 25, Mirek: 500},   // Very dim warm
		"light-br-right":   {On: false, Brightness: 0},
		"light-br-ceiling": {On: false, Brightness: 0},
	},
	"scene-reading": {
		"light-br-left":    {On: true, Brightness: 200, Mirek: 300},
		"light-br-right":   {On: true, Brightness: 200, Mirek: 300},
		"light-br-ceiling": {On: false, Brightness: 0},
	},
	// Kitchen scenes
	"scene-cooking": {
		"light-kt-main":    {On: true, Brightness: 254, Mirek: 250},  // Cool bright
		"light-kt-cabinet": {On: true, Brightness: 254, Mirek: 250},
	},
	"scene-morning": {
		"light-kt-main":    {On: true, Brightness: 180, Mirek: 350},  // Warm bright
		"light-kt-cabinet": {On: true, Brightness: 127, Mirek: 400},
	},
	// Office scenes
	"scene-focus": {
		"light-of-desk":     {On: true, Brightness: 254, Mirek: 250},  // Cool bright
		"light-of-monitor":  {On: true, Brightness: 150, Mirek: 200},
		"light-of-bookshelf": {On: false, Brightness: 0},
	},
}

// initializeDemoData creates the demo rooms, lights, and scenes
func (d *DemoBridge) initializeDemoData() {
	// Living Room lights
	livingRoomLights := []*models.Light{
		{
			ID:                "light-lr-ceiling",
			Name:              "Ceiling Light",
			On:                true,
			Brightness:        203, // ~80%
			SupportsColor:     true,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(326, 203), // Neutral white
		},
		{
			ID:                "light-lr-floor",
			Name:              "Floor Lamp",
			On:                true,
			Brightness:        152, // ~60%
			SupportsColor:     true,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(400, 152), // Warm
		},
		{
			ID:                "light-lr-tv-bias",
			Name:              "TV Bias Light",
			On:                true,
			Brightness:        101, // ~40%
			SupportsColor:     true,
			SupportsColorTemp: false,
			Color:             models.NewColorFromXY(0.15, 0.06, 101), // Blue
		},
		{
			ID:                "light-lr-accent",
			Name:              "Accent Strip",
			On:                false,
			Brightness:        0,
			SupportsColor:     true,
			SupportsColorTemp: false,
			Color:             models.NewColorFromXY(0.64, 0.33, 254), // Red (stored but off)
		},
	}

	// Bedroom lights
	bedroomLights := []*models.Light{
		{
			ID:                "light-br-left",
			Name:              "Bedside Left",
			On:                true,
			Brightness:        76, // ~30%
			SupportsColor:     true,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(454, 76), // Very warm
		},
		{
			ID:                "light-br-right",
			Name:              "Bedside Right",
			On:                false,
			Brightness:        0,
			SupportsColor:     true,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(400, 127),
		},
		{
			ID:                "light-br-ceiling",
			Name:              "Ceiling Light",
			On:                false,
			Brightness:        0,
			SupportsColor:     false,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(326, 254),
		},
	}

	// Kitchen lights
	kitchenLights := []*models.Light{
		{
			ID:                "light-kt-main",
			Name:              "Main Light",
			On:                true,
			Brightness:        254, // 100%
			SupportsColor:     false,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(233, 254), // Cool white
		},
		{
			ID:                "light-kt-cabinet",
			Name:              "Under Cabinet",
			On:                true,
			Brightness:        178, // ~70%
			SupportsColor:     false,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(250, 178),
		},
	}

	// Office lights
	officeLights := []*models.Light{
		{
			ID:                "light-of-desk",
			Name:              "Desk Lamp",
			On:                true,
			Brightness:        229, // ~90%
			SupportsColor:     true,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(300, 229), // Neutral
		},
		{
			ID:                "light-of-monitor",
			Name:              "Monitor Light",
			On:                true,
			Brightness:        127, // ~50%
			SupportsColor:     false,
			SupportsColorTemp: true,
			Color:             models.NewColorFromMirek(250, 127),
		},
		{
			ID:                "light-of-bookshelf",
			Name:              "Bookshelf",
			On:                true,
			Brightness:        101, // ~40%
			SupportsColor:     true,
			SupportsColorTemp: false,
			Color:             models.NewColorFromXY(0.32, 0.15, 101), // Purple
		},
	}

	// Create rooms
	d.rooms = []*models.Room{
		{
			ID:             "room-living",
			Name:           "Living Room",
			Lights:         livingRoomLights,
			GroupedLightID: "group-living",
		},
		{
			ID:             "room-bedroom",
			Name:           "Bedroom",
			Lights:         bedroomLights,
			GroupedLightID: "group-bedroom",
		},
		{
			ID:             "room-kitchen",
			Name:           "Kitchen",
			Lights:         kitchenLights,
			GroupedLightID: "group-kitchen",
		},
		{
			ID:             "room-office",
			Name:           "Office",
			Lights:         officeLights,
			GroupedLightID: "group-office",
		},
	}

	// Build light lookup map and set room IDs
	for _, room := range d.rooms {
		for _, light := range room.Lights {
			light.RoomID = room.ID
			d.lights[light.ID] = light
		}
		room.UpdateState()
	}

	// Create scenes
	d.scenes = []*models.Scene{
		// Living Room scenes
		{ID: "scene-movie-night", Name: "Movie Night", RoomID: "room-living", RoomName: "Living Room"},
		{ID: "scene-energize", Name: "Energize", RoomID: "room-living", RoomName: "Living Room"},
		{ID: "scene-relax", Name: "Relax", RoomID: "room-living", RoomName: "Living Room"},
		// Bedroom scenes
		{ID: "scene-sleep", Name: "Sleep", RoomID: "room-bedroom", RoomName: "Bedroom"},
		{ID: "scene-reading", Name: "Reading", RoomID: "room-bedroom", RoomName: "Bedroom"},
		// Kitchen scenes
		{ID: "scene-cooking", Name: "Cooking", RoomID: "room-kitchen", RoomName: "Kitchen"},
		{ID: "scene-morning", Name: "Morning", RoomID: "room-kitchen", RoomName: "Kitchen"},
		// Office scenes
		{ID: "scene-focus", Name: "Focus", RoomID: "room-office", RoomName: "Office"},
	}
}

// Compile-time check that DemoBridge implements BridgeClient
var _ BridgeClient = (*DemoBridge)(nil)
