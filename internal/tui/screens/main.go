package screens

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/angristan/hue-tui/internal/api"
	"github.com/angristan/hue-tui/internal/models"
	"github.com/angristan/hue-tui/internal/tui/messages"
)

// Direction represents the direction of a change
type Direction int

const (
	DirExact Direction = iota // Exact match required (for booleans)
	DirUp                     // Value is increasing
	DirDown                   // Value is decreasing
)

// PendingAdder is a function that registers a pending operation with direction
type PendingAdder func(lightID, field string, value interface{}, dir Direction)

// Colors
var (
	colorPrimary = lipgloss.Color("#B794F4")
	colorMuted   = lipgloss.Color("#6B6B80")
	colorSuccess = lipgloss.Color("#68D391")
	colorWarning = lipgloss.Color("#FBBF24")
	colorDim     = lipgloss.Color("#4A4A5A")
)

// Styles
var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 1)

	styleRoomName = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleLightOn = lipgloss.NewStyle().
			Foreground(colorWarning)

	styleLightOff = lipgloss.NewStyle().
			Foreground(colorDim)

	styleLightName = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA"))

	styleLightNameDim = lipgloss.NewStyle().
				Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleBrightness = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colorPrimary)

	styleSearch = lipgloss.NewStyle().
			Foreground(colorPrimary)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)
)

// listItem represents either a room header or a light in the unified list
type listItem struct {
	isRoom bool
	room   *models.Room
	light  *models.Light
}

// MainModel is the main dashboard screen model
type MainModel struct {
	rooms         []*models.Room
	scenes        []*models.Scene
	selectedIndex int
	scrollOffset  int        // Vertical scroll offset
	items         []listItem // Unified list of rooms and lights
	lightToRoom   map[string]*models.Room

	showPanel   bool
	searchMode  bool
	searchInput textinput.Model
	searchQuery string

	// Loading state
	loading bool
	spinner spinner.Model

	width  int
	height int
}

// NewMainModel creates a new main screen model
func NewMainModel(keys interface{}) MainModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 50

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return MainModel{
		searchInput: ti,
		lightToRoom: make(map[string]*models.Room),
		showPanel:   true, // Side panel on by default
		loading:     true, // Start in loading state
		spinner:     sp,
	}
}

// Init initializes the main screen
func (m MainModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *MainModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// visibleLines returns how many items fit in the viewport
func (m *MainModel) visibleLines() int {
	// Match the content height calculation in View()
	contentHeight := m.height - 5
	if m.searchMode || m.searchQuery != "" {
		contentHeight -= 1
	}
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Subtract scroll indicators (up to 2 lines)
	contentHeight -= 2

	// Room headers take 2 lines, lights take 1 line
	// Use conservative estimate: ~1.3 lines per item on average
	visible := contentHeight * 3 / 4
	if visible < 2 {
		visible = 2
	}
	return visible
}

// ensureVisible adjusts scrollOffset so selectedIndex is visible
func (m *MainModel) ensureVisible() {
	visible := m.visibleLines()

	// Scroll up if selection is above viewport
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}

	// Scroll down if selection is below viewport
	if m.selectedIndex >= m.scrollOffset+visible {
		m.scrollOffset = m.selectedIndex - visible + 1
	}

	// Clamp scroll offset
	maxScroll := len(m.items) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *MainModel) SetData(rooms []*models.Room, scenes []*models.Scene) {
	// Sort rooms alphabetically by name
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})
	m.rooms = rooms
	m.scenes = scenes
	m.loading = false
	m.scrollOffset = 0
	m.rebuildLightList()
}

func (m *MainModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m *MainModel) rebuildLightList() {
	m.items = nil
	m.lightToRoom = make(map[string]*models.Room)

	for _, room := range m.rooms {
		hasMatchingLights := false
		var roomLights []*models.Light

		for _, light := range room.Lights {
			if m.searchQuery == "" || strings.Contains(strings.ToLower(light.Name), strings.ToLower(m.searchQuery)) {
				roomLights = append(roomLights, light)
				m.lightToRoom[light.ID] = room
				hasMatchingLights = true
			}
		}

		if hasMatchingLights {
			// Sort lights alphabetically by name
			sort.Slice(roomLights, func(i, j int) bool {
				return roomLights[i].Name < roomLights[j].Name
			})
			// Add room header
			m.items = append(m.items, listItem{isRoom: true, room: room})
			// Add lights
			for _, light := range roomLights {
				m.items = append(m.items, listItem{isRoom: false, light: light, room: room})
			}
		}
	}

	if m.selectedIndex >= len(m.items) {
		m.selectedIndex = max(0, len(m.items)-1)
	}
	m.scrollOffset = 0
	m.ensureVisible()
}

func (m *MainModel) SelectedItem() *listItem {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.items) {
		return &m.items[m.selectedIndex]
	}
	return nil
}

func (m *MainModel) SelectedLight() *models.Light {
	if item := m.SelectedItem(); item != nil && !item.isRoom {
		return item.light
	}
	return nil
}

func (m *MainModel) SelectedRoom() *models.Room {
	if item := m.SelectedItem(); item != nil {
		return item.room
	}
	return nil
}

func (m *MainModel) IsRoomSelected() bool {
	if item := m.SelectedItem(); item != nil {
		return item.isRoom
	}
	return false
}

