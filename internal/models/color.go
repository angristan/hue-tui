package models

import (
	"math"
)

// ColorMode represents how the color is being controlled
type ColorMode int

const (
	ColorModeNone ColorMode = iota
	ColorModeColorTemp
	ColorModeHS
	ColorModeXY
)

// Color represents a light's color state with multiple color space representations
type Color struct {
	// Hue: 0-65535 (maps to 0-360 degrees)
	Hue uint16
	// Saturation: 0-254
	Saturation uint8
	// Brightness: 0-254
	Brightness uint8
	// ColorTemp in Mirek (153 = cool/blue, 500 = warm/orange)
	// Mirek = 1,000,000 / Kelvin
	Mirek uint16
	// XY color coordinates (CIE 1931 color space)
	X, Y float64
	// Current color mode
	Mode ColorMode

	// Cached RGB values
	cachedR, cachedG, cachedB uint8
	cacheValid                bool
}

// RGB returns the color as RGB values (0-255 each)
func (c *Color) RGB() (r, g, b uint8) {
	if c.cacheValid {
		return c.cachedR, c.cachedG, c.cachedB
	}

	switch c.Mode {
	case ColorModeHS:
		r, g, b = c.hsvToRGB()
	case ColorModeXY:
		r, g, b = c.xyToRGB()
	case ColorModeColorTemp:
		r, g, b = c.mirekToRGB()
	default:
		// Default to white
		r, g, b = 255, 255, 255
	}

	c.cachedR, c.cachedG, c.cachedB = r, g, b
	c.cacheValid = true
	return r, g, b
}

// InvalidateCache clears the cached RGB values
func (c *Color) InvalidateCache() {
	c.cacheValid = false
}

// hsvToRGB converts HSV to RGB
// Hue: 0-65535 -> 0-360, Saturation: 0-254 -> 0-1, Brightness: 0-254 -> 0-1
func (c *Color) hsvToRGB() (r, g, b uint8) {
	// Normalize values
	h := float64(c.Hue) / 65535.0 * 360.0
	s := float64(c.Saturation) / 254.0
	v := float64(c.Brightness) / 254.0

	if s == 0 {
		// Achromatic (gray)
		val := uint8(v * 255)
		return val, val, val
	}

	h = math.Mod(h, 360)
	h /= 60
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var rf, gf, bf float64
	switch int(i) {
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

// xyToRGB converts CIE 1931 XY color space to RGB
// Uses the Wide RGB D65 conversion matrix with gamma correction
func (c *Color) xyToRGB() (r, g, b uint8) {
	x := c.X
	y := c.Y

	// Avoid division by zero
	if y == 0 {
		return 255, 255, 255
	}

	// Calculate XYZ
	// Using brightness as Y (luminance)
	brightness := float64(c.Brightness) / 254.0
	Y := brightness
	X := (Y / y) * x
	Z := (Y / y) * (1 - x - y)

	// Convert XYZ to RGB using Wide RGB D65 matrix
	// This matrix is optimized for Hue bulbs
	rf := X*1.656492 - Y*0.354851 - Z*0.255038
	gf := -X*0.707196 + Y*1.655397 + Z*0.036152
	bf := X*0.051713 - Y*0.121364 + Z*1.011530

	// Apply reverse gamma correction
	rf = reverseGamma(rf)
	gf = reverseGamma(gf)
	bf = reverseGamma(bf)

	// Clamp and convert to 0-255
	return clampTo255(rf), clampTo255(gf), clampTo255(bf)
}

// reverseGamma applies reverse gamma correction for sRGB
func reverseGamma(value float64) float64 {
	if value <= 0.0031308 {
		return 12.92 * value
	}
	return 1.055*math.Pow(value, 1.0/2.4) - 0.055
}

// clampTo255 clamps a float to 0-255 range and converts to uint8
func clampTo255(value float64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 255
	}
	return uint8(value * 255)
}

// mirekToRGB converts color temperature in Mirek to RGB
// Mirek range: 153 (cool/6500K) to 500 (warm/2000K)
func (c *Color) mirekToRGB() (r, g, b uint8) {
	// Convert Mirek to Kelvin: K = 1,000,000 / Mirek
	kelvin := 1000000.0 / float64(c.Mirek)

	// Algorithm based on Tanner Helland's work
	// http://www.tannerhelland.com/4435/convert-temperature-rgb-algorithm-code/
	temp := kelvin / 100.0

	var rf, gf, bf float64

	// Red
	if temp <= 66 {
		rf = 255
	} else {
		rf = temp - 60
		rf = 329.698727446 * math.Pow(rf, -0.1332047592)
		rf = clampFloat(rf, 0, 255)
	}

	// Green
	if temp <= 66 {
		gf = temp
		gf = 99.4708025861*math.Log(gf) - 161.1195681661
		gf = clampFloat(gf, 0, 255)
	} else {
		gf = temp - 60
		gf = 288.1221695283 * math.Pow(gf, -0.0755148492)
		gf = clampFloat(gf, 0, 255)
	}

	// Blue
	if temp >= 66 {
		bf = 255
	} else if temp <= 19 {
		bf = 0
	} else {
		bf = temp - 10
		bf = 138.5177312231*math.Log(bf) - 305.0447927307
		bf = clampFloat(bf, 0, 255)
	}

	// Apply brightness
	brightness := float64(c.Brightness) / 254.0
	rf *= brightness
	gf *= brightness
	bf *= brightness

	return uint8(rf), uint8(gf), uint8(bf)
}

// clampFloat clamps a float64 to a range
func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// SetFromRGB sets the color from RGB values
func (c *Color) SetFromRGB(r, g, b uint8) {
	c.Hue, c.Saturation, c.Brightness = rgbToHSV(r, g, b)
	c.Mode = ColorModeHS
	c.InvalidateCache()
}

// rgbToHSV converts RGB to Hue HSV format
// Returns: Hue (0-65535), Saturation (0-254), Value/Brightness (0-254)
func rgbToHSV(r, g, b uint8) (h uint16, s, v uint8) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	// Value/Brightness
	v = uint8(max * 254)

	if max == 0 {
		return 0, 0, v
	}

	// Saturation
	s = uint8((delta / max) * 254)

	if delta == 0 {
		return 0, s, v
	}

	// Hue
	var hf float64
	switch {
	case rf == max:
		hf = (gf - bf) / delta
		if gf < bf {
			hf += 6
		}
	case gf == max:
		hf = 2 + (bf-rf)/delta
	default:
		hf = 4 + (rf-gf)/delta
	}

	hf /= 6
	h = uint16(hf * 65535)

	return h, s, v
}

