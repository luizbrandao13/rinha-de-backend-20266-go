package fraud

// distSq14 computes squared Euclidean distance (query float64, refs float64 row).
func distSq14(q *[14]float64, points []float64, dim, row int) float64 {
	base := row * dim
	p := points[base : base+dim]
	var s float64
	d0 := q[0] - p[0]
	d1 := q[1] - p[1]
	d2 := q[2] - p[2]
	d3 := q[3] - p[3]
	s += d0*d0 + d1*d1 + d2*d2 + d3*d3
	d4 := q[4] - p[4]
	d5 := q[5] - p[5]
	d6 := q[6] - p[6]
	d7 := q[7] - p[7]
	s += d4*d4 + d5*d5 + d6*d6 + d7*d7
	d8 := q[8] - p[8]
	d9 := q[9] - p[9]
	d10 := q[10] - p[10]
	d11 := q[11] - p[11]
	s += d8*d8 + d9*d9 + d10*d10 + d11*d11
	d12 := q[12] - p[12]
	d13 := q[13] - p[13]
	s += d12*d12 + d13*d13
	return s
}

// distSq14F32 computes squared distance (query float64, refs float32 row).
func distSq14F32(q *[14]float64, points []float32, dim, row int) float64 {
	base := row * dim
	p := points[base : base+dim]
	var s float64
	d0 := q[0] - float64(p[0])
	d1 := q[1] - float64(p[1])
	d2 := q[2] - float64(p[2])
	d3 := q[3] - float64(p[3])
	s += d0*d0 + d1*d1 + d2*d2 + d3*d3
	d4 := q[4] - float64(p[4])
	d5 := q[5] - float64(p[5])
	d6 := q[6] - float64(p[6])
	d7 := q[7] - float64(p[7])
	s += d4*d4 + d5*d5 + d6*d6 + d7*d7
	d8 := q[8] - float64(p[8])
	d9 := q[9] - float64(p[9])
	d10 := q[10] - float64(p[10])
	d11 := q[11] - float64(p[11])
	s += d8*d8 + d9*d9 + d10*d10 + d11*d11
	d12 := q[12] - float64(p[12])
	d13 := q[13] - float64(p[13])
	s += d12*d12 + d13*d13
	return s
}

// distSq14I16 computes squared distance (quantized int16 refs + query).
func distSq14I16(q *[14]int16, points []int16, dim, row int) float64 {
	var qf [14]float64
	for i := 0; i < dim; i++ {
		qf[i] = DequantDim(q[i])
	}
	return distSq14QF(&qf, points, dim, row)
}

// distSq14QF computes squared distance with a pre-dequantized query vector.
func distSq14QF(q *[14]float64, points []int16, dim, row int) float64 {
	base := row * dim
	p := points[base : base+dim]
	var s float64
	for i := 0; i < dim; i++ {
		pi := p[i]
		var pf float64
		if pi == QuantMissing {
			pf = -1
		} else {
			pf = float64(pi) / quantScale
		}
		d := q[i] - pf
		s += d * d
	}
	return s
}

// distSqRowsI16 computes squared distance between two int16 reference rows.
func distSqRowsI16(points []int16, dim, i, j int) float64 {
	bi, bj := i*dim, j*dim
	var s float64
	for k := 0; k < dim; k++ {
		pi := points[bi+k]
		pj := points[bj+k]
		var fi, fj float64
		if pi == QuantMissing {
			fi = -1
		} else {
			fi = float64(pi) / quantScale
		}
		if pj == QuantMissing {
			fj = -1
		} else {
			fj = float64(pj) / quantScale
		}
		d := fi - fj
		s += d * d
	}
	return s
}

// dequantQuery fills qf from a quantized query vector.
func dequantQuery(q *[14]int16, qf *[14]float64, dim int) {
	for i := 0; i < dim; i++ {
		qf[i] = DequantDim(q[i])
	}
}

func distSq14StoreI16(q *[14]int16, st *Store, row int) float64 {
	return distSq14I16(q, st.pointsI16, st.dim, row)
}

func distSq14Store(q *[14]float64, st *Store, row int) float64 {
	if len(st.pointsF32) > 0 {
		return distSq14F32(q, st.pointsF32, st.dim, row)
	}
	return distSq14(q, st.pointsF64, st.dim, row)
}

func distSqRows(points []float64, dim, i, j int) float64 {
	bi, bj := i*dim, j*dim
	var s float64
	for k := 0; k < dim; k++ {
		d := points[bi+k] - points[bj+k]
		s += d * d
	}
	return s
}