func (m MainModel) Update(msg tea.Msg, bridge api.BridgeClient, addPending PendingAdder) (MainModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				m.rebuildLightList()
				return m, nil
			case "enter":
				m.searchMode = false
				m.searchQuery = m.searchInput.Value()
				m.searchInput.Blur()
				m.rebuildLightList()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = m.searchInput.Value()
				m.rebuildLightList()
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.ensureVisible()
			}

		case "down", "j":
			if m.selectedIndex < len(m.items)-1 {
				m.selectedIndex++
				m.ensureVisible()
			}

		case "pgup":
			m.selectedIndex -= m.visibleLines()
			if m.selectedIndex < 0 {
				m.selectedIndex = 0
			}
			m.ensureVisible()

		case "pgdown":
			m.selectedIndex += m.visibleLines()
			if m.selectedIndex >= len(m.items) {
				m.selectedIndex = len(m.items) - 1
			}
			if m.selectedIndex < 0 {
				m.selectedIndex = 0
			}
			m.ensureVisible()

		case "home":
			m.selectedIndex = 0
			m.ensureVisible()

		case "end":
			m.selectedIndex = len(m.items) - 1
			if m.selectedIndex < 0 {
				m.selectedIndex = 0
			}
			m.ensureVisible()

		case "left", "h":
			if m.IsRoomSelected() {
				// Dim all lights in room
				if room := m.SelectedRoom(); room != nil {
					for _, light := range room.Lights {
						if light.On {
							newBrightness := max(10, light.BrightnessPct()-10)
							light.SetBrightnessPct(newBrightness)
							if addPending != nil {
								addPending(light.ID, "brightness", newBrightness, DirDown)
							}
							cmds = append(cmds, m.setBrightnessCmd(bridge, light.ID, newBrightness))
						}
					}
				}
			} else if light := m.SelectedLight(); light != nil && light.On {
				newBrightness := max(0, light.BrightnessPct()-10)
				if newBrightness == 0 {
					light.On = false
					if addPending != nil {
						addPending(light.ID, "on", false, DirExact)
					}
					cmds = append(cmds, m.toggleLightCmd(bridge, light.ID, false))
				} else {
					light.SetBrightnessPct(newBrightness)
					if addPending != nil {
						addPending(light.ID, "brightness", newBrightness, DirDown)
					}
					cmds = append(cmds, m.setBrightnessCmd(bridge, light.ID, newBrightness))
				}
			}

		case "right", "l":
			if m.IsRoomSelected() {
				// Brighten all lights in room
				if room := m.SelectedRoom(); room != nil {
					for _, light := range room.Lights {
						if light.On {
							newBrightness := min(100, light.BrightnessPct()+10)
							light.SetBrightnessPct(newBrightness)
							if addPending != nil {
								addPending(light.ID, "brightness", newBrightness, DirUp)
							}
							cmds = append(cmds, m.setBrightnessCmd(bridge, light.ID, newBrightness))
						}
					}
				}
			} else if light := m.SelectedLight(); light != nil {
				if !light.On {
					light.On = true
					light.SetBrightnessPct(10)
					if addPending != nil {
						addPending(light.ID, "on", true, DirExact)
						addPending(light.ID, "brightness", 10, DirUp)
					}
					cmds = append(cmds, m.toggleLightCmd(bridge, light.ID, true))
					cmds = append(cmds, m.setBrightnessCmd(bridge, light.ID, 10))
				} else {
					newBrightness := min(100, light.BrightnessPct()+10)
					light.SetBrightnessPct(newBrightness)
					if addPending != nil {
						addPending(light.ID, "brightness", newBrightness, DirUp)
					}
					cmds = append(cmds, m.setBrightnessCmd(bridge, light.ID, newBrightness))
				}
			}

		case " ":
			if m.IsRoomSelected() {
				// Toggle all lights in room
				if room := m.SelectedRoom(); room != nil && room.GroupedLightID != "" {
					newState := !room.AnyOn
					for _, l := range room.Lights {
						l.On = newState
						if addPending != nil {
							addPending(l.ID, "on", newState, DirExact)
						}
					}
					room.UpdateState()
					cmds = append(cmds, m.setGroupOnCmd(bridge, room.GroupedLightID, newState))
				}
			} else if light := m.SelectedLight(); light != nil {
				light.On = !light.On
				if addPending != nil {
					addPending(light.ID, "on", light.On, DirExact)
				}
				cmds = append(cmds, m.toggleLightCmd(bridge, light.ID, light.On))
			}

		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if light := m.SelectedLight(); light != nil {
				brightness := brightnessFromKey(msg.String())
				if brightness >= 0 {
					oldBrightness := light.BrightnessPct()
					light.SetBrightnessPct(brightness)
					if !light.On {
						light.On = true
						if addPending != nil {
							addPending(light.ID, "on", true, DirExact)
						}
						cmds = append(cmds, m.toggleLightCmd(bridge, light.ID, true))
					}
					if addPending != nil {
						dir := DirExact
						if brightness > oldBrightness {
							dir = DirUp
						} else if brightness < oldBrightness {
							dir = DirDown
						}
						addPending(light.ID, "brightness", brightness, dir)
					}
					cmds = append(cmds, m.setBrightnessCmd(bridge, light.ID, brightness))
				}
			}

		case "w":
			if light := m.SelectedLight(); light != nil && light.SupportsColorTemp && light.Color != nil {
				// Switch to temperature mode and make warmer (higher mirek = warmer)
				if light.Color.Mirek == 0 {
					light.Color.Mirek = 326 // Default to middle (3000K)
				}
				newMirek := min(500, int(light.Color.Mirek)+25)
				light.Color.Mirek = uint16(newMirek)
				light.Color.Mode = models.ColorModeColorTemp
				light.Color.InvalidateCache()
				if addPending != nil {
					addPending(light.ID, "color_temp", newMirek, DirUp)
				}
				cmds = append(cmds, m.setColorTempCmd(bridge, light.ID, newMirek))
			}

		case "c":
			if light := m.SelectedLight(); light != nil && light.SupportsColorTemp && light.Color != nil {
				// Switch to temperature mode and make cooler (lower mirek = cooler)
				if light.Color.Mirek == 0 {
					light.Color.Mirek = 326 // Default to middle (3000K)
				}
				newMirek := max(153, int(light.Color.Mirek)-25)
				light.Color.Mirek = uint16(newMirek)
				light.Color.Mode = models.ColorModeColorTemp
				light.Color.InvalidateCache()
				if addPending != nil {
					addPending(light.ID, "color_temp", newMirek, DirDown)
				}
				cmds = append(cmds, m.setColorTempCmd(bridge, light.ID, newMirek))
			}

		case "[":
			// Decrease hue (rotate color wheel left)
			if light := m.SelectedLight(); light != nil && light.SupportsColor && light.Color != nil {
				// Initialize HS from current color if switching from other mode
				if light.Color.Mode != models.ColorModeHS {
					r, g, b := light.Color.RGB()
					h, s := rgbToHueSat(r, g, b)
					light.Color.Hue = uint16(float64(h) / 360.0 * 65535.0)
					light.Color.Saturation = uint8(float64(s) / 100.0 * 254.0)
					light.Color.Brightness = light.Brightness // Preserve brightness
				}
				newHue := (int(light.Color.Hue) - 3640 + 65536) % 65536 // -20° in hue units
				light.Color.Hue = uint16(newHue)
				light.Color.Mode = models.ColorModeHS
				light.Color.InvalidateCache()
				if addPending != nil {
					x, y := api.HSToXY(light.Color.Hue, light.Color.Saturation)
					addPending(light.ID, "color_xy", struct{ X, Y float64 }{x, y}, DirExact)
				}
				cmds = append(cmds, m.setColorHSCmd(bridge, light.ID, light.Color.Hue, light.Color.Saturation))
			}

		case "]":
			// Increase hue (rotate color wheel right)
			if light := m.SelectedLight(); light != nil && light.SupportsColor && light.Color != nil {
				// Initialize HS from current color if switching from other mode
				if light.Color.Mode != models.ColorModeHS {
					r, g, b := light.Color.RGB()
					h, s := rgbToHueSat(r, g, b)
					light.Color.Hue = uint16(float64(h) / 360.0 * 65535.0)
					light.Color.Saturation = uint8(float64(s) / 100.0 * 254.0)
					light.Color.Brightness = light.Brightness // Preserve brightness
				}
				newHue := (int(light.Color.Hue) + 3640) % 65536 // +20° in hue units
				light.Color.Hue = uint16(newHue)
				light.Color.Mode = models.ColorModeHS
				light.Color.InvalidateCache()
				if addPending != nil {
					x, y := api.HSToXY(light.Color.Hue, light.Color.Saturation)
					addPending(light.ID, "color_xy", struct{ X, Y float64 }{x, y}, DirExact)
				}
				cmds = append(cmds, m.setColorHSCmd(bridge, light.ID, light.Color.Hue, light.Color.Saturation))
			}

		case "-":
			// Decrease saturation
			if light := m.SelectedLight(); light != nil && light.SupportsColor && light.Color != nil {
				// Initialize HS from current color if switching from other mode
				if light.Color.Mode != models.ColorModeHS {
					r, g, b := light.Color.RGB()
					h, s := rgbToHueSat(r, g, b)
					light.Color.Hue = uint16(float64(h) / 360.0 * 65535.0)
					light.Color.Saturation = uint8(float64(s) / 100.0 * 254.0)
					light.Color.Brightness = light.Brightness // Preserve brightness
				}
				newSat := max(0, int(light.Color.Saturation)-25)
				light.Color.Saturation = uint8(newSat)
				light.Color.Mode = models.ColorModeHS
				light.Color.InvalidateCache()
				if addPending != nil {
					x, y := api.HSToXY(light.Color.Hue, light.Color.Saturation)
					addPending(light.ID, "color_xy", struct{ X, Y float64 }{x, y}, DirExact)
				}
				cmds = append(cmds, m.setColorHSCmd(bridge, light.ID, light.Color.Hue, light.Color.Saturation))
			}

		case "=", "+":
			// Increase saturation
			if light := m.SelectedLight(); light != nil && light.SupportsColor && light.Color != nil {
				// Initialize HS from current color if switching from other mode
				if light.Color.Mode != models.ColorModeHS {
					r, g, b := light.Color.RGB()
					h, s := rgbToHueSat(r, g, b)
					light.Color.Hue = uint16(float64(h) / 360.0 * 65535.0)
					light.Color.Saturation = uint8(float64(s) / 100.0 * 254.0)
					light.Color.Brightness = light.Brightness // Preserve brightness
				}
				newSat := min(254, int(light.Color.Saturation)+25)
				light.Color.Saturation = uint8(newSat)
				light.Color.Mode = models.ColorModeHS
				light.Color.InvalidateCache()
				if addPending != nil {
					x, y := api.HSToXY(light.Color.Hue, light.Color.Saturation)
					addPending(light.ID, "color_xy", struct{ X, Y float64 }{x, y}, DirExact)
				}
				cmds = append(cmds, m.setColorHSCmd(bridge, light.ID, light.Color.Hue, light.Color.Saturation))
			}

		case "a":
			if room := m.SelectedRoom(); room != nil && room.GroupedLightID != "" {
				for _, l := range room.Lights {
					l.On = true
					if addPending != nil {
						addPending(l.ID, "on", true, DirExact)
					}
				}
				room.UpdateState()
				cmds = append(cmds, m.setGroupOnCmd(bridge, room.GroupedLightID, true))
			}

		case "x":
			if room := m.SelectedRoom(); room != nil && room.GroupedLightID != "" {
				for _, l := range room.Lights {
					l.On = false
					if addPending != nil {
						addPending(l.ID, "on", false, DirExact)
					}
				}
				room.UpdateState()
				cmds = append(cmds, m.setGroupOnCmd(bridge, room.GroupedLightID, false))
			}

		case "s":
			roomID := ""
			if room := m.SelectedRoom(); room != nil {
				roomID = room.ID
			}
			return m, func() tea.Msg { return messages.ShowScenesMsg{RoomID: roomID} }

		case "/":
			m.searchMode = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "tab":
			m.showPanel = !m.showPanel

		case "r":
			m.loading = true
			cmds = append(cmds, m.spinner.Tick)
			return m, tea.Batch(func() tea.Msg { return messages.RefreshMsg{} }, tea.Batch(cmds...))
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	var b strings.Builder

	// Header
	header := styleHeader.Render(" HUE CLI ")
	var status string
	if m.loading {
		status = lipgloss.NewStyle().Foreground(colorWarning).Render(" ⟳ Loading...")
	} else {
		status = lipgloss.NewStyle().Foreground(colorSuccess).Render(" ● Connected")
	}
	headerLine := header + status
	b.WriteString(headerLine)
	b.WriteString("\n")

	// Search bar
	if m.searchMode {
		b.WriteString(styleSearch.Render("/ ") + m.searchInput.View())
		b.WriteString("\n")
	} else if m.searchQuery != "" {
		b.WriteString(styleSearch.Render("/ " + m.searchQuery + " "))
		b.WriteString(styleMuted.Render("(esc to clear)"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Calculate content area with responsive layout
	contentWidth := m.width
	panelWidth := 0
	// Auto-hide panel on narrow terminals, show panel only if enabled and width >= 80
	showPanelNow := m.showPanel && m.width >= 80
	if showPanelNow {
		// Panel takes ~30% of width, with min 30 and max 45
		panelWidth = m.width * 30 / 100
		if panelWidth < 30 {
			panelWidth = 30
		}
		if panelWidth > 45 {
			panelWidth = 45
		}
		contentWidth = m.width - panelWidth - 3
	}

	// Main content with vertical scrolling
	var content strings.Builder
	visible := m.visibleLines()
	endIdx := m.scrollOffset + visible
	if endIdx > len(m.items) {
		endIdx = len(m.items)
	}

	// Show scroll indicator at top if scrolled
	if m.scrollOffset > 0 {
		content.WriteString(styleMuted.Render(fmt.Sprintf("  ↑ %d more above", m.scrollOffset)))
		content.WriteString("\n")
	}

	for idx := m.scrollOffset; idx < endIdx; idx++ {
		item := m.items[idx]
		isSelected := idx == m.selectedIndex

		if item.isRoom {
			// Add blank line before room (except first visible item)
			if idx > m.scrollOffset {
				content.WriteString("\n")
			}
			content.WriteString(m.renderRoomHeader(item.room, isSelected))
			content.WriteString("\n")
		} else {
			// Light row - no extra spacing needed
			content.WriteString(m.renderLightRow(item.light, isSelected, contentWidth))
			content.WriteString("\n")
		}
	}

	// Show scroll indicator at bottom if more items
	if endIdx < len(m.items) {
		content.WriteString(styleMuted.Render(fmt.Sprintf("  ↓ %d more below", len(m.items)-endIdx)))
		content.WriteString("\n")
	}

	if len(m.items) == 0 {
		if m.loading {
			content.WriteString(fmt.Sprintf("  %s Loading lights...", m.spinner.View()))
		} else {
			content.WriteString(styleMuted.Render("  No lights found"))
		}
		content.WriteString("\n")
	}

	// Calculate content height (total height minus header, status, help)
	contentHeight := m.height - 5 // header(1) + search area(1) + blank(1) + status(1) + help(1)
	if m.searchMode || m.searchQuery != "" {
		contentHeight -= 1
	}
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Constrain content to fixed height to prevent overflow
	contentStr := content.String()
	contentStyle := lipgloss.NewStyle().Height(contentHeight).MaxHeight(contentHeight)

	// Layout with panel
	if showPanelNow {
		panel := m.renderPanel(panelWidth)
		// Set fixed width on content to prevent panel from shifting during loading
		contentStyle = contentStyle.Width(contentWidth)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, contentStyle.Render(contentStr), "  ", panel))
	} else {
		b.WriteString(contentStyle.Render(contentStr))
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	// Help bar
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m MainModel) renderRoomHeader(room *models.Room, selected bool) string {
	// Cursor - always same width character
	cursor := styleMuted.Render("  ")
	if selected {
		cursor = styleSelected.Render("> ")
	}

	// Room name (always use styleRoomName to preserve MarginTop and avoid flicker)
	nameStyle := styleRoomName

	// Count lights on
	lightsOn := 0
	totalBrightness := 0
	for _, light := range room.Lights {
		if light.On {
			lightsOn++
			totalBrightness += light.BrightnessPct()
		}
	}

	// Room summary
	summary := fmt.Sprintf("(%d/%d on", lightsOn, len(room.Lights))
	if lightsOn > 0 {
		avgBrightness := totalBrightness / lightsOn
		summary += fmt.Sprintf(" • %d%%", avgBrightness)
	}
	summary += ")"

	return fmt.Sprintf("%s%s %s", cursor, nameStyle.Render(room.Name), styleMuted.Render(summary))
}

func (m MainModel) renderLightRow(light *models.Light, selected bool, width int) string {
	// Cursor - always same width character
	cursor := styleMuted.Render("  ")
	if selected {
		cursor = styleSelected.Render("> ")
	}

	// Status icon
	icon := styleLightOff.Render("○")
	if light.On {
		icon = styleLightOn.Render("●")
	}

	// Calculate layout dynamically based on available width
	// Fixed parts: cursor(2) + icon(1) + space(1) + spaces(2) + space(1) + pct(4) + color(2) = 13
	fixedParts := 13
	availableForNameAndBar := width - fixedParts

	// Split available space: ~60% for name, ~40% for bar
	barWidth := availableForNameAndBar * 35 / 100
	if barWidth < 8 {
		barWidth = 8
	}
	if barWidth > 20 {
		barWidth = 20
	}

	nameWidth := availableForNameAndBar - barWidth
	if nameWidth < 10 {
		nameWidth = 10
	}
	if nameWidth > 45 {
		nameWidth = 45
	}

	// Name
	nameStyle := styleLightNameDim
	if light.On {
		nameStyle = styleLightName
	}
	if selected {
		nameStyle = styleSelected
	}
	name := nameStyle.Render(truncate(light.Name, nameWidth))

	// Brightness bar
	bar := m.renderBrightnessBar(light.BrightnessPct(), light.On, barWidth)

	// Percentage
	pct := styleBrightness.Render(fmt.Sprintf("%3d%%", light.BrightnessPct()))

	// Color indicator
	colorInd := ""
	if light.Color != nil && light.On {
		r, g, bl := light.Color.RGB()
		colorInd = lipgloss.NewStyle().
			Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, bl))).
			Render(" ◆")
	}

	return fmt.Sprintf("%s%s %s  %s %s%s", cursor, icon, name, bar, pct, colorInd)
}

