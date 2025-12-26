package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"

	"github.com/angristan/hue-tui/internal/models"
)

// HueBridge represents a connection to a Philips Hue bridge
type HueBridge struct {
	host     string
	appKey   string
	bridgeID string
	client   *http.Client

	// Device name cache for resolving light owners
	deviceNames map[string]string
	deviceMu    sync.RWMutex
}

// NewHueBridge creates a new bridge client
func NewHueBridge(host, appKey, bridgeID string) *HueBridge {
	return &HueBridge{
		host:        host,
		appKey:      appKey,
		bridgeID:    bridgeID,
		deviceNames: make(map[string]string),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// Host returns the bridge host
func (b *HueBridge) Host() string {
	return b.host
}

// BridgeID returns the bridge identifier
func (b *HueBridge) BridgeID() string {
	return b.bridgeID
}

// doRequest performs an authenticated API request
func (b *HueBridge) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("https://%s%s", b.host, path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("hue-application-key", b.appKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return b.client.Do(req)
}

// apiResponse wraps the V2 API response format
type apiResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Description string `json:"description"`
	} `json:"errors"`
}

// GetLights retrieves all lights from the bridge
func (b *HueBridge) GetLights(ctx context.Context) (lights []*models.Light, err error) {
	resp, err := b.doRequest(ctx, "GET", "/clip/v2/resource/light", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get lights: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode lights response: %w", err)
	}

	if len(apiResp.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Description)
	}

	var rawLights []lightResource
	if err := json.Unmarshal(apiResp.Data, &rawLights); err != nil {
		return nil, fmt.Errorf("failed to parse lights: %w", err)
	}

	result := make([]*models.Light, len(rawLights))
	for i, raw := range rawLights {
		result[i] = raw.toModel()
	}

	return result, nil
}

// lightResource represents the V2 API light resource
type lightResource struct {
	ID       string `json:"id"`
	Metadata struct {
		Name      string `json:"name"`
		Archetype string `json:"archetype"`
	} `json:"metadata"`
	On struct {
		On bool `json:"on"`
	} `json:"on"`
	Dimming *struct {
		Brightness float64 `json:"brightness"`
	} `json:"dimming"`
	ColorTemperature *struct {
		Mirek       *int `json:"mirek"`
		MirekValid  bool `json:"mirek_valid"`
		MirekSchema struct {
			MirekMinimum int `json:"mirek_minimum"`
			MirekMaximum int `json:"mirek_maximum"`
		} `json:"mirek_schema"`
	} `json:"color_temperature"`
	Color *struct {
		XY struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"xy"`
		Gamut *struct {
			Red   struct{ X, Y float64 } `json:"red"`
			Green struct{ X, Y float64 } `json:"green"`
			Blue  struct{ X, Y float64 } `json:"blue"`
		} `json:"gamut"`
	} `json:"color"`
	Owner struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"owner"`
}

func (r *lightResource) toModel() *models.Light {
	light := &models.Light{
		ID:                r.ID,
		Name:              r.Metadata.Name,
		On:                r.On.On,
		Reachable:         true, // V2 API doesn't have this directly
		DeviceID:          r.Owner.Rid,
		SupportsColor:     r.Color != nil,
		SupportsColorTemp: r.ColorTemperature != nil,
	}

	// Brightness
	if r.Dimming != nil {
		light.Brightness = uint8(r.Dimming.Brightness / 100.0 * 254)
	}

	// Color
	if r.Color != nil {
		brightness := light.Brightness
		if brightness == 0 {
			brightness = 254
		}
		light.Color = models.NewColorFromXY(r.Color.XY.X, r.Color.XY.Y, brightness)
	} else if r.ColorTemperature != nil && r.ColorTemperature.Mirek != nil {
		brightness := light.Brightness
		if brightness == 0 {
			brightness = 254
		}
		light.Color = models.NewColorFromMirek(uint16(*r.ColorTemperature.Mirek), brightness)
	}

	return light
}

// GetRooms retrieves all rooms from the bridge
func (b *HueBridge) GetRooms(ctx context.Context) (rooms []*models.Room, err error) {
	resp, err := b.doRequest(ctx, "GET", "/clip/v2/resource/room", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode rooms response: %w", err)
	}

	if len(apiResp.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Description)
	}

	var rawRooms []roomResource
	if err := json.Unmarshal(apiResp.Data, &rawRooms); err != nil {
		return nil, fmt.Errorf("failed to parse rooms: %w", err)
	}

	result := make([]*models.Room, len(rawRooms))
	for i, raw := range rawRooms {
		result[i] = raw.toModel()
	}

	return result, nil
}

