package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// BridgeConfig stores connection details for a Hue bridge
type BridgeConfig struct {
	// IP address or hostname of the bridge
	Host string `json:"host"`
	// Application key (username) for authentication
	Username string `json:"username"`
	// Unique bridge identifier
	BridgeID string `json:"bridge_id"`
}

// Config stores all application configuration
type Config struct {
	// List of configured bridges
	Bridges []BridgeConfig `json:"bridges"`
	// ID of the last used bridge
	LastBridgeID string `json:"last_bridge_id,omitempty"`
}

var (
	ErrBridgeNotFound = errors.New("bridge not found")
	ErrNoBridges      = errors.New("no bridges configured")
)

// configDir returns the configuration directory path
func configDir() (string, error) {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "hue-cli"), nil
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hue-cli"), nil
}

// configPath returns the full path to the config file
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// AddBridge adds or updates a bridge configuration
func (c *Config) AddBridge(bridge BridgeConfig) {
	// Check if bridge already exists and update it
	for i, b := range c.Bridges {
		if b.BridgeID == bridge.BridgeID {
			c.Bridges[i] = bridge
			return
		}
	}

	// Add new bridge
	c.Bridges = append(c.Bridges, bridge)
}

// GetBridge returns the bridge configuration by ID
func (c *Config) GetBridge(bridgeID string) (*BridgeConfig, error) {
	for i := range c.Bridges {
		if c.Bridges[i].BridgeID == bridgeID {
			return &c.Bridges[i], nil
		}
	}
	return nil, ErrBridgeNotFound
}

// GetLastBridge returns the last used bridge or the first available
func (c *Config) GetLastBridge() (*BridgeConfig, error) {
	if len(c.Bridges) == 0 {
		return nil, ErrNoBridges
	}

	// Try to get the last used bridge
	if c.LastBridgeID != "" {
		bridge, err := c.GetBridge(c.LastBridgeID)
		if err == nil {
			return bridge, nil
		}
	}

	// Fall back to first bridge
	return &c.Bridges[0], nil
}

// RemoveBridge removes a bridge by ID
func (c *Config) RemoveBridge(bridgeID string) {
	for i, b := range c.Bridges {
		if b.BridgeID == bridgeID {
			c.Bridges = append(c.Bridges[:i], c.Bridges[i+1:]...)
			return
		}
	}
}

// HasBridges returns true if at least one bridge is configured
func (c *Config) HasBridges() bool {
	return len(c.Bridges) > 0
}
