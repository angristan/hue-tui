package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/angristan/hue-tui/internal/models"
	"github.com/angristan/hue-tui/internal/tui/messages"
	"github.com/angristan/hue-tui/internal/tui/styles"
)

// ScenesModel is the scenes modal model
type ScenesModel struct {
	scenes   []*models.Scene
	rooms    []*models.Room
	selected int

	// Grouped scenes by room
	groupedScenes map[string][]*models.Scene
	roomOrder     []string

	// Flat list for navigation
	flatList []sceneItem

	// Window size
	width  int
	height int
}

type sceneItem struct {
	scene    *models.Scene
	isHeader bool
	roomName string
}

// NewScenesModel creates a new scenes screen model
func NewScenesModel() ScenesModel {
	return ScenesModel{}
}

// SetSize sets the terminal size
func (m *ScenesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetScenes sets the scene data
func (m *ScenesModel) SetScenes(scenes []*models.Scene, rooms []*models.Room) {
	m.scenes = scenes
	m.rooms = rooms
	m.groupedScenes = models.ScenesByRoom(scenes)

	// Build room order and flat list
	m.roomOrder = nil
	m.flatList = nil

	roomNames := make(map[string]string)
	for _, room := range rooms {
		roomNames[room.ID] = room.Name
	}

	for _, room := range rooms {
		if scenes, ok := m.groupedScenes[room.ID]; ok && len(scenes) > 0 {
			m.roomOrder = append(m.roomOrder, room.ID)

			// Add room header
			m.flatList = append(m.flatList, sceneItem{
				isHeader: true,
				roomName: room.Name,
			})

			// Add scenes
			for _, scene := range scenes {
				m.flatList = append(m.flatList, sceneItem{
					scene:    scene,
					roomName: room.Name,
				})
			}
		}
	}

	// Reset selection
	m.selected = 0
	// Skip to first scene (not header)
	for i, item := range m.flatList {
		if !item.isHeader {
			m.selected = i
			break
		}
	}
}

// Update handles messages
func (m ScenesModel) Update(msg tea.Msg) (ScenesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "s", "q":
			return m, func() tea.Msg { return messages.HideScenesMsg{} }

		case "up", "k":
			m.movePrev()

		case "down", "j":
			m.moveNext()

		case "enter":
			if m.selected >= 0 && m.selected < len(m.flatList) {
				item := m.flatList[m.selected]
				if !item.isHeader && item.scene != nil {
					return m, func() tea.Msg {
						return messages.SceneActivatedMsg{SceneID: item.scene.ID}
					}
				}
			}
		}
	}

	return m, nil
}

func (m *ScenesModel) moveNext() {
	for i := m.selected + 1; i < len(m.flatList); i++ {
		if !m.flatList[i].isHeader {
			m.selected = i
			return
		}
	}
}

func (m *ScenesModel) movePrev() {
	for i := m.selected - 1; i >= 0; i-- {
		if !m.flatList[i].isHeader {
			m.selected = i
			return
		}
	}
}

// View renders the scenes modal
func (m ScenesModel) View() string {
	var b strings.Builder

	// Modal title
	b.WriteString(styles.StyleModalTitle.Render("Scenes"))
	b.WriteString("\n\n")

	// Scene list
	for i, item := range m.flatList {
		if item.isHeader {
			b.WriteString(styles.StyleRoomTitle.Render(item.roomName))
			b.WriteString("\n")
			continue
		}

		style := styles.StyleSceneItem
		cursor := "  "
		if i == m.selected {
			style = styles.StyleSceneItemSelected
			cursor = "> "
		}

		b.WriteString(cursor + style.Render(item.scene.Name) + "\n")
	}

	if len(m.flatList) == 0 {
		b.WriteString(styles.StyleTextMuted.Render("No scenes available"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.StyleHelp.Render("↑/↓ navigate • enter activate • esc close"))

	// Wrap in modal style - responsive width (60-80% of screen, 40-60 chars)
	content := b.String()
	modalWidth := m.width * 70 / 100
	if modalWidth < 40 {
		modalWidth = 40
	}
	if modalWidth > 60 {
		modalWidth = 60
	}
	modal := styles.StyleModal.Width(modalWidth).Render(content)

	// Center in screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
