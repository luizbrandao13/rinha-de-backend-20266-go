package fraud

import "math"

const quantScale = 32767

// QuantMissing marks absent last-transaction dimensions (maps to float -1).
const QuantMissing int16 = -32768

// QuantizeDim converts one normalized dimension to int16.
func QuantizeDim(v float64) int16 {
	if v == -1 {
		return QuantMissing
	}
	v = Round4(v)
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return int16(math.Round(v * quantScale))
}

// DequantDim converts int16 back to float64 for distance math.
func DequantDim(v int16) float64 {
	if v == QuantMissing {
		return -1
	}
	return float64(v) / quantScale
}

// VectorizeQueryI16 builds the query vector as quantized int16.
func VectorizeQueryI16(r *Request, n NormFast, mcc *MCCTable, out *[14]int16) error {
	var f [14]float64
	if err := VectorizeQuery(r, n, mcc, &f); err != nil {
		return err
	}
	for i := range f {
		out[i] = QuantizeDim(f[i])
	}
	return nil
}
