package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/angristan/hue-tui/internal/api"
	"github.com/angristan/hue-tui/internal/config"
	"github.com/angristan/hue-tui/internal/models"
	"github.com/angristan/hue-tui/internal/tui/messages"
	"github.com/angristan/hue-tui/internal/tui/screens"
)

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
	bridge *api.HueBridge
	events *api.EventSubscription

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
func NewModel(cfg *config.Config) Model {
	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	// Determine initial screen
	if cfg.HasBridges() {
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
	cmds := []tea.Cmd{
		tea.SetWindowTitle("Hue CLI"),
	}

	// Start with appropriate screen initialization
	switch m.screen {
	case ScreenSetup:
		cmds = append(cmds, m.setupScreen.Init())
	case ScreenMain:
		cmds = append(cmds, m.mainScreen.Init(), m.fetchDataCmd())
	}

	return tea.Batch(cmds...)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

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
		m.config.AddBridge(config.BridgeConfig{
			Host:     msg.Bridge.Host(),
			Username: msg.AppKey,
			BridgeID: msg.Bridge.BridgeID(),
		})
		m.config.LastBridgeID = msg.Bridge.BridgeID()
		if err := m.config.Save(); err != nil {
			m.err = err
		}

		m.screen = ScreenMain
		m.mainScreen.SetLoading(true)
		cmds = append(cmds, m.mainScreen.Init(), m.fetchDataCmd())

	case messages.DataFetchedMsg:
		m.rooms = msg.Rooms
		m.scenes = msg.Scenes
		m.mainScreen.SetData(m.rooms, m.scenes)
		m.scenesScreen.SetScenes(m.scenes, m.rooms)

		// Start event subscription
		if m.events == nil && m.bridge != nil {
			m.events = api.NewEventSubscription(m.bridge, func(events []api.Event) {
				// Handle events - will be converted to tea.Msg in production
			})
			if err := m.events.Start(m.ctx); err != nil {
				m.err = err
			}
		}

	case messages.ErrorMsg:
		m.err = msg.Err

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
	}

	// Route to current screen
	switch m.screen {
	case ScreenSetup:
		var cmd tea.Cmd
		m.setupScreen, cmd = m.setupScreen.Update(msg)
		cmds = append(cmds, cmd)

	case ScreenMain:
		var cmd tea.Cmd
		m.mainScreen, cmd = m.mainScreen.Update(msg, m.bridge)
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
	switch m.screen {
	case ScreenSetup:
		return m.setupScreen.View()
	case ScreenMain:
		return m.mainScreen.View()
	case ScreenScenes:
		return m.scenesScreen.View()
	default:
		return "Unknown screen"
	}
}

// fetchDataCmd creates a command to fetch all data from the bridge
func (m Model) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		if m.bridge == nil {
			return messages.ErrorMsg{Err: config.ErrNoBridges}
		}

		rooms, scenes, err := m.bridge.FetchAll(m.ctx)
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