// roomResource represents the V2 API room resource
type roomResource struct {
	ID       string `json:"id"`
	Metadata struct {
		Name      string `json:"name"`
		Archetype string `json:"archetype"`
	} `json:"metadata"`
	Children []struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"children"`
	Services []struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"services"`
}

func (r *roomResource) toModel() *models.Room {
	room := &models.Room{
		ID:   r.ID,
		Name: r.Metadata.Name,
	}

	// Find grouped_light service for room-level control
	for _, svc := range r.Services {
		if svc.Rtype == "grouped_light" {
			room.GroupedLightID = svc.Rid
			break
		}
	}

	// Collect device IDs from children
	for _, child := range r.Children {
		if child.Rtype == "device" {
			room.DeviceIDs = append(room.DeviceIDs, child.Rid)
		}
	}

	return room
}

// GetScenes retrieves all scenes from the bridge
func (b *HueBridge) GetScenes(ctx context.Context) (scenes []*models.Scene, err error) {
	resp, err := b.doRequest(ctx, "GET", "/clip/v2/resource/scene", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get scenes: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode scenes response: %w", err)
	}

	if len(apiResp.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Description)
	}

	var rawScenes []sceneResource
	if err := json.Unmarshal(apiResp.Data, &rawScenes); err != nil {
		return nil, fmt.Errorf("failed to parse scenes: %w", err)
	}

	result := make([]*models.Scene, len(rawScenes))
	for i, raw := range rawScenes {
		result[i] = raw.toModel()
	}

	return result, nil
}

