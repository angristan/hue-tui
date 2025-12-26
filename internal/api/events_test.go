package api

import (
	"encoding/json"
	"testing"
)

func TestParseMessage_LightUpdate(t *testing.T) {
	// Sample SSE event data from Hue bridge
	message := `[{
		"creationtime": "2024-01-15T10:30:00Z",
		"id": "event-123",
		"type": "update",
		"data": [{
			"id": "light-abc-123",
			"type": "light",
			"on": {"on": true},
			"dimming": {"brightness": 75.5}
		}]
	}]`

	sub := &EventSubscription{}
	events := sub.parseMessage([]byte(message))

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != EventTypeUpdate {
		t.Errorf("Expected type 'update', got '%s'", event.Type)
	}
	if event.Resource != "light" {
		t.Errorf("Expected resource 'light', got '%s'", event.Resource)
	}
	if event.ResourceID != "light-abc-123" {
		t.Errorf("Expected resourceID 'light-abc-123', got '%s'", event.ResourceID)
	}
}

func TestParseMessage_MultipleEvents(t *testing.T) {
	message := `[{
		"creationtime": "2024-01-15T10:30:00Z",
		"id": "event-123",
		"type": "update",
		"data": [
			{"id": "light-1", "type": "light", "on": {"on": true}},
			{"id": "light-2", "type": "light", "on": {"on": false}}
		]
	}]`

	sub := &EventSubscription{}
	events := sub.parseMessage([]byte(message))

	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	if events[0].ResourceID != "light-1" {
		t.Errorf("Expected first event ID 'light-1', got '%s'", events[0].ResourceID)
	}
	if events[1].ResourceID != "light-2" {
		t.Errorf("Expected second event ID 'light-2', got '%s'", events[1].ResourceID)
	}
}

func TestParseMessage_MixedResourceTypes(t *testing.T) {
	message := `[{
		"creationtime": "2024-01-15T10:30:00Z",
		"id": "event-123",
		"type": "update",
		"data": [
			{"id": "light-1", "type": "light"},
			{"id": "room-1", "type": "room"},
			{"id": "scene-1", "type": "scene"}
		]
	}]`

	sub := &EventSubscription{}
	events := sub.parseMessage([]byte(message))

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	resources := map[string]bool{}
	for _, e := range events {
		resources[e.Resource] = true
	}

	if !resources["light"] || !resources["room"] || !resources["scene"] {
		t.Error("Expected light, room, and scene resources")
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	sub := &EventSubscription{}
	events := sub.parseMessage([]byte("invalid json"))

	if len(events) != 0 {
		t.Errorf("Expected 0 events for invalid JSON, got %d", len(events))
	}
}

func TestParseMessage_EmptyArray(t *testing.T) {
	sub := &EventSubscription{}
	events := sub.parseMessage([]byte("[]"))

	if len(events) != 0 {
		t.Errorf("Expected 0 events for empty array, got %d", len(events))
	}
}

func TestParseLightUpdate_OnOff(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "light-123",
		Resource:   "light",
		Data:       json.RawMessage(`{"id": "light-123", "on": {"on": true}}`),
	}

	update, err := ParseLightUpdate(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if update.ID != "light-123" {
		t.Errorf("Expected ID 'light-123', got '%s'", update.ID)
	}
	if update.On == nil || *update.On != true {
		t.Error("Expected On to be true")
	}
	if update.Brightness != nil {
		t.Error("Expected Brightness to be nil")
	}
}

func TestParseLightUpdate_Brightness(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "light-123",
		Resource:   "light",
		Data:       json.RawMessage(`{"id": "light-123", "dimming": {"brightness": 75.5}}`),
	}

	update, err := ParseLightUpdate(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if update.Brightness == nil {
		t.Fatal("Expected Brightness to be set")
	}
	if *update.Brightness != 75.5 {
		t.Errorf("Expected Brightness 75.5, got %f", *update.Brightness)
	}
}

func TestParseLightUpdate_ColorTemp(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "light-123",
		Resource:   "light",
		Data:       json.RawMessage(`{"id": "light-123", "color_temperature": {"mirek": 350}}`),
	}

	update, err := ParseLightUpdate(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if update.ColorTemp == nil {
		t.Fatal("Expected ColorTemp to be set")
	}
	if *update.ColorTemp != 350 {
		t.Errorf("Expected ColorTemp 350, got %d", *update.ColorTemp)
	}
}

func TestParseLightUpdate_ColorXY(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "light-123",
		Resource:   "light",
		Data:       json.RawMessage(`{"id": "light-123", "color": {"xy": {"x": 0.4573, "y": 0.41}}}`),
	}

	update, err := ParseLightUpdate(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if update.ColorXY == nil {
		t.Fatal("Expected ColorXY to be set")
	}
	if update.ColorXY.X != 0.4573 || update.ColorXY.Y != 0.41 {
		t.Errorf("Expected ColorXY {0.4573, 0.41}, got {%f, %f}", update.ColorXY.X, update.ColorXY.Y)
	}
}

func TestParseLightUpdate_AllFields(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "light-123",
		Resource:   "light",
		Data: json.RawMessage(`{
			"id": "light-123",
			"on": {"on": true},
			"dimming": {"brightness": 80},
			"color_temperature": {"mirek": 300},
			"color": {"xy": {"x": 0.5, "y": 0.5}}
		}`),
	}

	update, err := ParseLightUpdate(event)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if update.On == nil || *update.On != true {
		t.Error("Expected On to be true")
	}
	if update.Brightness == nil || *update.Brightness != 80 {
		t.Error("Expected Brightness to be 80")
	}
	if update.ColorTemp == nil || *update.ColorTemp != 300 {
		t.Error("Expected ColorTemp to be 300")
	}
	if update.ColorXY == nil || update.ColorXY.X != 0.5 || update.ColorXY.Y != 0.5 {
		t.Error("Expected ColorXY to be {0.5, 0.5}")
	}
}

func TestParseLightUpdate_NotLightEvent(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "room-123",
		Resource:   "room",
		Data:       json.RawMessage(`{"id": "room-123"}`),
	}

	_, err := ParseLightUpdate(event)
	if err == nil {
		t.Error("Expected error for non-light event")
	}
}

func TestParseLightUpdate_InvalidJSON(t *testing.T) {
	event := Event{
		Type:       EventTypeUpdate,
		ResourceID: "light-123",
		Resource:   "light",
		Data:       json.RawMessage(`invalid`),
	}

	_, err := ParseLightUpdate(event)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestEventTypes(t *testing.T) {
	if EventTypeUpdate != "update" {
		t.Errorf("Expected EventTypeUpdate to be 'update'")
	}
	if EventTypeAdd != "add" {
		t.Errorf("Expected EventTypeAdd to be 'add'")
	}
	if EventTypeDelete != "delete" {
		t.Errorf("Expected EventTypeDelete to be 'delete'")
	}
	if EventTypeError != "error" {
		t.Errorf("Expected EventTypeError to be 'error'")
	}
}
