package models

// Scene represents a Philips Hue scene
type Scene struct {
	// Unique identifier from the bridge
	ID string
	// User-friendly name
	Name string
	// ID of the room/group this scene belongs to
	RoomID string
	// Name of the room (for display purposes)
	RoomName string
	// Whether this is a dynamic scene
	IsDynamic bool
}

// ScenesByRoom groups scenes by their room ID
func ScenesByRoom(scenes []*Scene) map[string][]*Scene {
	grouped := make(map[string][]*Scene)
	for _, scene := range scenes {
		grouped[scene.RoomID] = append(grouped[scene.RoomID], scene)
	}
	return grouped
}
