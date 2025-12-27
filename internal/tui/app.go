package tui

import (
	"context"
	"log"
	"os"

	"github.com/angristan/hue-tui/internal/api"
	"github.com/angristan/hue-tui/internal/config"
	"github.com/angristan/hue-tui/internal/models"
	"github.com/angristan/hue-tui/internal/tui/messages"
	"github.com/angristan/hue-tui/internal/tui/screens"
	tea "github.com/charmbracelet/bubbletea"
)

var debugMode = os.Getenv("HUE_DEBUG") != ""
var debugLog *log.Logger

func init() {
	if debugMode {
		f, err := os.OpenFile("hue-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			debugLog = log.New(os.Stderr, "[HUE] ", log.LstdFlags|log.Lmicroseconds)
		} else {
			debugLog = log.New(f, "[HUE] ", log.LstdFlags|log.Lmicroseconds)
		}
		debugLog.Println("Debug mode enabled")
	}
}

func debugf(format string, args ...interface{}) {
	if debugMode && debugLog != nil {
		debugLog.Printf(format, args...)
	}
}

// Screen represents the current screen state
type Screen int

const (
	ScreenSetup Screen = iota
	ScreenMain
	ScreenScenes
)

// Model is the main application model
type Model struct {
	// Configuration
	config *config.Config

	// Bridge connection
	bridge   api.BridgeClient
	events   *api.EventSubscription
	demoMode bool

	// Event handling
	eventChan chan tea.Msg
	pending   *PendingTracker

	// Data
	rooms  []*models.Room
	scenes []*models.Scene

	// Current screen
	screen Screen

	// Screen models
	setupScreen  screens.SetupModel
	mainScreen   screens.MainModel
	scenesScreen screens.ScenesModel

	// Window size
	width  int
	height int

	// Error state
	err error

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewModel creates a new application model
func NewModel(cfg *config.Config, demoMode bool) Model {
	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		eventChan: make(chan tea.Msg, 100),
		pending:   NewPendingTracker(),
		demoMode:  demoMode,
	}

	// Determine initial screen
	if demoMode {
		// Demo mode: use demo bridge, go straight to main screen
		m.screen = ScreenMain
		m.bridge = api.NewDemoBridge()
	} else if cfg.HasBridges() {
		m.screen = ScreenMain
		bridgeCfg, _ := cfg.GetLastBridge()
		if bridgeCfg != nil {
			m.bridge = api.NewHueBridge(bridgeCfg.Host, bridgeCfg.Username, bridgeCfg.BridgeID)
		}
	} else {
		m.screen = ScreenSetup
	}

	// Initialize screen models
	m.setupScreen = screens.NewSetupModel()
	m.mainScreen = screens.NewMainModel(nil)
	m.scenesScreen = screens.NewScenesModel()

	return m
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	debugf("Init called, screen=%d, demoMode=%v, bridge=%v", m.screen, m.demoMode, m.bridge != nil)
	cmds := []tea.Cmd{
		tea.SetWindowTitle("Hue CLI"),
	}

	// Start with appropriate screen initialization
	switch m.screen {
	case ScreenSetup:
		debugf("Init: starting setup screen")
		cmds = append(cmds, m.setupScreen.Init())
	case ScreenMain:
		debugf("Init: starting main screen, will fetch data")
		cmds = append(cmds, m.mainScreen.Init(), m.fetchDataCmd())
	}

	return tea.Batch(cmds...)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Log all message types for debugging
	debugf("Update received message: %T", msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainScreen.SetSize(msg.Width, msg.Height)
		m.setupScreen.SetSize(msg.Width, msg.Height)
		m.scenesScreen.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		// Global key handlers
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit
		}

	case messages.BridgeConnectedMsg:
		// Bridge connection successful
		m.bridge = msg.Bridge
		// Only save config for real bridges, not demo mode
		if !m.demoMode {
			m.config.AddBridge(config.BridgeConfig{
				Host:     msg.Bridge.Host(),
				Username: msg.AppKey,
				BridgeID: msg.Bridge.BridgeID(),
			})
			m.config.LastBridgeID = msg.Bridge.BridgeID()
			if err := m.config.Save(); err != nil {
				m.err = err
			}
		}

		m.screen = ScreenMain
		m.mainScreen.SetLoading(true)
		cmds = append(cmds, m.mainScreen.Init(), m.fetchDataCmd())

	case messages.DataFetchedMsg:
		debugf("DataFetchedMsg received: %d rooms, %d scenes", len(msg.Rooms), len(msg.Scenes))
		m.rooms = msg.Rooms
		m.scenes = msg.Scenes
		m.mainScreen.SetData(m.rooms, m.scenes)
		m.scenesScreen.SetScenes(m.scenes, m.rooms)
		debugf("SetData called, mainScreen.loading should be false now")

		// Start event subscription (skip in demo mode - state changes are immediate)
		if m.events == nil && m.bridge != nil && !m.demoMode {
			debugf("Starting event subscription")
			// Cast to *HueBridge for event subscription (only real bridges support SSE)
			if hueBridge, ok := m.bridge.(*api.HueBridge); ok {
				m.events = api.NewEventSubscription(hueBridge, func(events []api.Event) {
					debugf("Received %d events from WebSocket", len(events))
					for _, event := range events {
						debugf("  Event: type=%s resource=%s id=%s", event.Type, event.Resource, event.ResourceID)
						if event.Resource == "light" && event.Type == api.EventTypeUpdate {
							if update, err := api.ParseLightUpdate(event); err == nil {
								msg := messages.LightUpdateMsg{
									LightID: update.ID,
									On:      update.On,
								}
								if update.Brightness != nil {
									b := int(*update.Brightness)
									msg.Brightness = &b
								}
								if update.ColorTemp != nil {
									msg.ColorTemp = update.ColorTemp
								}
								if update.ColorXY != nil {
									msg.ColorXY = &struct{ X, Y float64 }{update.ColorXY.X, update.ColorXY.Y}
								}
								debugf("  Parsed light update: id=%s on=%v brightness=%v", update.ID, update.On, update.Brightness)
								// Non-blocking send to avoid deadlock
								select {
								case m.eventChan <- msg:
									debugf("  Sent to event channel")
								default:
									debugf("  Channel full, dropped event")
								}
							} else {
								debugf("  Failed to parse light update: %v", err)
							}
						}
					}
				})
				if err := m.events.Start(m.ctx); err != nil {
					debugf("Failed to start event subscription: %v", err)
					m.err = err
				} else {
					debugf("Event subscription started successfully")
				}
				cmds = append(cmds, m.listenForEvents())
			}
		}

	case messages.ErrorMsg:
		m.err = msg.Err
		// Stop the loading spinner on error
		m.mainScreen.SetLoading(false)

	case messages.ShowScenesMsg:
		m.screen = ScreenScenes
		m.scenesScreen.SetRoomFilter(msg.RoomID)
		return m, nil

	case messages.HideScenesMsg:
		m.screen = ScreenMain
		return m, nil

	case messages.SceneActivatedMsg:
		m.screen = ScreenMain
		if m.bridge != nil {
			cmds = append(cmds, m.activateSceneCmd(msg.SceneID))
		}

	case messages.RefreshMsg:
		m.mainScreen.SetLoading(true)
		cmds = append(cmds, m.mainScreen.Init(), m.fetchDataCmd())

	case messages.LightUpdateMsg:
		// Handle real-time light updates from WebSocket
		debugf("Handling LightUpdateMsg: id=%s on=%v brightness=%v colorTemp=%v",
			msg.LightID, msg.On, msg.Brightness, msg.ColorTemp)

		light := m.findLightByID(msg.LightID)
		if light == nil {
			debugf("  Light not found: %s", msg.LightID)
			cmds = append(cmds, m.listenForEvents())
			return m, tea.Batch(cmds...)
		}
		debugf("  Found light: %s (%s)", light.Name, light.ID)

		updated := false

		if msg.On != nil {
			if !m.pending.MatchesAndClear(msg.LightID, "on", *msg.On) {
				debugf("  Applying on=%v (no pending match)", *msg.On)
				light.On = *msg.On
				updated = true
			} else {
				debugf("  Ignoring on=%v (matched pending op)", *msg.On)
			}
		}

		if msg.Brightness != nil {
			if !m.pending.MatchesAndClear(msg.LightID, "brightness", *msg.Brightness) {
				debugf("  Applying brightness=%v (no pending match)", *msg.Brightness)
				light.SetBrightnessPct(*msg.Brightness)
				updated = true
			} else {
				debugf("  Ignoring brightness=%v (matched pending op)", *msg.Brightness)
			}
		}

		// Check pending ops BEFORE processing (MatchesAndClear removes them)
		hasPendingColorXY := m.pending.HasPending(msg.LightID, "color_xy")
		hasPendingColorTemp := m.pending.HasPending(msg.LightID, "color_temp")

		if msg.ColorTemp != nil {
			// Ignore invalid colorTemp (0 or outside valid mirek range 153-500)
			// The bridge sends colorTemp=0 when switching to XY color mode
			if *msg.ColorTemp < 153 || *msg.ColorTemp > 500 {
				debugf("  Ignoring invalid colorTemp=%v (outside range 153-500)", *msg.ColorTemp)
			} else if hasPendingColorXY {
				// Ignore colorTemp if we have a pending color_xy change
				// (they're mutually exclusive modes)
				debugf("  Ignoring colorTemp=%v (pending color_xy op exists)", *msg.ColorTemp)
			} else if !m.pending.MatchesAndClear(msg.LightID, "color_temp", *msg.ColorTemp) {
				debugf("  Applying colorTemp=%v (no pending match)", *msg.ColorTemp)
				if light.Color == nil {
					light.Color = &models.Color{}
				}
				light.Color.Mirek = uint16(*msg.ColorTemp)
				light.Color.Mode = models.ColorModeColorTemp
				light.Color.InvalidateCache()
				updated = true
			} else {
				debugf("  Ignoring colorTemp=%v (matched pending op)", *msg.ColorTemp)
			}
		}

		if msg.ColorXY != nil {
			xy := struct{ X, Y float64 }{msg.ColorXY.X, msg.ColorXY.Y}
			// Check if we have ANY pending color_xy op (ignore echoes during rapid changes)
			if hasPendingColorXY {
				if m.pending.MatchesAndClear(msg.LightID, "color_xy", xy) {
					debugf("  Ignoring colorXY (matched pending op)")
				} else {
					debugf("  Ignoring colorXY={%v,%v} (pending color_xy op exists, waiting for final value)", msg.ColorXY.X, msg.ColorXY.Y)
				}
			} else if hasPendingColorTemp {
				// Ignore colorXY if we have a pending color_temp change
				// (they're mutually exclusive modes - bridge sends both)
				debugf("  Ignoring colorXY={%v,%v} (pending color_temp op exists)", msg.ColorXY.X, msg.ColorXY.Y)
			} else {
				debugf("  Applying colorXY={%v,%v} (no pending match)", msg.ColorXY.X, msg.ColorXY.Y)
				if light.Color == nil {
					light.Color = &models.Color{}
				}
				light.Color.X = msg.ColorXY.X
				light.Color.Y = msg.ColorXY.Y
				light.Color.Mode = models.ColorModeXY
				light.Color.InvalidateCache()
				updated = true
			}
		}

		debugf("  Updated=%v", updated)

		if updated {
			// Update room state (AllOn/AnyOn)
			for _, room := range m.rooms {
				for _, l := range room.Lights {
					if l.ID == msg.LightID {
						room.UpdateState()
						break
					}
				}
			}
		}

		cmds = append(cmds, m.listenForEvents())
	}

	// Route to current screen
	switch m.screen {
	case ScreenSetup:
		var cmd tea.Cmd
		m.setupScreen, cmd = m.setupScreen.Update(msg)
		cmds = append(cmds, cmd)

	case ScreenMain:
		var cmd tea.Cmd
		m.mainScreen, cmd = m.mainScreen.Update(msg, m.bridge, func(lightID, field string, value interface{}, dir screens.Direction) {
			m.pending.AddWithDirection(lightID, field, value, Direction(dir))
		})
		cmds = append(cmds, cmd)

	case ScreenScenes:
		var cmd tea.Cmd
		m.scenesScreen, cmd = m.scenesScreen.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the current screen
func (m Model) View() string {
	var view string
	switch m.screen {
	case ScreenSetup:
		view = m.setupScreen.View()
	case ScreenMain:
		view = m.mainScreen.View()
	case ScreenScenes:
		view = m.scenesScreen.View()
	default:
		view = "Unknown screen"
	}

	// Append error message if there's an error
	if m.err != nil {
		view += "\n\n  ⚠ Error: " + m.err.Error()
	}

	return view
}

// fetchDataCmd creates a command to fetch all data from the bridge
func (m Model) fetchDataCmd() tea.Cmd {
	debugf("fetchDataCmd called, bridge=%v, demoMode=%v", m.bridge != nil, m.demoMode)
	// Capture bridge reference directly to avoid closure issues
	bridge := m.bridge
	ctx := m.ctx
	return func() tea.Msg {
		debugf("fetchDataCmd executing, bridge=%v", bridge != nil)
		if bridge == nil {
			debugf("fetchDataCmd: bridge is nil!")
			return messages.ErrorMsg{Err: config.ErrNoBridges}
		}

		rooms, scenes, err := bridge.FetchAll(ctx)
		debugf("fetchDataCmd: FetchAll returned %d rooms, %d scenes, err=%v", len(rooms), len(scenes), err)
		if err != nil {
			return messages.ErrorMsg{Err: err}
		}

		return messages.DataFetchedMsg{Rooms: rooms, Scenes: scenes}
	}
}

// activateSceneCmd creates a command to activate a scene
func (m Model) activateSceneCmd(sceneID string) tea.Cmd {
	return func() tea.Msg {
		if m.bridge == nil {
			return messages.ErrorMsg{Err: config.ErrNoBridges}
		}

		err := m.bridge.ActivateScene(m.ctx, sceneID)
		if err != nil {
			return messages.ErrorMsg{Err: err}
		}

		return messages.RefreshMsg{}
	}
}

// listenForEvents creates a command that waits for the next event from the channel
func (m Model) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		return <-m.eventChan
	}
}

// findLightByID finds a light by its ID across all rooms
func (m Model) findLightByID(lightID string) *models.Light {
	for _, room := range m.rooms {
		for _, light := range room.Lights {
			if light.ID == lightID {
				return light
			}
		}
	}
	return nil
}
