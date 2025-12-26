package screens

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/angristan/hue-tui/internal/api"
	"github.com/angristan/hue-tui/internal/tui/messages"
	"github.com/angristan/hue-tui/internal/tui/styles"
)

// SetupState represents the current setup state
type SetupState int

const (
	StateDiscovering SetupState = iota
	StateBridgeList
	StateManualEntry
	StatePairing
	StateSuccess
	StateError
)

// SetupModel is the setup screen model
type SetupModel struct {
	state    SetupState
	bridges  []api.DiscoveredBridge
	selected int
	input    textinput.Model
	spinner  spinner.Model
	err      error
	message  string

	// Pairing state
	pairingHost     string
	pairingBridgeID string

	// Window size
	width  int
	height int
}

// NewSetupModel creates a new setup screen model
func NewSetupModel() SetupModel {
	ti := textinput.New()
	ti.Placeholder = "192.168.1.x"
	ti.CharLimit = 45

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.StyleSpinner

	return SetupModel{
		state:   StateDiscovering,
		input:   ti,
		spinner: sp,
	}
}

// Init initializes the setup screen
func (m SetupModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.discoverCmd(),
	)
}

// SetSize sets the terminal size
func (m *SetupModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages
func (m SetupModel) Update(msg tea.Msg) (SetupModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateBridgeList:
			switch msg.String() {
			case "up", "k":
				if m.selected > 0 {
					m.selected--
				}
			case "down", "j":
				if m.selected < len(m.bridges) {
					m.selected++
				}
			case "enter":
				if m.selected < len(m.bridges) {
					// Start pairing with selected bridge
					bridge := m.bridges[m.selected]
					m.state = StatePairing
					m.pairingHost = bridge.Host
					m.pairingBridgeID = bridge.BridgeID
					cmds = append(cmds, m.pairCmd())
				} else {
					// Manual entry selected
					m.state = StateManualEntry
					m.input.Focus()
					cmds = append(cmds, textinput.Blink)
				}
			case "m":
				m.state = StateManualEntry
				m.input.Focus()
				cmds = append(cmds, textinput.Blink)
			case "r":
				m.state = StateDiscovering
				cmds = append(cmds, m.discoverCmd())
			}

		case StateManualEntry:
			switch msg.String() {
			case "enter":
				host := strings.TrimSpace(m.input.Value())
				if host != "" {
					m.state = StatePairing
					m.pairingHost = host
					cmds = append(cmds, m.pairCmd())
				}
			case "esc":
				m.state = StateBridgeList
				m.input.Blur()
			}
		}

	case BridgesDiscoveredMsg:
		m.bridges = msg.Bridges
		m.state = StateBridgeList

	case PairingSuccessMsg:
		m.state = StateSuccess
		m.message = "Successfully paired with bridge!"
		return m, func() tea.Msg {
			return messages.BridgeConnectedMsg{
				Bridge: msg.Bridge,
				AppKey: msg.AppKey,
			}
		}

	case PairingErrorMsg:
		m.state = StateError
		m.err = msg.Err

	case DiscoveryErrorMsg:
		m.state = StateBridgeList
		m.err = msg.Err

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update text input
	if m.state == StateManualEntry {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the setup screen
func (m SetupModel) View() string {
	var b strings.Builder

	// Header
	header := styles.StyleHeaderGradient.Render("  Hue CLI Setup  ")
	b.WriteString(lipgloss.Place(m.width, 3, lipgloss.Center, lipgloss.Top, header))
	b.WriteString("\n\n")

	// Content based on state
	var content string
	switch m.state {
	case StateDiscovering:
		content = m.renderDiscovering()
	case StateBridgeList:
		content = m.renderBridgeList()
	case StateManualEntry:
		content = m.renderManualEntry()
	case StatePairing:
		content = m.renderPairing()
	case StateSuccess:
		content = m.renderSuccess()
	case StateError:
		content = m.renderError()
	}

	b.WriteString(lipgloss.Place(m.width, m.height-6, lipgloss.Center, lipgloss.Center, content))

	return b.String()
}

func (m SetupModel) renderDiscovering() string {
	return fmt.Sprintf("%s Searching for Hue bridges...", m.spinner.View())
}

func (m SetupModel) renderBridgeList() string {
	var b strings.Builder

	if len(m.bridges) == 0 {
		b.WriteString(styles.StyleTextMuted.Render("No bridges found.\n\n"))
	} else {
		b.WriteString("Found bridges:\n\n")
		for i, bridge := range m.bridges {
			cursor := "  "
			style := styles.StyleLightName
			if i == m.selected {
				cursor = "> "
				style = styles.StyleSceneItemSelected
			}
			name := bridge.Host
			if bridge.BridgeID != "" && len(bridge.BridgeID) >= 8 {
				name = fmt.Sprintf("%s (%s)", bridge.Host, bridge.BridgeID[:8])
			}
			b.WriteString(cursor + style.Render(name) + "\n")
		}
	}

	// Manual entry option
	cursor := "  "
	style := styles.StyleLightName
	if m.selected >= len(m.bridges) {
		cursor = "> "
		style = styles.StyleSceneItemSelected
	}
	b.WriteString("\n" + cursor + style.Render("Enter IP manually...") + "\n")

	b.WriteString("\n" + styles.StyleHelp.Render("↑/↓ navigate • enter select • r refresh • m manual"))

	return b.String()
}

func (m SetupModel) renderManualEntry() string {
	var b strings.Builder

	b.WriteString("Enter bridge IP address:\n\n")
	b.WriteString(styles.StyleInputFocused.Render(m.input.View()))
	b.WriteString("\n\n" + styles.StyleHelp.Render("enter confirm • esc back"))

	return b.String()
}

func (m SetupModel) renderPairing() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s Pairing with %s...\n\n", m.spinner.View(), m.pairingHost))
	b.WriteString(styles.StylePrimary.Render("Press the link button on your Hue bridge"))

	return b.String()
}

func (m SetupModel) renderSuccess() string {
	return styles.StyleSuccess.Render("✓ " + m.message)
}

func (m SetupModel) renderError() string {
	return styles.StyleError.Render("✗ Error: " + m.err.Error())
}

// Commands

func (m SetupModel) discoverCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		bridges, err := api.DiscoverAll(ctx, 5*time.Second)
		if err != nil {
			return DiscoveryErrorMsg{Err: err}
		}
		return BridgesDiscoveredMsg{Bridges: bridges}
	}
}

func (m SetupModel) pairCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()

		appKey, err := api.CreateAppKey(ctx, m.pairingHost, "hue-cli-go#device", 30*time.Second)
		if err != nil {
			return PairingErrorMsg{Err: err}
		}

		// Get bridge ID
		bridgeID, err := api.GetBridgeID(ctx, m.pairingHost)
		if err != nil {
			return PairingErrorMsg{Err: err}
		}

		bridge := api.NewHueBridge(m.pairingHost, appKey, bridgeID)

		return PairingSuccessMsg{
			Bridge: bridge,
			AppKey: appKey,
		}
	}
}

// Messages

type BridgesDiscoveredMsg struct {
	Bridges []api.DiscoveredBridge
}

type DiscoveryErrorMsg struct {
	Err error
}

type PairingSuccessMsg struct {
	Bridge *api.HueBridge
	AppKey string
}

type PairingErrorMsg struct {
	Err error
}