// sceneResource represents the V2 API scene resource
type sceneResource struct {
	ID       string `json:"id"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Group struct {
		Rid   string `json:"rid"`
		Rtype string `json:"rtype"`
	} `json:"group"`
	Speed     float64 `json:"speed"`
	AutoDynac bool    `json:"auto_dynamic"`
}

func (r *sceneResource) toModel() *models.Scene {
	return &models.Scene{
		ID:        r.ID,
		Name:      r.Metadata.Name,
		RoomID:    r.Group.Rid,
		IsDynamic: r.AutoDynac,
	}
}

// GetDevices retrieves all devices and caches their names
func (b *HueBridge) GetDevices(ctx context.Context) (err error) {
	resp, err := b.doRequest(ctx, "GET", "/clip/v2/resource/device", nil)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode devices response: %w", err)
	}

	var devices []struct {
		ID       string `json:"id"`
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(apiResp.Data, &devices); err != nil {
		return fmt.Errorf("failed to parse devices: %w", err)
	}

	b.deviceMu.Lock()
	for _, d := range devices {
		b.deviceNames[d.ID] = d.Metadata.Name
	}
	b.deviceMu.Unlock()

	return nil
}

// SetLightOn turns a light on or off
func (b *HueBridge) SetLightOn(ctx context.Context, lightID string, on bool) error {
	body := fmt.Sprintf(`{"on":{"on":%t}}`, on)
	return b.setLightState(ctx, lightID, body)
}

// SetLightBrightness sets a light's brightness (0-100)
func (b *HueBridge) SetLightBrightness(ctx context.Context, lightID string, brightness int) error {
	if brightness < 0 {
		brightness = 0
	}
	if brightness > 100 {
		brightness = 100
	}
	body := fmt.Sprintf(`{"dimming":{"brightness":%d}}`, brightness)
	return b.setLightState(ctx, lightID, body)
}

// SetLightColorTemp sets a light's color temperature in mirek (153-500)
func (b *HueBridge) SetLightColorTemp(ctx context.Context, lightID string, mirek int) error {
	if mirek < 153 {
		mirek = 153
	}
	if mirek > 500 {
		mirek = 500
	}
	body := fmt.Sprintf(`{"color_temperature":{"mirek":%d}}`, mirek)
	return b.setLightState(ctx, lightID, body)
}

// SetLightColorXY sets a light's color using XY coordinates
func (b *HueBridge) SetLightColorXY(ctx context.Context, lightID string, x, y float64) error {
	body := fmt.Sprintf(`{"color":{"xy":{"x":%.4f,"y":%.4f}}}`, x, y)
	return b.setLightState(ctx, lightID, body)
}

// HSToXY converts Hue/Saturation values to XY color space coordinates.
// hue is in range 0-65535, sat is in range 0-254.
// Returns x, y coordinates in CIE 1931 color space.
func HSToXY(hue uint16, sat uint8) (x, y float64) {
	h := float64(hue) / 65535.0 * 360.0
	s := float64(sat) / 254.0

	// HSV to RGB (with V=1 for max brightness)
	c := s
	xx := c * (1 - abs64(mod64(h/60.0, 2)-1))
	m := 1.0 - c

	var r, g, bl float64
	switch int(h/60.0) % 6 {
	case 0:
		r, g, bl = c, xx, 0
	case 1:
		r, g, bl = xx, c, 0
	case 2:
		r, g, bl = 0, c, xx
	case 3:
		r, g, bl = 0, xx, c
	case 4:
		r, g, bl = xx, 0, c
	case 5:
		r, g, bl = c, 0, xx
	}
	r, g, bl = r+m, g+m, bl+m

	// Apply gamma correction
	r = applyGammaForXY(r)
	g = applyGammaForXY(g)
	bl = applyGammaForXY(bl)

	// RGB to XYZ
	X := r*0.664511 + g*0.154324 + bl*0.162028
	Y := r*0.283881 + g*0.668433 + bl*0.047685
	Z := r*0.000088 + g*0.072310 + bl*0.986039

	// XYZ to xy
	sum := X + Y + Z
	if sum == 0 {
		sum = 1
	}
	return X / sum, Y / sum
}

func (b *HueBridge) SetLightColorHS(ctx context.Context, lightID string, hue uint16, sat uint8) error {
	// Convert to XY for the Hue API (V2 API uses XY)
	xyX, xyY := HSToXY(hue, sat)

	body := fmt.Sprintf(`{"color":{"xy":{"x":%.4f,"y":%.4f}}}`, xyX, xyY)
	return b.setLightState(ctx, lightID, body)
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func mod64(a, b float64) float64 {
	return a - b*float64(int(a/b))
}

func applyGammaForXY(value float64) float64 {
	if value > 0.04045 {
		return pow((value+0.055)/1.055, 2.4)
	}
	return value / 12.92
}

func pow(base, exp float64) float64 {
	// Simple power function using math
	return math.Pow(base, exp)
}

// setLightState sends a PUT request to update light state
func (b *HueBridge) setLightState(ctx context.Context, lightID, bodyStr string) (err error) {
	path := fmt.Sprintf("/clip/v2/resource/light/%s", lightID)
	resp, err := b.doRequest(ctx, "PUT", path, strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("failed to set light state: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SetGroupedLightOn turns all lights in a group on or off
func (b *HueBridge) SetGroupedLightOn(ctx context.Context, groupedLightID string, on bool) (err error) {
	body := fmt.Sprintf(`{"on":{"on":%t}}`, on)
	path := fmt.Sprintf("/clip/v2/resource/grouped_light/%s", groupedLightID)
	resp, err := b.doRequest(ctx, "PUT", path, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to set grouped light state: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ActivateScene activates a scene
func (b *HueBridge) ActivateScene(ctx context.Context, sceneID string) (err error) {
	body := `{"recall":{"action":"active"}}`
	path := fmt.Sprintf("/clip/v2/resource/scene/%s", sceneID)
	resp, err := b.doRequest(ctx, "PUT", path, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to activate scene: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// AssignLightsToRooms assigns lights to rooms based on device ownership
func (b *HueBridge) AssignLightsToRooms(lights []*models.Light, rooms []*models.Room) []*models.Room {
	// Build device to room mapping from room.DeviceIDs
	deviceToRoom := make(map[string]*models.Room)
	for _, room := range rooms {
		room.Lights = nil // Clear existing lights
		for _, deviceID := range room.DeviceIDs {
			deviceToRoom[deviceID] = room
		}
	}

	// Create "Other Lights" room for ungrouped lights
	otherRoom := &models.Room{
		ID:   "other",
		Name: "Other Lights",
	}

	// Assign lights to rooms based on device ID
	for _, light := range lights {
		if room, ok := deviceToRoom[light.DeviceID]; ok {
			room.Lights = append(room.Lights, light)
		} else {
			otherRoom.Lights = append(otherRoom.Lights, light)
		}
	}

	// Update room states and filter empty rooms
	result := make([]*models.Room, 0, len(rooms)+1)
	for _, room := range rooms {
		if len(room.Lights) > 0 {
			room.UpdateState()
			result = append(result, room)
		}
	}

	// Add other room if it has lights
	if len(otherRoom.Lights) > 0 {
		otherRoom.UpdateState()
		result = append(result, otherRoom)
	}

	return result
}

// FetchAll retrieves all resources from the bridge
func (b *HueBridge) FetchAll(ctx context.Context) ([]*models.Room, []*models.Scene, error) {
	// Fetch rooms (includes device IDs in children)
	rooms, err := b.GetRooms(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch rooms: %w", err)
	}

	// Build room lookup for scene assignment
	roomByID := make(map[string]*models.Room)
	for _, room := range rooms {
		roomByID[room.ID] = room
	}

	// Fetch lights
	lights, err := b.GetLights(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch lights: %w", err)
	}

	// Fetch devices to map light ID -> device ID
	resp, err := b.doRequest(ctx, "GET", "/clip/v2/resource/device", nil)
	if err == nil {
		func() {
			defer func() {
				_ = resp.Body.Close() // Error ignored: optional device fetch
			}()

			var apiResp apiResponse
			if json.NewDecoder(resp.Body).Decode(&apiResp) == nil {
				var devices []struct {
					ID       string `json:"id"`
					Services []struct {
						Rid   string `json:"rid"`
						Rtype string `json:"rtype"`
					} `json:"services"`
				}
				if json.Unmarshal(apiResp.Data, &devices) == nil {
					// Map light ID to device ID
					for _, device := range devices {
						for _, svc := range device.Services {
							if svc.Rtype == "light" {
								// Find the light and set its device ID
								for _, light := range lights {
									if light.ID == svc.Rid {
										light.DeviceID = device.ID
										break
									}
								}
							}
						}
					}
				}
			}
		}()
	}

	// Assign lights to rooms using device IDs
	rooms = b.AssignLightsToRooms(lights, rooms)

	// Fetch scenes
	scenes, err := b.GetScenes(ctx)
	if err != nil {
		return rooms, nil, fmt.Errorf("failed to fetch scenes: %w", err)
	}

	// Add room names to scenes
	for _, scene := range scenes {
		if room, ok := roomByID[scene.RoomID]; ok {
			scene.RoomName = room.Name
		}
	}

	return rooms, scenes, nil
}
