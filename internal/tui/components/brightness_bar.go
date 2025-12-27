package components

import (
	"github.com/angristan/hue-tui/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// BrightnessBarStyle defines the style options for brightness bars
type BrightnessBarStyle struct {
	Width       int
	ShowPercent bool
	Vertical    bool
}

// DefaultBrightnessBarStyle returns the default brightness bar style
func DefaultBrightnessBarStyle() BrightnessBarStyle {
	return BrightnessBarStyle{
		Width:       10,
		ShowPercent: false,
		Vertical:    false,
	}
}

// RenderBrightnessBarStyled renders a brightness bar with custom style
func RenderBrightnessBarStyled(brightness int, on bool, style BrightnessBarStyle) string {
	if !on {
		empty := ""
		for i := 0; i < style.Width; i++ {
			empty += "─"
		}
		return styles.StyleBrightnessBarEmpty.Render(empty)
	}

	segments := (brightness * style.Width) / 100
	if brightness > 0 && segments == 0 {
		segments = 1
	}

	var result string
	for i := 1; i <= style.Width; i++ {
		segmentBrightness := (i * 100) / style.Width
		if i <= segments {
			color := getBrightnessColorForSegment(i, style.Width, brightness)
			result += lipgloss.NewStyle().Foreground(color).Render("█")
		} else {
			result += styles.StyleBrightnessBarEmpty.Render("─")
		}
		_ = segmentBrightness
	}

	return result
}

// getBrightnessColorForSegment returns the color for a specific segment
func getBrightnessColorForSegment(segment, total, brightness int) lipgloss.Color {
	// Map segment to 1-10 scale
	mappedSegment := ((segment * 10) / total)
	if mappedSegment < 1 {
		mappedSegment = 1
	}
	if mappedSegment > 10 {
		mappedSegment = 10
	}

	return styles.GetBrightnessColor(mappedSegment, brightness)
}

// RenderVerticalBrightnessBar renders a vertical brightness bar
func RenderVerticalBrightnessBar(brightness int, on bool, height int) string {
	if !on {
		result := ""
		for i := 0; i < height; i++ {
			result += styles.StyleBrightnessBarEmpty.Render("─") + "\n"
		}
		return result
	}

	segments := (brightness * height) / 100
	if brightness > 0 && segments == 0 {
		segments = 1
	}

	var result string
	for i := height; i >= 1; i-- {
		if i <= segments {
			color := getBrightnessColorForSegment(i, height, brightness)
			result += lipgloss.NewStyle().Foreground(color).Render("█") + "\n"
		} else {
			result += styles.StyleBrightnessBarEmpty.Render("─") + "\n"
		}
	}

	return result
}
