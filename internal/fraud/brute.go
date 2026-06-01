package fraud

// BruteEvaluate runs exact kNN over all reference rows (for correctness checks).
func (e *Engine) BruteEvaluate(req *Request) (approved bool, fraudScore float64, fraudNeighbors int, err error) {
	var q [14]int16
	if err := VectorizeQueryI16(req, e.norm, e.mcc, &q); err != nil {
		return false, 0, 0, err
	}
	var qf [14]float64
	dequantQuery(&q, &qf, e.store.dim)
	var h neighborHeap
	n := e.store.N()
	for idx := 0; idx < n; idx++ {
		d2 := distSq14QF(&qf, e.store.pointsI16, e.store.dim, idx)
		h.push(d2, int32(idx))
	}
	neighbors := h.snapshot()
	fc := NeighborFraudCount(e.store.labels, neighbors)
	fs := FraudScoreFromNeighbors(fc)
	return Approved(fs), fs, fc, nil
}
