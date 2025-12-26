package api

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var eventsDebug = os.Getenv("HUE_DEBUG") != ""
var eventsLog *log.Logger

func init() {
	if eventsDebug {
		f, err := os.OpenFile("hue-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			eventsLog = log.New(os.Stderr, "[EVENTS] ", log.LstdFlags|log.Lmicroseconds)
		} else {
			eventsLog = log.New(f, "[EVENTS] ", log.LstdFlags|log.Lmicroseconds)
		}
	}
}

func eventsDebugf(format string, args ...interface{}) {
	if eventsDebug && eventsLog != nil {
		eventsLog.Printf(format, args...)
	}
}

// EventType represents the type of event from the bridge
type EventType string

const (
	EventTypeUpdate EventType = "update"
	EventTypeAdd    EventType = "add"
	EventTypeDelete EventType = "delete"
	EventTypeError  EventType = "error"
)

// Event represents an event from the Hue bridge
type Event struct {
	Type       EventType
	ResourceID string
	Resource   string // "light", "room", "scene", etc.
	Data       json.RawMessage
}

// LightUpdateEvent contains updated light state
type LightUpdateEvent struct {
	ID         string
	On         *bool
	Brightness *float64
	ColorTemp  *int
	ColorXY    *struct {
		X, Y float64
	}
}

// EventHandler is called when an event is received
type EventHandler func(events []Event)

// EventSubscription manages an SSE connection to the bridge for events
type EventSubscription struct {
	bridge  *HueBridge
	handler EventHandler
	resp    *http.Response
	mu      sync.Mutex
	done    chan struct{}
	running bool

	// Event batching
	eventBatch   []Event
	batchMu      sync.Mutex
	batchTimer   *time.Timer
	batchTimeout time.Duration
}

// NewEventSubscription creates a new event subscription
func NewEventSubscription(bridge *HueBridge, handler EventHandler) *EventSubscription {
	return &EventSubscription{
		bridge:       bridge,
		handler:      handler,
		done:         make(chan struct{}),
		batchTimeout: 50 * time.Millisecond,
	}
}

// Start begins listening for events
func (s *EventSubscription) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	go s.run(ctx)
	return nil
}

// Stop stops the event subscription
func (s *EventSubscription) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	close(s.done)

	if s.resp != nil {
		return s.resp.Body.Close()
	}
	return nil
}

// run is the main event loop
func (s *EventSubscription) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
		}

		err := s.connect(ctx)
		if err != nil {
			eventsDebugf("Connection error: %v, reconnecting in 5s", err)
			// Wait before reconnecting
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			case <-s.done:
				return
			}
			continue
		}

		s.readLoop(ctx)

		// Connection lost, close and reconnect
		s.mu.Lock()
		if s.resp != nil {
			_ = s.resp.Body.Close()
			s.resp = nil
		}
		s.mu.Unlock()

		eventsDebugf("Connection lost, reconnecting...")
	}
}

// connect establishes the SSE connection
func (s *EventSubscription) connect(ctx context.Context) error {
	url := fmt.Sprintf("https://%s/eventstream/clip/v2", s.bridge.host)
	eventsDebugf("Connecting to SSE: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("hue-application-key", s.bridge.appKey)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 0, // No timeout for SSE
	}

	resp, err := client.Do(req)
	if err != nil {
		eventsDebugf("SSE connection failed: %v", err)
		return fmt.Errorf("failed to connect to event stream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		eventsDebugf("SSE bad status: %s", resp.Status)
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	eventsDebugf("SSE connected successfully (status: %s, content-type: %s)",
		resp.Status, resp.Header.Get("Content-Type"))

	s.mu.Lock()
	s.resp = resp
	s.mu.Unlock()

	return nil
}

// readLoop reads events from the SSE stream
func (s *EventSubscription) readLoop(ctx context.Context) {
	eventsDebugf("Starting SSE read loop")

	s.mu.Lock()
	resp := s.resp
	s.mu.Unlock()

	if resp == nil {
		eventsDebugf("Read loop: response is nil")
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for large events
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var dataBuffer strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			eventsDebugf("Read loop: context done")
			return
		case <-s.done:
			eventsDebugf("Read loop: done signal received")
			return
		default:
		}

		line := scanner.Text()

		// SSE format: lines starting with "data:" contain JSON
		// Empty line signals end of event
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			dataBuffer.WriteString(data)
		} else if line == "" && dataBuffer.Len() > 0 {
			// End of event, process the data
			eventData := dataBuffer.String()
			dataBuffer.Reset()

			eventsDebugf("Read loop: received event (%d bytes)", len(eventData))
			events := s.parseMessage([]byte(eventData))
			eventsDebugf("Read loop: parsed %d events", len(events))
			if len(events) > 0 {
				s.batchEvents(events)
			}
		}
		// Ignore other lines (id:, event:, retry:, comments starting with :)
	}

	if err := scanner.Err(); err != nil {
		eventsDebugf("Read loop: scanner error: %v", err)
	} else {
		eventsDebugf("Read loop: stream ended")
	}
}

