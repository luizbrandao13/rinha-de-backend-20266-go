package fraud

// distSq14 computes squared Euclidean distance between a 14D query and a dataset row.
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

func distSqRows(points []float64, dim, i, j int) float64 {
	bi, bj := i*dim, j*dim
	var s float64
	for k := 0; k < dim; k++ {
		d := points[bi+k] - points[bj+k]
		s += d * d
	}
	return s
}
