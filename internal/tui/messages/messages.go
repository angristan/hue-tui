package messages

import (
	"github.com/angristan/hue-tui/internal/api"
	"github.com/angristan/hue-tui/internal/models"
)

// BridgeConnectedMsg indicates successful bridge connection
type BridgeConnectedMsg struct {
	Bridge *api.HueBridge
	AppKey string
}

// DataFetchedMsg contains fetched data from the bridge
type DataFetchedMsg struct {
	Rooms  []*models.Room
	Scenes []*models.Scene
}

// ErrorMsg indicates an error occurred
type ErrorMsg struct {
	Err error
}

// ShowScenesMsg requests showing the scenes modal
type ShowScenesMsg struct {
	RoomID string // Filter scenes to this room (empty = show all)
}

// HideScenesMsg requests hiding the scenes modal
type HideScenesMsg struct{}

// SceneActivatedMsg indicates a scene was activated
type SceneActivatedMsg struct {
	SceneID string
}

// RefreshMsg requests a data refresh
type RefreshMsg struct{}

// LightUpdateMsg indicates a light state change
type LightUpdateMsg struct {
	LightID    string
	On         *bool
	Brightness *int
	ColorTemp  *int
	ColorXY    *struct{ X, Y float64 }
}