func (m MainModel) renderBrightnessBar(brightness int, on bool, width int) string {
	if !on || brightness == 0 {
		return lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("─", width))
	}

	filled := (brightness * width) / 100
	if brightness > 0 && filled == 0 {
		filled = 1
	}

	// Gradient from dim to bright
	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			// Color intensity based on position
			intensity := 100 + (i * 155 / width)
			color := lipgloss.Color(fmt.Sprintf("#%02X%02X00", intensity, intensity/2))
			bar.WriteString(lipgloss.NewStyle().Foreground(color).Render("█"))
		} else {
			bar.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render("─"))
		}
	}
	return bar.String()
}

func (m MainModel) renderPanel(panelWidth int) string {
	// Show loading state in panel to avoid flicker
	if m.loading {
		return stylePanel.Width(panelWidth - 4).Render(m.spinner.View() + " Loading...")
	}

	// Check if room is selected
	if m.IsRoomSelected() {
		return m.renderRoomPanel(panelWidth)
	}

	light := m.SelectedLight()
	if light == nil {
		return stylePanel.Width(panelWidth - 4).Render(styleMuted.Render("No selection"))
	}

	// Bar width is panel width minus padding (2 on each side) minus label space
	barWidth := panelWidth - 10
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 25 {
		barWidth = 25
	}

	var content strings.Builder

	// Title
	content.WriteString(styleSelected.Render(light.Name))
	content.WriteString("\n\n")

	// Status
	status := styleLightOff.Render("○ Off")
	if light.On {
		status = styleLightOn.Render("● On")
	}
	content.WriteString(status)
	content.WriteString("\n\n")

	// Brightness
	content.WriteString(styleMuted.Render("Brightness: "))
	content.WriteString(fmt.Sprintf("%d%%\n", light.BrightnessPct()))
	content.WriteString(m.renderBrightnessBar(light.BrightnessPct(), light.On, barWidth))
	content.WriteString("\n\n")

	// Color mode display
	if light.Color != nil {
		// For the color preview, show color at full brightness so it's visible
		r, g, bl := getColorPreview(light.Color)
		colorBox := lipgloss.NewStyle().
			Background(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, bl))).
			Render("    ")

		switch light.Color.Mode {
		case models.ColorModeColorTemp:
			// Temperature mode
			content.WriteString(styleMuted.Render("Mode: "))
			content.WriteString("Temperature\n\n")

			if light.Color.Mirek > 0 {
				kelvin := 1000000 / int(light.Color.Mirek)
				content.WriteString(styleMuted.Render("Temp: "))
				content.WriteString(fmt.Sprintf("%dK\n", kelvin))

				// Temperature bar (153=cold to 500=warm)
				content.WriteString(m.renderTempBar(int(light.Color.Mirek), barWidth))
				content.WriteString("\n")
				content.WriteString(styleMuted.Render("     cool ← → warm\n"))
			}

			content.WriteString("\n")
			content.WriteString(styleMuted.Render("Color: "))
			content.WriteString(colorBox)

		case models.ColorModeHS:
			// HS Color mode
			content.WriteString(styleMuted.Render("Mode: "))
			content.WriteString("Color (HS)\n\n")

			// Hue (convert from 0-65535 to 0-360°)
			hueDeg := int(float64(light.Color.Hue) / 65535.0 * 360.0)
			content.WriteString(styleMuted.Render("Hue: "))
			content.WriteString(fmt.Sprintf("%d°\n", hueDeg))
			content.WriteString(m.renderHueBar(hueDeg, barWidth))
			content.WriteString("\n\n")

			// Saturation (convert from 0-254 to 0-100%)
			satPct := int(float64(light.Color.Saturation) / 254.0 * 100.0)
			content.WriteString(styleMuted.Render("Saturation: "))
			content.WriteString(fmt.Sprintf("%d%%\n", satPct))
			content.WriteString(m.renderSatBar(satPct, hueDeg, barWidth))
			content.WriteString("\n\n")

			content.WriteString(styleMuted.Render("Color: "))
			content.WriteString(colorBox)

		case models.ColorModeXY:
			// XY Color mode - convert to HS for display
			content.WriteString(styleMuted.Render("Mode: "))
			content.WriteString("Color (XY)\n\n")

			// Convert RGB to HSV for display
			hueDeg, satPct := rgbToHueSat(r, g, bl)

			content.WriteString(styleMuted.Render("Hue: "))
			content.WriteString(fmt.Sprintf("%d°\n", hueDeg))
			content.WriteString(m.renderHueBar(hueDeg, barWidth))
			content.WriteString("\n\n")

			content.WriteString(styleMuted.Render("Saturation: "))
			content.WriteString(fmt.Sprintf("%d%%\n", satPct))
			content.WriteString(m.renderSatBar(satPct, hueDeg, barWidth))
			content.WriteString("\n\n")

			content.WriteString(styleMuted.Render("Color: "))
			content.WriteString(colorBox)

		default:
			content.WriteString(styleMuted.Render("Color: "))
			content.WriteString(colorBox)
		}
	}

	// Room
	if room := m.SelectedRoom(); room != nil {
		content.WriteString("\n\n")
		content.WriteString(styleMuted.Render("Room: "))
		content.WriteString(room.Name)
	}

	// Use panel width minus border padding
	return stylePanel.Width(panelWidth - 4).Render(content.String())
}

