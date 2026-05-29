package fraud

import "testing"

func TestDistSqMatchesKnownNeighbor(t *testing.T) {
	st, err := LoadStore("/tmp/refs.bin")
	if err != nil {
		t.Skip(err)
	}
	q := [14]float64{0.0442, 0.0833, 0.05, 0.6957, 0.6667, 1, 0.0184, 0.0339, 0.05, 0, 1, 0, 0.15, 0.0303}
	points := st.Points()
	dim := st.Dim()
	got := distSq14(&q, points, dim, 811572)
	want := 0.25374107
	if got < want-1e-6 || got > want+1e-6 {
		t.Fatalf("distSq14=%v want ~%v", got, want)
	}
}

func TestHeapKNNMatchesInsertion(t *testing.T) {
	st, err := LoadStore("/tmp/refs.bin")
	if err != nil {
		t.Skip(err)
	}
	q := [14]float64{0.0442, 0.0833, 0.05, 0.6957, 0.6667, 1, 0.0184, 0.0339, 0.05, 0, 1, 0, 0.15, 0.0303}
	points := st.Points()
	dim := st.Dim()
	n := st.N()

	var h neighborHeap
	for idx := 0; idx < n; idx++ {
		d2 := distSq14(&q, points, dim, idx)
		h.push(d2, int32(idx))
	}
	heapIdx := h.snapshot()

	// C-style insertion kNN
	dists := [5]float64{1e30, 1e30, 1e30, 1e30, 1e30}
	idxs := [5]int32{-1, -1, -1, -1, -1}
	for i := 0; i < n; i++ {
		d := distSq14(&q, points, dim, i)
		for j := 0; j < 5; j++ {
			if d < dists[j] {
				for k := 4; k > j; k-- {
					dists[k] = dists[k-1]
					idxs[k] = idxs[k-1]
				}
				dists[j] = d
				idxs[j] = int32(i)
				break
			}
		}
	}
	if !sameNeighborSet(heapIdx, idxs) {
		t.Fatalf("heap=%v insertion=%v", heapIdx, idxs)
	}
}
