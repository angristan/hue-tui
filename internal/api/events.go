package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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
	ID   string
	On   *bool
	Brightness *float64
	ColorTemp  *int
	ColorXY    *struct {
		X, Y float64
	}
}

// EventHandler is called when an event is received
type EventHandler func(events []Event)

// EventSubscription manages a WebSocket connection to the bridge for events
type EventSubscription struct {
	bridge  *HueBridge
	handler EventHandler
	conn    *websocket.Conn
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
func (s *EventSubscription) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)

	if s.conn != nil {
		s.conn.Close()
	}
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

		// Connection lost, reconnect
		s.mu.Lock()
		if s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
		s.mu.Unlock()
	}
}

// connect establishes the WebSocket connection
func (s *EventSubscription) connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		HandshakeTimeout: 10 * time.Second,
	}

	url := fmt.Sprintf("wss://%s/eventstream/clip/v2", s.bridge.host)

	header := http.Header{}
	header.Set("hue-application-key", s.bridge.appKey)

	conn, _, err := dialer.DialContext(ctx, url, header)
	if err != nil {
		return fmt.Errorf("failed to connect to event stream: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	return nil
}

// readLoop reads events from the WebSocket
func (s *EventSubscription) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
		}

		s.mu.Lock()
		conn := s.conn
		s.mu.Unlock()

		if conn == nil {
			return
		}

		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		events := s.parseMessage(message)
		if len(events) > 0 {
			s.batchEvents(events)
		}
	}
}

// parseMessage parses a WebSocket message into events
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
		ID      string `json:"id"`
		On      *struct {
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