func (m MainModel) renderRoomPanel(panelWidth int) string {
	room := m.SelectedRoom()
	if room == nil {
		return stylePanel.Width(panelWidth - 4).Render(styleMuted.Render("No room selected"))
	}

	// Bar width scales with panel
	barWidth := panelWidth - 10
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 25 {
		barWidth = 25
	}

	var content strings.Builder

	// Title
	content.WriteString(styleSelected.Render(room.Name))
	content.WriteString("\n\n")

	// Status
	lightsOn := 0
	totalBrightness := 0
	for _, light := range room.Lights {
		if light.On {
			lightsOn++
			totalBrightness += light.BrightnessPct()
		}
	}

	if lightsOn == 0 {
		content.WriteString(styleLightOff.Render("○ All Off"))
	} else if lightsOn == len(room.Lights) {
		content.WriteString(styleLightOn.Render("● All On"))
	} else {
		content.WriteString(styleLightOn.Render(fmt.Sprintf("● %d/%d On", lightsOn, len(room.Lights))))
	}
	content.WriteString("\n\n")

	// Average brightness
	if lightsOn > 0 {
		avgBrightness := totalBrightness / lightsOn
		content.WriteString(styleMuted.Render("Avg Brightness: "))
		content.WriteString(fmt.Sprintf("%d%%\n", avgBrightness))
		content.WriteString(m.renderBrightnessBar(avgBrightness, true, barWidth))
		content.WriteString("\n\n")
	} else {
		content.WriteString(styleMuted.Render("Avg Brightness: "))
		content.WriteString("--\n")
		content.WriteString(m.renderBrightnessBar(0, false, barWidth))
		content.WriteString("\n\n")
	}

	// Lights list - scale max items with height
	content.WriteString(styleMuted.Render("Lights:\n"))
	maxLights := 8
	maxNameLen := panelWidth - 8
	if maxNameLen < 12 {
		maxNameLen = 12
	}
	for i, light := range room.Lights {
		if i >= maxLights {
			content.WriteString(fmt.Sprintf("  ... +%d more\n", len(room.Lights)-maxLights))
			break
		}
		icon := styleLightOff.Render("○")
		if light.On {
			icon = styleLightOn.Render("●")
		}
		name := light.Name
		if len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "…"
		}
		content.WriteString(fmt.Sprintf("  %s %s\n", icon, name))
	}

	// Controls hint
	content.WriteString("\n")
	content.WriteString(styleMuted.Render("←→ dim • space toggle"))

	return stylePanel.Width(panelWidth - 4).Render(content.String())
}

