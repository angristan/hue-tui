package components

import (
	"fmt"
	"strings"

	"github.com/angristan/hue-tui/internal/models"
	"github.com/angristan/hue-tui/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderLightCard renders a single light card
func RenderLightCard(light *models.Light, selected bool, maxWidth int) string {
	// Status indicator
	statusIcon := "○"
	statusStyle := styles.StyleStatusOff
	if light.On {
		statusIcon = "●"
		statusStyle = styles.StyleStatusOn
	}

	// Color indicator (if color light)
	colorIndicator := ""
	if light.Color != nil && light.On {
		r, g, bl := light.Color.RGB()
		colorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, bl)))
		colorIndicator = colorStyle.Render(" ◆")
	}

	// Name styling based on state
	nameStyle := styles.StyleLightName
	if !light.On {
		nameStyle = styles.StyleLightNameDim
	}

	// Build the card content
	status := statusStyle.Render(statusIcon)
	name := nameStyle.Render(light.Name)
	brightness := RenderBrightnessBar(light.BrightnessPct(), light.On)

	// First line: status + name + color
	line1 := fmt.Sprintf("%s %s%s", status, name, colorIndicator)

	// Second line: brightness bar
	line2 := fmt.Sprintf("  %s %3d%%", brightness, light.BrightnessPct())

	content := line1 + "\n" + line2

	// Card style based on selection
	cardStyle := styles.StyleLightCard
	if selected {
		cardStyle = styles.StyleLightCardSelected
	}

	// Ensure minimum width for the card
	cardWidth := maxWidth / 3
	if cardWidth < 25 {
		cardWidth = 25
	}
	if cardWidth > 40 {
		cardWidth = 40
	}

	return cardStyle.Width(cardWidth).Render(content)
}

// RenderBrightnessBar renders a brightness indicator bar
func RenderBrightnessBar(brightness int, on bool) string {
	if !on {
		// All empty when off
		return styles.StyleBrightnessBarEmpty.Render("──────────")
	}

	var b strings.Builder
	segments := brightness / 10
	if brightness > 0 && segments == 0 {
		segments = 1
	}

	for i := 1; i <= 10; i++ {
		if i <= segments {
			color := styles.GetBrightnessColor(i, brightness)
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render("█"))
		} else {
			b.WriteString(styles.StyleBrightnessBarEmpty.Render("─"))
		}
	}

	return b.String()
}

// RenderRoomHeader renders a room header
func RenderRoomHeader(room *models.Room) string {
	var b strings.Builder

	// Room name
	name := styles.StyleRoomTitle.Render(room.Name)

	// Room status
	status := ""
	if room.AllOn {
		status = styles.StyleStatusOn.Render(" (all on)")
	} else if room.AnyOn {
		status = styles.StyleStatusOn.Render(fmt.Sprintf(" (%d on)", countOn(room)))
	}

	// Light count
	count := styles.StyleHelp.Render(fmt.Sprintf(" [%d lights]", len(room.Lights)))

	b.WriteString(name)
	b.WriteString(status)
	b.WriteString(count)

	return b.String()
}

func countOn(room *models.Room) int {
	count := 0
	for _, light := range room.Lights {
		if light.On {
			count++
		}
	}
	return count
}
