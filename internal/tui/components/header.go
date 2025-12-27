package components

import (
	"strings"

	"github.com/angristan/hue-tui/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderHeader renders the application header
func RenderHeader(width int, status string) string {
	title := " Hue CLI "

	// Create gradient-like effect with the lavender theme
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		Background(styles.ColorPrimary).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(styles.ColorSuccess).
		Padding(0, 1)

	if status == "" {
		status = "Disconnected"
		statusStyle = statusStyle.Foreground(styles.ColorError)
	}

	left := titleStyle.Render(title)
	right := statusStyle.Render(status)

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacing := width - leftWidth - rightWidth

	if spacing < 0 {
		spacing = 0
	}

	headerBg := lipgloss.NewStyle().
		Background(styles.ColorSurface).
		Width(width)

	return headerBg.Render(left + strings.Repeat(" ", spacing) + right)
}