func (m MainModel) renderTempBar(mirek int, width int) string {
	// Map mirek 153-500 to 0-width
	// 153 = cool (left), 500 = warm (right)
	pos := (mirek - 153) * width / (500 - 153)
	if pos < 0 {
		pos = 0
	}
	if pos >= width {
		pos = width - 1
	}

	var bar strings.Builder
	for i := 0; i < width; i++ {
		// Gradient from cool blue to warm orange
		ratio := float64(i) / float64(width)
		r := uint8(255 * ratio)
		g := uint8(180 - 80*ratio)
		b := uint8(255 * (1 - ratio))

		char := "─"
		if i == pos {
			char = "●"
		}
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))).Render(char))
	}
	return bar.String()
}

func (m MainModel) renderHueBar(hueDeg int, width int) string {
	pos := hueDeg * width / 360
	if pos >= width {
		pos = width - 1
	}

	var bar strings.Builder
	for i := 0; i < width; i++ {
		// Rainbow gradient
		hue := float64(i) / float64(width) * 360.0
		r, g, b := hueToRGB(hue)

		char := "─"
		if i == pos {
			char = "●"
		}
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))).Render(char))
	}
	return bar.String()
}

func (m MainModel) renderSatBar(satPct int, hueDeg int, width int) string {
	pos := satPct * width / 100
	if pos >= width {
		pos = width - 1
	}

	// Get the hue color at full saturation
	fullR, fullG, fullB := hueToRGB(float64(hueDeg))

	var bar strings.Builder
	for i := 0; i < width; i++ {
		// Gradient from white/gray to full color
		ratio := float64(i) / float64(width)
		r := uint8(255 - ratio*(255-float64(fullR)))
		g := uint8(255 - ratio*(255-float64(fullG)))
		b := uint8(255 - ratio*(255-float64(fullB)))

		char := "─"
		if i == pos {
			char = "●"
		}
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))).Render(char))
	}
	return bar.String()
}

