package api

import (
	"math"
	"testing"
)

func TestHSToXY(t *testing.T) {
	tests := []struct {
		name    string
		hue     uint16
		sat     uint8
		wantX   float64
		wantY   float64
		epsilon float64
	}{
		{
			name:    "red (hue=0)",
			hue:     0,
			sat:     254,
			wantX:   0.70,
			wantY:   0.30,
			epsilon: 0.02,
		},
		{
			name:    "green (hue=21845, ~120°)",
			hue:     21845,
			sat:     254,
			wantX:   0.17,
			wantY:   0.75,
			epsilon: 0.02,
		},
		{
			name:    "blue (hue=43690, ~240°)",
			hue:     43690,
			sat:     254,
			wantX:   0.15,
			wantY:   0.04,
			epsilon: 0.02,
		},
		{
			name:    "white (saturation=0)",
			hue:     0,
			sat:     0,
			wantX:   0.323,
			wantY:   0.329,
			epsilon: 0.02,
		},
		{
			name:    "purple (hue=54613, ~300°)",
			hue:     54613,
			sat:     254,
			wantX:   0.385,
			wantY:   0.155,
			epsilon: 0.02,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := HSToXY(tt.hue, tt.sat)

			if math.Abs(gotX-tt.wantX) > tt.epsilon {
				t.Errorf("HSToXY() X = %v, want %v (±%v)", gotX, tt.wantX, tt.epsilon)
			}
			if math.Abs(gotY-tt.wantY) > tt.epsilon {
				t.Errorf("HSToXY() Y = %v, want %v (±%v)", gotY, tt.wantY, tt.epsilon)
			}
		})
	}
}

func TestHSToXY_Consistency(t *testing.T) {
	// Test that HSToXY produces consistent results
	hue := uint16(32768)
	sat := uint8(200)

	x1, y1 := HSToXY(hue, sat)
	x2, y2 := HSToXY(hue, sat)

	if x1 != x2 || y1 != y2 {
		t.Errorf("HSToXY not consistent: (%v,%v) vs (%v,%v)", x1, y1, x2, y2)
	}
}

func TestHSToXY_ValidRange(t *testing.T) {
	// XY values should be in range [0, 1]
	testCases := []struct {
		hue uint16
		sat uint8
	}{
		{0, 0},
		{0, 254},
		{32768, 127},
		{65535, 254},
		{16384, 200},
	}

	for _, tc := range testCases {
		x, y := HSToXY(tc.hue, tc.sat)

		if x < 0 || x > 1 {
			t.Errorf("HSToXY(%d, %d) X=%v out of range [0,1]", tc.hue, tc.sat, x)
		}
		if y < 0 || y > 1 {
			t.Errorf("HSToXY(%d, %d) Y=%v out of range [0,1]", tc.hue, tc.sat, y)
		}
	}
}

func TestHSToXY_HueWrap(t *testing.T) {
	// Hue 0 and hue 65535 should produce similar results (both are ~red)
	x0, y0 := HSToXY(0, 254)
	xMax, yMax := HSToXY(65535, 254)

	// They should be very close (within 1% of hue range difference)
	epsilon := 0.02
	if math.Abs(x0-xMax) > epsilon || math.Abs(y0-yMax) > epsilon {
		t.Errorf("Hue wrap: (0)=(%v,%v) vs (65535)=(%v,%v) differ too much",
			x0, y0, xMax, yMax)
	}
}
