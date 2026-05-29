package fraud

// distSq14 computes squared Euclidean distance between a 14D query and a dataset row.
func distSq14(q *[14]float32, points []float32, dim, row int) float64 {
	base := row * dim
	p := points[base : base+dim]
	var s float64
	d0 := float64(q[0]) - float64(p[0])
	d1 := float64(q[1]) - float64(p[1])
	d2 := float64(q[2]) - float64(p[2])
	d3 := float64(q[3]) - float64(p[3])
	s += d0*d0 + d1*d1 + d2*d2 + d3*d3
	d4 := float64(q[4]) - float64(p[4])
	d5 := float64(q[5]) - float64(p[5])
	d6 := float64(q[6]) - float64(p[6])
	d7 := float64(q[7]) - float64(p[7])
	s += d4*d4 + d5*d5 + d6*d6 + d7*d7
	d8 := float64(q[8]) - float64(p[8])
	d9 := float64(q[9]) - float64(p[9])
	d10 := float64(q[10]) - float64(p[10])
	d11 := float64(q[11]) - float64(p[11])
	s += d8*d8 + d9*d9 + d10*d10 + d11*d11
	d12 := float64(q[12]) - float64(p[12])
	d13 := float64(q[13]) - float64(p[13])
	s += d12*d12 + d13*d13
	return s
}

func distSqRows(points []float32, dim, i, j int) float64 {
	bi, bj := i*dim, j*dim
	var s float64
	for k := 0; k < dim; k++ {
		d := float64(points[bi+k]) - float64(points[bj+k])
		s += d * d
	}
	return s
}