func hueToRGB(hue float64) (r, g, b uint8) {
	h := hue / 60.0
	x := 1 - abs(mod(h, 2)-1)

	var rf, gf, bf float64
	switch int(h) % 6 {
	case 0:
		rf, gf, bf = 1, x, 0
	case 1:
		rf, gf, bf = x, 1, 0
	case 2:
		rf, gf, bf = 0, 1, x
	case 3:
		rf, gf, bf = 0, x, 1
	case 4:
		rf, gf, bf = x, 0, 1
	case 5:
		rf, gf, bf = 1, 0, x
	}

	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func mod(a, b float64) float64 {
	return a - b*float64(int(a/b))
}

// getColorPreview returns RGB values for displaying a color preview at full brightness
func getColorPreview(c *models.Color) (r, g, b uint8) {
	switch c.Mode {
	case models.ColorModeHS:
		// Use HSV with full brightness for preview
		return hsvToRGBFull(c.Hue, c.Saturation)
	case models.ColorModeXY:
		// Convert XY to RGB at full brightness
		return xyToRGBFull(c.X, c.Y)
	case models.ColorModeColorTemp:
		// Use full brightness for temp preview
		return mirekToRGBFull(c.Mirek)
	default:
		return 255, 255, 255
	}
}

func hsvToRGBFull(hue uint16, sat uint8) (r, g, b uint8) {
	h := float64(hue) / 65535.0 * 360.0
	s := float64(sat) / 254.0
	v := 1.0 // Full brightness

	if s == 0 {
		val := uint8(v * 255)
		return val, val, val
	}

	h = float64(int(h) % 360)
	h /= 60
	i := int(h)
	f := h - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var rf, gf, bf float64
	switch i {
	case 0:
		rf, gf, bf = v, t, p
	case 1:
		rf, gf, bf = q, v, p
	case 2:
		rf, gf, bf = p, v, t
	case 3:
		rf, gf, bf = p, q, v
	case 4:
		rf, gf, bf = t, p, v
	default:
		rf, gf, bf = v, p, q
	}

	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}

func xyToRGBFull(x, y float64) (r, g, b uint8) {
	if y == 0 {
		return 255, 255, 255
	}

	// Use Y=1 for full brightness
	Y := 1.0
	X := (Y / y) * x
	Z := (Y / y) * (1 - x - y)

	// XYZ to RGB
	rf := X*1.656492 - Y*0.354851 - Z*0.255038
	gf := -X*0.707196 + Y*1.655397 + Z*0.036152
	bf := X*0.051713 - Y*0.121364 + Z*1.011530

	// Gamma correction and clamp
	rf = gammaCorrect(rf)
	gf = gammaCorrect(gf)
	bf = gammaCorrect(bf)

	return clamp255(rf), clamp255(gf), clamp255(bf)
}

func mirekToRGBFull(mirek uint16) (r, g, b uint8) {
	if mirek == 0 {
		return 255, 255, 255
	}
	kelvin := 1000000.0 / float64(mirek)
	temp := kelvin / 100.0

	var rf, gf, bf float64

	if temp <= 66 {
		rf = 255
	} else {
		rf = temp - 60
		rf = 329.698727446 * math.Pow(rf, -0.1332047592)
		rf = math.Max(0, math.Min(255, rf))
	}

	if temp <= 66 {
		gf = 99.4708025861*math.Log(temp) - 161.1195681661
		gf = math.Max(0, math.Min(255, gf))
	} else {
		gf = temp - 60
		gf = 288.1221695283 * math.Pow(gf, -0.0755148492)
		gf = math.Max(0, math.Min(255, gf))
	}

	if temp >= 66 {
		bf = 255
	} else if temp <= 19 {
		bf = 0
	} else {
		bf = temp - 10
		bf = 138.5177312231*math.Log(bf) - 305.0447927307
		bf = math.Max(0, math.Min(255, bf))
	}

	return uint8(rf), uint8(gf), uint8(bf)
}

func gammaCorrect(value float64) float64 {
	if value <= 0.0031308 {
		return 12.92 * value
	}
	return 1.055*math.Pow(value, 1.0/2.4) - 0.055
}

func clamp255(value float64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 255
	}
	return uint8(value * 255)
}

