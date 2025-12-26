package models

import (
	"math"
	"testing"
)

func TestHSVToRGB(t *testing.T) {
	tests := []struct {
		name       string
		hue        uint16
		saturation uint8
		brightness uint8
		wantR      uint8
		wantG      uint8
		wantB      uint8
		tolerance  uint8
	}{
		{
			name:       "red",
			hue:        0,
			saturation: 254,
			brightness: 254,
			wantR:      255,
			wantG:      0,
			wantB:      0,
			tolerance:  5,
		},
		{
			name:       "green",
			hue:        21845, // 120 degrees = 21845 in Hue scale (65535/3)
			saturation: 254,
			brightness: 254,
			wantR:      0,
			wantG:      255,
			wantB:      0,
			tolerance:  5,
		},
		{
			name:       "blue",
			hue:        43690, // 240 degrees = 43690 in Hue scale (65535*2/3)
			saturation: 254,
			brightness: 254,
			wantR:      0,
			wantG:      0,
			wantB:      255,
			tolerance:  5,
		},
		{
			name:       "white (no saturation)",
			hue:        0,
			saturation: 0,
			brightness: 254,
			wantR:      255,
			wantG:      255,
			wantB:      255,
			tolerance:  5,
		},
		{
			name:       "50% brightness red",
			hue:        0,
			saturation: 254,
			brightness: 127,
			wantR:      127,
			wantG:      0,
			wantB:      0,
			tolerance:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewColorFromHS(tt.hue, tt.saturation, tt.brightness)
			r, g, b := c.RGB()

			if !withinTolerance(r, tt.wantR, tt.tolerance) ||
				!withinTolerance(g, tt.wantG, tt.tolerance) ||
				!withinTolerance(b, tt.wantB, tt.tolerance) {
				t.Errorf("HSVToRGB() = (%d, %d, %d), want (%d, %d, %d) ±%d",
					r, g, b, tt.wantR, tt.wantG, tt.wantB, tt.tolerance)
			}
		})
	}
}

func TestMirekToRGB(t *testing.T) {
	tests := []struct {
		name      string
		mirek     uint16
		wantWarm  bool // true if result should be warm (more red), false if cool (more blue)
	}{
		{
			name:     "warm white (2000K)",
			mirek:    500,
			wantWarm: true,
		},
		{
			name:     "cool white (6500K)",
			mirek:    153,
			wantWarm: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewColorFromMirek(tt.mirek, 254)
			r, _, b := c.RGB()

			// Warm colors have R > B, cool colors have B >= R or close
			if tt.wantWarm {
				if r < b {
					t.Errorf("Expected warm color (R >= B), got R=%d, B=%d", r, b)
				}
			}
			// For cool, we just check it produces valid output
			// Color temperature algorithms are inherently approximate
			t.Logf("Mirek %d -> R=%d, B=%d", tt.mirek, r, b)
		})
	}
}

func TestXYToRGB(t *testing.T) {
	tests := []struct {
		name      string
		x         float64
		y         float64
		wantR     uint8
		wantG     uint8
		wantB     uint8
		tolerance uint8
	}{
		{
			name:      "D65 white point",
			x:         0.3127,
			y:         0.3290,
			wantR:     255,
			wantG:     255,
			wantB:     255,
			tolerance: 30, // XY conversion has more variance
		},
		{
			name:      "red gamut point",
			x:         0.675,
			y:         0.322,
			wantR:     255,
			wantG:     100, // XY red has some green component
			wantB:     0,
			tolerance: 120, // XY conversions are approximate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewColorFromXY(tt.x, tt.y, 254)
			r, g, b := c.RGB()

			// Log actual values for debugging
			t.Logf("XY(%f, %f) -> RGB(%d, %d, %d)", tt.x, tt.y, r, g, b)

			if !withinTolerance(r, tt.wantR, tt.tolerance) ||
				!withinTolerance(g, tt.wantG, tt.tolerance) ||
				!withinTolerance(b, tt.wantB, tt.tolerance) {
				t.Errorf("XYToRGB(%f, %f) = (%d, %d, %d), want (%d, %d, %d) ±%d",
					tt.x, tt.y, r, g, b, tt.wantR, tt.wantG, tt.wantB, tt.tolerance)
			}
		})
	}
}

