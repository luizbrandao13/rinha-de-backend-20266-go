package fraud

import "math"

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clamp01f32(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Round4 matches data-generator vector rounding.
func Round4(v float64) float64 {
	if v == -1 {
		return -1
	}
	return math.Round(v*10000) / 10000
}

func round4f32(v float32) float32 {
	return float32(Round4(float64(v)))
}