func rgbToHueSat(r, g, b uint8) (hueDeg, satPct int) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	maxVal := rf
	if gf > maxVal {
		maxVal = gf
	}
	if bf > maxVal {
		maxVal = bf
	}

	minVal := rf
	if gf < minVal {
		minVal = gf
	}
	if bf < minVal {
		minVal = bf
	}

	delta := maxVal - minVal

	// Saturation
	if maxVal == 0 {
		satPct = 0
	} else {
		satPct = int((delta / maxVal) * 100)
	}

	// Hue
	if delta == 0 {
		hueDeg = 0
	} else {
		var hf float64
		switch {
		case rf == maxVal:
			hf = (gf - bf) / delta
			if gf < bf {
				hf += 6
			}
		case gf == maxVal:
			hf = 2 + (bf-rf)/delta
		default:
			hf = 4 + (rf-gf)/delta
		}
		hueDeg = int(hf * 60)
	}

	return hueDeg, satPct
}

func (m MainModel) renderStatusBar() string {
	// Count lights on and active rooms
	lightsOn := 0
	totalLights := 0
	activeRooms := make(map[string]bool)

	for _, item := range m.items {
		if !item.isRoom && item.light != nil {
			totalLights++
			if item.light.On {
				lightsOn++
				if item.room != nil {
					activeRooms[item.room.ID] = true
				}
			}
		}
	}
	roomsActive := len(activeRooms)
	totalRooms := len(m.rooms)

	// Build status string
	status := fmt.Sprintf("%d/%d lights on", lightsOn, totalLights)
	if totalRooms > 0 {
		status += fmt.Sprintf(" • %d/%d rooms active", roomsActive, totalRooms)
	}

	return styleMuted.Render(status)
}

