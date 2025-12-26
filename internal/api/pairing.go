package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrLinkButtonNotPressed = errors.New("link button not pressed")
	ErrPairingTimeout       = errors.New("pairing timeout - link button was not pressed")
)

// pairingRequest is the body sent to create an app key
type pairingRequest struct {
	DeviceType        string `json:"devicetype"`
	GenerateClientKey bool   `json:"generateclientkey,omitempty"`
}

// pairingResponse represents a response from the pairing endpoint
type pairingResponse struct {
	Success *struct {
		Username  string `json:"username"`
		ClientKey string `json:"clientkey,omitempty"`
	} `json:"success,omitempty"`
	Error *struct {
		Type        int    `json:"type"`
		Address     string `json:"address"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

// CreateAppKey attempts to create an application key on the bridge
// The user must press the link button on the bridge within the timeout
func CreateAppKey(ctx context.Context, host string, appName string, timeout time.Duration) (string, error) {
	// Create HTTP client that accepts self-signed certs (Hue bridges use self-signed)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := fmt.Sprintf("https://%s/api", host)

	body := pairingRequest{
		DeviceType:        appName,
		GenerateClientKey: true,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(timeout)
	retryInterval := time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// Network error, wait and retry
			time.Sleep(retryInterval)
			continue
		}

		var responses []pairingResponse
		if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("failed to decode pairing response: %w", err)
		}
		resp.Body.Close()

		if len(responses) == 0 {
			time.Sleep(retryInterval)
			continue
		}

		response := responses[0]

		if response.Success != nil {
			return response.Success.Username, nil
		}

		if response.Error != nil {
			// Error type 101 = link button not pressed
			if response.Error.Type == 101 {
				time.Sleep(retryInterval)
				continue
			}
			return "", fmt.Errorf("pairing error: %s", response.Error.Description)
		}

		time.Sleep(retryInterval)
	}

	return "", ErrPairingTimeout
}

// GetBridgeID retrieves the bridge ID from the config endpoint
func GetBridgeID(ctx context.Context, host string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := fmt.Sprintf("https://%s/api/0/config", host)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get bridge config: %w", err)
	}
	defer resp.Body.Close()

	var config struct {
		BridgeID string `json:"bridgeid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return "", fmt.Errorf("failed to decode bridge config: %w", err)
	}

	return config.BridgeID, nil
}
