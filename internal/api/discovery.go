package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

// DiscoveredBridge represents a Hue bridge found during discovery
type DiscoveredBridge struct {
	// IP address of the bridge
	Host string
	// Unique bridge identifier
	BridgeID string
	// Model ID (e.g., "BSB002")
	ModelID string
	// Name from mDNS or cloud
	Name string
}

// DiscoverMDNS discovers Hue bridges on the local network using mDNS
func DiscoverMDNS(ctx context.Context, timeout time.Duration) ([]DiscoveredBridge, error) {
	var bridges []DiscoveredBridge
	var mu sync.Mutex

	// Create a channel for mDNS entries
	entriesCh := make(chan *mdns.ServiceEntry, 10)

	// Start a goroutine to collect entries
	go func() {
		for entry := range entriesCh {
			bridge := DiscoveredBridge{
				Host: entry.AddrV4.String(),
				Name: entry.Name,
			}

			// Parse bridge ID from TXT records
			for _, txt := range entry.InfoFields {
				if strings.HasPrefix(txt, "bridgeid=") {
					bridge.BridgeID = strings.TrimPrefix(txt, "bridgeid=")
				}
				if strings.HasPrefix(txt, "modelid=") {
					bridge.ModelID = strings.TrimPrefix(txt, "modelid=")
				}
			}

			// Use hostname if no name
			if bridge.Name == "" && entry.Host != "" {
				bridge.Name = strings.TrimSuffix(entry.Host, ".")
			}

			mu.Lock()
			bridges = append(bridges, bridge)
			mu.Unlock()
		}
	}()

	// Create query params for Hue service
	params := mdns.DefaultParams("_hue._tcp")
	params.Entries = entriesCh
	params.Timeout = timeout
	params.DisableIPv6 = true

	// Run the query
	err := mdns.Query(params)
	close(entriesCh)

	if err != nil {
		return bridges, fmt.Errorf("mDNS query failed: %w", err)
	}

	return bridges, nil
}

// nupnpResponse represents the response from Hue cloud discovery
type nupnpResponse struct {
	ID                string `json:"id"`
	InternalIPAddress string `json:"internalipaddress"`
	Port              int    `json:"port"`
}

// DiscoverCloud discovers Hue bridges using the Philips Hue cloud service (NUPNP)
func DiscoverCloud(ctx context.Context, timeout time.Duration) (bridges []DiscoveredBridge, err error) {
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://discovery.meethue.com", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud discovery request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud discovery returned status %d", resp.StatusCode)
	}

	var results []nupnpResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([]DiscoveredBridge, len(results))
	for i, r := range results {
		result[i] = DiscoveredBridge{
			Host:     r.InternalIPAddress,
			BridgeID: r.ID,
		}
	}

	return result, nil
}

// DiscoverAll runs both mDNS and cloud discovery concurrently
// Returns results from whichever method finds bridges first, or combines results
func DiscoverAll(ctx context.Context, timeout time.Duration) ([]DiscoveredBridge, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		bridges []DiscoveredBridge
		err     error
		source  string
	}

	results := make(chan result, 2)

	// Run mDNS discovery
	go func() {
		bridges, err := DiscoverMDNS(ctx, timeout)
		results <- result{bridges: bridges, err: err, source: "mDNS"}
	}()

	// Run cloud discovery
	go func() {
		bridges, err := DiscoverCloud(ctx, timeout)
		results <- result{bridges: bridges, err: err, source: "cloud"}
	}()

	// Collect results
	var allBridges []DiscoveredBridge
	seen := make(map[string]bool)
	var lastErr error
	received := 0

	for received < 2 {
		select {
		case r := <-results:
			received++
			if r.err != nil {
				lastErr = r.err
				continue
			}
			for _, b := range r.bridges {
				key := b.Host
				if b.BridgeID != "" {
					key = b.BridgeID
				}
				if !seen[key] {
					seen[key] = true
					allBridges = append(allBridges, b)
				}
			}
		case <-ctx.Done():
			return allBridges, ctx.Err()
		}
	}

	if len(allBridges) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return allBridges, nil
}
