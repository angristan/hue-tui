package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/angristan/hue-tui/internal/models"
	"github.com/angristan/hue-tui/internal/tui/styles"
)

// RenderRoomPanel renders a complete room panel with all its lights
func RenderRoomPanel(room *models.Room, selectedLightID string, width int) string {
	var b strings.Builder

	// Room header
	b.WriteString(RenderRoomHeader(room))
	b.WriteString("\n")

	// Render lights in a grid
	lightCards := make([]string, len(room.Lights))
	for i, light := range room.Lights {
		isSelected := light.ID == selectedLightID
		lightCards[i] = RenderLightCard(light, isSelected, width)
	}

	// Arrange cards in rows (3 per row)
	cardsPerRow := 3
	cardWidth := width / cardsPerRow

	for i := 0; i < len(lightCards); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(lightCards) {
			end = len(lightCards)
		}
		row := lightCards[i:end]
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, row...))
		b.WriteString("\n")
	}
	_ = cardWidth

	// Panel style
	return styles.StyleRoomPanel.Width(width - 4).Render(b.String())
}

// RenderRoomControl renders the room-level controls
func RenderRoomControl(room *models.Room, focused bool) string {
	var b strings.Builder

	// Room name with status
	nameStyle := styles.StyleRoomTitle
	if focused {
		nameStyle = nameStyle.Bold(true).Foreground(styles.ColorPrimary)
	}

	b.WriteString(nameStyle.Render(room.Name))

	// Toggle button
	toggleLabel := "All Off"
	toggleStyle := styles.StyleButton
	if room.AllOn {
		toggleLabel = "All Off"
	} else {
		toggleLabel = "All On"
	}
	if focused {
		toggleStyle = styles.StyleButtonFocused
	}

	b.WriteString("  ")
	b.WriteString(toggleStyle.Render(toggleLabel))

	// Brightness indicator
	if room.AnyOn {
		avgBrightness := room.AverageBrightness()
		b.WriteString("  ")
		b.WriteString(RenderBrightnessBar(avgBrightness, true))
	}

	return b.String()
}

// RenderLightList renders lights as a simple list (for narrow displays)
func RenderLightList(lights []*models.Light, selectedID string, width int) string {
	var b strings.Builder

	for _, light := range lights {
		isSelected := light.ID == selectedID

		// Status icon
		statusIcon := "○"
		statusStyle := styles.StyleStatusOff
		if light.On {
			statusIcon = "●"
			statusStyle = styles.StyleStatusOn
		}

		// Selection indicator
		cursor := "  "
		rowStyle := lipgloss.NewStyle()
		if isSelected {
			cursor = "> "
			rowStyle = rowStyle.Foreground(styles.ColorPrimary)
		}

		// Name
		nameStyle := styles.StyleLightName
		if !light.On {
			nameStyle = styles.StyleLightNameDim
		}
		if isSelected {
			nameStyle = nameStyle.Foreground(styles.ColorPrimary)
		}

		// Build row
		status := statusStyle.Render(statusIcon)
		name := nameStyle.Render(truncate(light.Name, width-20))
		brightness := RenderBrightnessBar(light.BrightnessPct(), light.On)

		row := cursor + status + " " + name
		// Pad to align brightness
		padding := width - 15 - lipgloss.Width(row)
		if padding > 0 {
			row += strings.Repeat(" ", padding)
		}
		row += " " + brightness

		b.WriteString(rowStyle.Render(row))
		b.WriteString("\n")
	}

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