// parseMessage parses an SSE data payload into events
func (s *EventSubscription) parseMessage(message []byte) []Event {
	var rawEvents []struct {
		CreationTime string `json:"creationtime"`
		Data         []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Owner *struct {
				Rid   string `json:"rid"`
				Rtype string `json:"rtype"`
			} `json:"owner"`
			On *struct {
				On bool `json:"on"`
			} `json:"on"`
			Dimming *struct {
				Brightness float64 `json:"brightness"`
			} `json:"dimming"`
			ColorTemperature *struct {
				Mirek int `json:"mirek"`
			} `json:"color_temperature"`
			Color *struct {
				XY struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"xy"`
			} `json:"color"`
		} `json:"data"`
		ID   string `json:"id"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(message, &rawEvents); err != nil {
		eventsDebugf("Parse error: %v (data: %s)", err, string(message[:min(200, len(message))]))
		return nil
	}

	var events []Event
	for _, rawEvent := range rawEvents {
		eventType := EventType(rawEvent.Type)
		for _, data := range rawEvent.Data {
			event := Event{
				Type:       eventType,
				ResourceID: data.ID,
				Resource:   data.Type,
			}

			// Re-marshal the data for the handler
			dataBytes, _ := json.Marshal(data)
			event.Data = dataBytes

			events = append(events, event)
		}
	}

	return events
}

// batchEvents adds events to the batch and schedules delivery
func (s *EventSubscription) batchEvents(events []Event) {
	s.batchMu.Lock()
	defer s.batchMu.Unlock()

	eventsDebugf("Batching %d events (batch size now: %d)", len(events), len(s.eventBatch)+len(events))
	s.eventBatch = append(s.eventBatch, events...)

	// Cancel existing timer and create new one
	if s.batchTimer != nil {
		s.batchTimer.Stop()
	}

	s.batchTimer = time.AfterFunc(s.batchTimeout, func() {
		s.deliverBatch()
	})
}

// deliverBatch sends the batched events to the handler
func (s *EventSubscription) deliverBatch() {
	s.batchMu.Lock()
	batch := s.eventBatch
	s.eventBatch = nil
	s.batchMu.Unlock()

	eventsDebugf("Delivering batch of %d events", len(batch))
	if len(batch) > 0 && s.handler != nil {
		s.handler(batch)
	}
}

// ParseLightUpdate parses a light update event
func ParseLightUpdate(event Event) (*LightUpdateEvent, error) {
	if event.Resource != "light" {
		return nil, fmt.Errorf("not a light event")
	}

	var data struct {
		ID string `json:"id"`
		On *struct {
			On bool `json:"on"`
		} `json:"on"`
		Dimming *struct {
			Brightness float64 `json:"brightness"`
		} `json:"dimming"`
		ColorTemperature *struct {
			Mirek int `json:"mirek"`
		} `json:"color_temperature"`
		Color *struct {
			XY struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"xy"`
		} `json:"color"`
	}

	if err := json.Unmarshal(event.Data, &data); err != nil {
		return nil, err
	}

	update := &LightUpdateEvent{
		ID: data.ID,
	}

	if data.On != nil {
		update.On = &data.On.On
	}
	if data.Dimming != nil {
		update.Brightness = &data.Dimming.Brightness
	}
	if data.ColorTemperature != nil {
		update.ColorTemp = &data.ColorTemperature.Mirek
	}
	if data.Color != nil {
		update.ColorXY = &struct{ X, Y float64 }{data.Color.XY.X, data.Color.XY.Y}
	}

	return update, nil
}