// RGBToXY converts RGB to CIE 1931 XY color space
func RGBToXY(r, g, b uint8) (x, y float64) {
	// Normalize RGB
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	// Apply gamma correction
	rf = applyGamma(rf)
	gf = applyGamma(gf)
	bf = applyGamma(bf)

	// Convert to XYZ using Wide RGB D65 matrix
	X := rf*0.664511 + gf*0.154324 + bf*0.162028
	Y := rf*0.283881 + gf*0.668433 + bf*0.047685
	Z := rf*0.000088 + gf*0.072310 + bf*0.986039

	// Calculate xy chromaticity
	sum := X + Y + Z
	if sum == 0 {
		return 0.3127, 0.3290 // D65 white point
	}

	x = X / sum
	y = Y / sum

	return x, y
}

// applyGamma applies gamma correction for sRGB
func applyGamma(value float64) float64 {
	if value > 0.04045 {
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	return value / 12.92
}

// NewColorFromHS creates a Color from Hue and Saturation values
func NewColorFromHS(hue uint16, saturation, brightness uint8) *Color {
	return &Color{
		Hue:        hue,
		Saturation: saturation,
		Brightness: brightness,
		Mode:       ColorModeHS,
	}
}

// NewColorFromXY creates a Color from XY coordinates
func NewColorFromXY(x, y float64, brightness uint8) *Color {
	return &Color{
		X:          x,
		Y:          y,
		Brightness: brightness,
		Mode:       ColorModeXY,
	}
}

// NewColorFromMirek creates a Color from color temperature
func NewColorFromMirek(mirek uint16, brightness uint8) *Color {
	return &Color{
		Mirek:      mirek,
		Brightness: brightness,
		Mode:       ColorModeColorTemp,
	}
}

// HexString returns the color as a hex string (e.g., "#FF0000")
func (c *Color) HexString() string {
	r, g, b := c.RGB()
	return "#" + hexByte(r) + hexByte(g) + hexByte(b)
}

func hexByte(b uint8) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0x0F]})
}

// BrightnessPct returns brightness as a percentage (0-100)
func (c *Color) BrightnessPct() int {
	return int(float64(c.Brightness) / 254.0 * 100)
}