func (m MainModel) renderHelp() string {
	keys := []string{
		styleHelpKey.Render("↑↓") + " nav",
		styleHelpKey.Render("pgup/dn") + " scroll",
		styleHelpKey.Render("←→") + " dim",
		styleHelpKey.Render("space") + " toggle",
		styleHelpKey.Render("w/c") + " temp",
		styleHelpKey.Render("[]") + " hue",
		styleHelpKey.Render("-/=") + " sat",
		styleHelpKey.Render("a/x") + " room",
		styleHelpKey.Render("s") + " scenes",
		styleHelpKey.Render("q") + " quit",
	}

	// For narrow terminals, show fewer keys
	if m.width < 60 {
		keys = []string{
			styleHelpKey.Render("↑↓") + " nav",
			styleHelpKey.Render("space") + " toggle",
			styleHelpKey.Render("q") + " quit",
		}
	} else if m.width < 90 {
		keys = []string{
			styleHelpKey.Render("↑↓") + " nav",
			styleHelpKey.Render("←→") + " dim",
			styleHelpKey.Render("space") + " toggle",
			styleHelpKey.Render("s") + " scenes",
			styleHelpKey.Render("q") + " quit",
		}
	}

	return styleHelp.Render(strings.Join(keys, "  "))
}

// Commands
func (m MainModel) toggleLightCmd(bridge api.BridgeClient, lightID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if bridge == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bridge.SetLightOn(ctx, lightID, on); err != nil {
			return messages.ErrorMsg{Err: err}
		}
		return nil
	}
}

func (m MainModel) setBrightnessCmd(bridge api.BridgeClient, lightID string, brightness int) tea.Cmd {
	return func() tea.Msg {
		if bridge == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bridge.SetLightBrightness(ctx, lightID, brightness); err != nil {
			return messages.ErrorMsg{Err: err}
		}
		return nil
	}
}

func (m MainModel) setColorTempCmd(bridge api.BridgeClient, lightID string, mirek int) tea.Cmd {
	return func() tea.Msg {
		if bridge == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bridge.SetLightColorTemp(ctx, lightID, mirek); err != nil {
			return messages.ErrorMsg{Err: err}
		}
		return nil
	}
}

func (m MainModel) setColorHSCmd(bridge api.BridgeClient, lightID string, hue uint16, sat uint8) tea.Cmd {
	return func() tea.Msg {
		if bridge == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bridge.SetLightColorHS(ctx, lightID, hue, sat); err != nil {
			return messages.ErrorMsg{Err: err}
		}
		return nil
	}
}

func (m MainModel) setGroupOnCmd(bridge api.BridgeClient, groupID string, on bool) tea.Cmd {
	return func() tea.Msg {
		if bridge == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := bridge.SetGroupedLightOn(ctx, groupID, on); err != nil {
			return messages.ErrorMsg{Err: err}
		}
		return nil
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s + strings.Repeat(" ", maxLen-len(s))
	}
	return s[:maxLen-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func brightnessFromKey(key string) int {
	switch key {
	case "0":
		return 100
	case "1":
		return 10
	case "2":
		return 20
	case "3":
		return 30
	case "4":
		return 40
	case "5":
		return 50
	case "6":
		return 60
	case "7":
		return 70
	case "8":
		return 80
	case "9":
		return 90
	default:
		return -1
	}
}

var styleMuted = lipgloss.NewStyle().Foreground(colorMuted)
