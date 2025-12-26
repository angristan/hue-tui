package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "hue-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override config directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Create a config
	cfg := &Config{
		Bridges: []BridgeConfig{
			{
				Host:     "192.168.1.100",
				Username: "test-user-key",
				BridgeID: "001788FFFE123456",
			},
		},
		LastBridgeID: "001788FFFE123456",
	}

	// Save it
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "hue-cli", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify
	if len(loaded.Bridges) != 1 {
		t.Errorf("Expected 1 bridge, got %d", len(loaded.Bridges))
	}
	if loaded.Bridges[0].Host != "192.168.1.100" {
		t.Errorf("Expected host 192.168.1.100, got %s", loaded.Bridges[0].Host)
	}
	if loaded.LastBridgeID != "001788FFFE123456" {
		t.Errorf("Expected LastBridgeID 001788FFFE123456, got %s", loaded.LastBridgeID)
	}
}

func TestConfigAddBridge(t *testing.T) {
	cfg := &Config{}

	// Add first bridge
	cfg.AddBridge(BridgeConfig{
		Host:     "192.168.1.100",
		Username: "key1",
		BridgeID: "bridge1",
	})

	if len(cfg.Bridges) != 1 {
		t.Errorf("Expected 1 bridge, got %d", len(cfg.Bridges))
	}

	// Add second bridge
	cfg.AddBridge(BridgeConfig{
		Host:     "192.168.1.101",
		Username: "key2",
		BridgeID: "bridge2",
	})

	if len(cfg.Bridges) != 2 {
		t.Errorf("Expected 2 bridges, got %d", len(cfg.Bridges))
	}

	// Update existing bridge
	cfg.AddBridge(BridgeConfig{
		Host:     "192.168.1.200", // New IP
		Username: "key1-updated",
		BridgeID: "bridge1", // Same ID
	})

	if len(cfg.Bridges) != 2 {
		t.Errorf("Expected 2 bridges after update, got %d", len(cfg.Bridges))
	}

	// Verify update
	bridge, err := cfg.GetBridge("bridge1")
	if err != nil {
		t.Fatalf("Failed to get bridge: %v", err)
	}
	if bridge.Host != "192.168.1.200" {
		t.Errorf("Expected updated host 192.168.1.200, got %s", bridge.Host)
	}
}

func TestConfigGetBridge(t *testing.T) {
	cfg := &Config{
		Bridges: []BridgeConfig{
			{Host: "192.168.1.100", Username: "key1", BridgeID: "bridge1"},
			{Host: "192.168.1.101", Username: "key2", BridgeID: "bridge2"},
		},
	}

	// Get existing bridge
	bridge, err := cfg.GetBridge("bridge2")
	if err != nil {
		t.Fatalf("Failed to get existing bridge: %v", err)
	}
	if bridge.Host != "192.168.1.101" {
		t.Errorf("Expected host 192.168.1.101, got %s", bridge.Host)
	}

	// Get non-existing bridge
	_, err = cfg.GetBridge("nonexistent")
	if err != ErrBridgeNotFound {
		t.Errorf("Expected ErrBridgeNotFound, got %v", err)
	}
}

func TestConfigGetLastBridge(t *testing.T) {
	// Empty config
	cfg := &Config{}
	_, err := cfg.GetLastBridge()
	if err != ErrNoBridges {
		t.Errorf("Expected ErrNoBridges for empty config, got %v", err)
	}

	// Config with bridges but no last
	cfg = &Config{
		Bridges: []BridgeConfig{
			{Host: "192.168.1.100", Username: "key1", BridgeID: "bridge1"},
		},
	}
	bridge, err := cfg.GetLastBridge()
	if err != nil {
		t.Fatalf("Failed to get last bridge: %v", err)
	}
	if bridge.BridgeID != "bridge1" {
		t.Errorf("Expected bridge1, got %s", bridge.BridgeID)
	}

	// Config with last bridge ID
	cfg.LastBridgeID = "bridge1"
	cfg.Bridges = append(cfg.Bridges, BridgeConfig{
		Host: "192.168.1.101", Username: "key2", BridgeID: "bridge2",
	})

	bridge, err = cfg.GetLastBridge()
	if err != nil {
		t.Fatalf("Failed to get last bridge: %v", err)
	}
	if bridge.BridgeID != "bridge1" {
		t.Errorf("Expected last bridge bridge1, got %s", bridge.BridgeID)
	}
}

func TestConfigRemoveBridge(t *testing.T) {
	cfg := &Config{
		Bridges: []BridgeConfig{
			{Host: "192.168.1.100", Username: "key1", BridgeID: "bridge1"},
			{Host: "192.168.1.101", Username: "key2", BridgeID: "bridge2"},
		},
	}

	cfg.RemoveBridge("bridge1")

	if len(cfg.Bridges) != 1 {
		t.Errorf("Expected 1 bridge after removal, got %d", len(cfg.Bridges))
	}
	if cfg.Bridges[0].BridgeID != "bridge2" {
		t.Errorf("Expected remaining bridge to be bridge2")
	}

	// Remove non-existent (should not panic)
	cfg.RemoveBridge("nonexistent")
	if len(cfg.Bridges) != 1 {
		t.Errorf("Expected 1 bridge, got %d", len(cfg.Bridges))
	}
}

func TestConfigHasBridges(t *testing.T) {
	cfg := &Config{}
	if cfg.HasBridges() {
		t.Error("Expected HasBridges() to be false for empty config")
	}

	cfg.Bridges = []BridgeConfig{{Host: "test", Username: "test", BridgeID: "test"}}
	if !cfg.HasBridges() {
		t.Error("Expected HasBridges() to be true")
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "hue-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override config directory
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Load from non-existent file should return empty config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error for non-existent config, got %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if len(cfg.Bridges) != 0 {
		t.Errorf("Expected empty bridges, got %d", len(cfg.Bridges))
	}
}
