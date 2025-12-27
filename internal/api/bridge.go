package api

import (
	"context"

	"github.com/angristan/hue-tui/internal/models"
)

// BridgeClient defines the interface for interacting with a Hue bridge.
// This abstraction allows for both real bridge connections and demo mode.
type BridgeClient interface {
	// FetchAll retrieves all rooms and scenes from the bridge
	FetchAll(ctx context.Context) ([]*models.Room, []*models.Scene, error)

	// Light control methods
	SetLightOn(ctx context.Context, lightID string, on bool) error
	SetLightBrightness(ctx context.Context, lightID string, brightness int) error
	SetLightColorTemp(ctx context.Context, lightID string, mirek int) error
	SetLightColorXY(ctx context.Context, lightID string, x, y float64) error
	SetLightColorHS(ctx context.Context, lightID string, hue uint16, sat uint8) error

	// Group control
	SetGroupedLightOn(ctx context.Context, groupedLightID string, on bool) error

	// Scene control
	ActivateScene(ctx context.Context, sceneID string) error

	// Metadata
	Host() string
	BridgeID() string
}

// Compile-time check that HueBridge implements BridgeClient
var _ BridgeClient = (*HueBridge)(nil)
