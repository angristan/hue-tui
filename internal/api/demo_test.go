package api

import (
	"context"
	"testing"
)

func TestDemoBridgeData(t *testing.T) {
	d := NewDemoBridge()
	rooms, scenes, err := d.FetchAll(context.Background())

	if err != nil {
		t.Fatalf("FetchAll returned error: %v", err)
	}

	t.Logf("Rooms: %d, Scenes: %d", len(rooms), len(scenes))

	if len(rooms) == 0 {
		t.Error("No rooms returned")
	}

	for _, r := range rooms {
		t.Logf("  Room: %s (%d lights)", r.Name, len(r.Lights))
		if len(r.Lights) == 0 {
			t.Errorf("Room %s has no lights", r.Name)
		}
	}

	if len(scenes) == 0 {
		t.Error("No scenes returned")
	}
}