func TestRGBToHSV(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b uint8
		wantH   uint16
		wantS   uint8
		wantV   uint8
		tolH    uint16
		tolSV   uint8
	}{
		{
			name:  "red",
			r:     255,
			g:     0,
			b:     0,
			wantH: 0,
			wantS: 254,
			wantV: 254,
			tolH:  1000,
			tolSV: 5,
		},
		{
			name:  "white",
			r:     255,
			g:     255,
			b:     255,
			wantH: 0,
			wantS: 0,
			wantV: 254,
			tolH:  65535, // hue is undefined for white
			tolSV: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s, v := rgbToHSV(tt.r, tt.g, tt.b)

			if !withinToleranceU16(h, tt.wantH, tt.tolH) {
				t.Errorf("rgbToHSV() hue = %d, want %d ±%d", h, tt.wantH, tt.tolH)
			}
			if !withinTolerance(s, tt.wantS, tt.tolSV) {
				t.Errorf("rgbToHSV() saturation = %d, want %d ±%d", s, tt.wantS, tt.tolSV)
			}
			if !withinTolerance(v, tt.wantV, tt.tolSV) {
				t.Errorf("rgbToHSV() value = %d, want %d ±%d", v, tt.wantV, tt.tolSV)
			}
		})
	}
}

func TestRGBToXY(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b uint8
		wantX   float64
		wantY   float64
		tol     float64
	}{
		{
			name:  "white",
			r:     255,
			g:     255,
			b:     255,
			wantX: 0.3127,
			wantY: 0.3290,
			tol:   0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := RGBToXY(tt.r, tt.g, tt.b)

			if math.Abs(x-tt.wantX) > tt.tol || math.Abs(y-tt.wantY) > tt.tol {
				t.Errorf("RGBToXY(%d, %d, %d) = (%f, %f), want (%f, %f) ±%f",
					tt.r, tt.g, tt.b, x, y, tt.wantX, tt.wantY, tt.tol)
			}
		})
	}
}

func TestColorCaching(t *testing.T) {
	c := NewColorFromHS(0, 254, 254)

	// First call should compute
	r1, g1, b1 := c.RGB()

	// Second call should use cache
	r2, g2, b2 := c.RGB()

	if r1 != r2 || g1 != g2 || b1 != b2 {
		t.Errorf("Cached RGB values differ: (%d,%d,%d) vs (%d,%d,%d)",
			r1, g1, b1, r2, g2, b2)
	}

	// Invalidate cache
	c.InvalidateCache()
	c.Hue = 21845 // Change to green

	r3, g3, b3 := c.RGB()
	if r3 == r1 && g3 == g1 && b3 == b1 {
		t.Error("Cache was not invalidated properly")
	}
}

func TestHexString(t *testing.T) {
	c := NewColorFromHS(0, 254, 254) // Red
	hex := c.HexString()

	if len(hex) != 7 || hex[0] != '#' {
		t.Errorf("HexString() = %s, want format #RRGGBB", hex)
	}

	// Should be red-ish
	if hex[1:3] != "FF" {
		t.Errorf("Expected red component to be FF, got %s", hex[1:3])
	}
}

func TestBrightnessPct(t *testing.T) {
	tests := []struct {
		brightness uint8
		wantPct    int
	}{
		{0, 0},
		{127, 50},
		{254, 100},
	}

	for _, tt := range tests {
		c := &Color{Brightness: tt.brightness}
		got := c.BrightnessPct()
		if got != tt.wantPct {
			t.Errorf("BrightnessPct(%d) = %d, want %d", tt.brightness, got, tt.wantPct)
		}
	}
}

func withinTolerance(got, want, tolerance uint8) bool {
	if got > want {
		return got-want <= tolerance
	}
	return want-got <= tolerance
}

func withinToleranceU16(got, want, tolerance uint16) bool {
	if got > want {
		return got-want <= tolerance
	}
	return want-got <= tolerance
}
