package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette - Lavender theme matching Python original
var (
	// Primary colors
	ColorPrimary    = lipgloss.Color("#B794F4") // Lavender
	ColorSecondary  = lipgloss.Color("#9F7AEA") // Darker lavender
	ColorAccent     = lipgloss.Color("#E9D8FD") // Light lavender
	ColorBackground = lipgloss.Color("#1A1A2E") // Dark background
	ColorSurface    = lipgloss.Color("#2D2D44") // Surface color
	ColorSurfaceAlt = lipgloss.Color("#3D3D5C") // Alternate surface

	// Text colors
	ColorText        = lipgloss.Color("#FAFAFA") // Primary text
	ColorTextMuted   = lipgloss.Color("#A0A0B0") // Muted text
	ColorTextDim     = lipgloss.Color("#6B6B80") // Dim text
	ColorTextInverse = lipgloss.Color("#1A1A2E") // Inverse text

	// State colors
	ColorSuccess = lipgloss.Color("#68D391") // Green
	ColorWarning = lipgloss.Color("#F6E05E") // Yellow
	ColorError   = lipgloss.Color("#FC8181") // Red
	ColorInfo    = lipgloss.Color("#63B3ED") // Blue

	// Light states
	ColorLightOn  = lipgloss.Color("#FBBF24") // Warm yellow for on
	ColorLightOff = lipgloss.Color("#4A4A5A") // Gray for off

	// Brightness bar colors (gradient from dim to bright)
	ColorBrightness1  = lipgloss.Color("#3D3D5C")
	ColorBrightness2  = lipgloss.Color("#4A4A6A")
	ColorBrightness3  = lipgloss.Color("#5A5A7A")
	ColorBrightness4  = lipgloss.Color("#6A6A8A")
	ColorBrightness5  = lipgloss.Color("#7A7A9A")
	ColorBrightness6  = lipgloss.Color("#8A8AAA")
	ColorBrightness7  = lipgloss.Color("#9A9ABA")
	ColorBrightness8  = lipgloss.Color("#AAAACA")
	ColorBrightness9  = lipgloss.Color("#BABADA")
	ColorBrightness10 = lipgloss.Color("#FBBF24")
)

// Styles for various UI components
var (
	// Base styles
	StyleBase = lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorText)

	// Header styles
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorSurface).
			Padding(0, 2).
			MarginBottom(1)

	StyleHeaderGradient = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorText).
				Background(ColorPrimary).
				Padding(0, 2)

	// Room panel styles
	StyleRoomPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSurface).
			Padding(0, 1).
			MarginBottom(1)

	StyleRoomTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			MarginBottom(1)

	// Light card styles
	StyleLightCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSurfaceAlt).
			Padding(0, 1).
			MarginRight(1).
			MarginBottom(1)

	StyleLightCardSelected = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1).
				MarginRight(1).
				MarginBottom(1)

	StyleLightName = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleLightNameDim = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	// Status indicators
	StyleStatusOn = lipgloss.NewStyle().
			Foreground(ColorLightOn).
			Bold(true)

	StyleStatusOff = lipgloss.NewStyle().
			Foreground(ColorLightOff)

	// Brightness bar styles
	StyleBrightnessBarEmpty = lipgloss.NewStyle().
				Foreground(ColorSurfaceAlt)

	// Button styles
	StyleButton = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSurface).
			Padding(0, 2).
			MarginRight(1)

	StyleButtonFocused = lipgloss.NewStyle().
				Foreground(ColorTextInverse).
				Background(ColorPrimary).
				Padding(0, 2).
				MarginRight(1)

	// Modal styles
	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Background(ColorSurface).
			Padding(1, 2)

	StyleModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Input styles
	StyleInput = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSurfaceAlt).
			Padding(0, 1)

	StyleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Help styles
	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			MarginTop(1)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Side panel styles
	StyleSidePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			Width(30)

	StyleSidePanelTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				MarginBottom(1)

	// Slider styles
	StyleSliderTrack = lipgloss.NewStyle().
				Foreground(ColorSurfaceAlt)

	StyleSliderFill = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Color preset styles
	StyleColorPreset = lipgloss.NewStyle().
				Padding(0, 1).
				MarginRight(1)

	StyleColorPresetSelected = lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(ColorPrimary).
					Padding(0, 1).
					MarginRight(1)

	// Scene list styles
	StyleSceneItem = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	StyleSceneItemSelected = lipgloss.NewStyle().
				Foreground(ColorTextInverse).
				Background(ColorPrimary).
				Padding(0, 1)

	// Search bar styles
	StyleSearchBar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorSurfaceAlt).
			Padding(0, 1).
			MarginBottom(1)

	StyleSearchBarFocused = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1).
				MarginBottom(1)

	// Loading/spinner styles
	StyleSpinner = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Error styles
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Success styles
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// Text muted style
	StyleTextMuted = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	// Primary style
	StylePrimary = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)
)

// GetBrightnessColor returns the appropriate color for a brightness segment
func GetBrightnessColor(segment int, brightness int) lipgloss.Color {
	threshold := segment * 10
	if brightness >= threshold {
		switch segment {
		case 1:
			return ColorBrightness1
		case 2:
			return ColorBrightness2
		case 3:
			return ColorBrightness3
		case 4:
			return ColorBrightness4
		case 5:
			return ColorBrightness5
		case 6:
			return ColorBrightness6
		case 7:
			return ColorBrightness7
		case 8:
			return ColorBrightness8
		case 9:
			return ColorBrightness9
		case 10:
			return ColorBrightness10
		}
	}
	return ColorSurfaceAlt
}
