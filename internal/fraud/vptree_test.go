package fraud

import (
	"math"
	"math/rand"
	"testing"
)

func bruteK5(q []float64, points []float32, dim int) [5]int32 {
	type pair struct {
		d float64
		i int32
	}
	var best [5]pair
	for j := range best {
		best[j].d = math.Inf(1)
		best[j].i = -1
	}
	n := len(points) / dim
	for idx := 0; idx < n; idx++ {
		d := distanceSquared14to32(q, row(points, dim, uint32(idx)))
		// insert into best 5
		worst := 0
		for j := 1; j < 5; j++ {
			if best[j].d > best[worst].d {
				worst = j
			}
		}
		if d < best[worst].d {
			best[worst] = pair{d: d, i: int32(idx)}
		}
	}
	var out [5]int32
	for j := 0; j < 5; j++ {
		out[j] = best[j].i
	}
	return out
}

func sameNeighborSet(a, b [5]int32) bool {
	am := map[int32]int{}
	bm := map[int32]int{}
	for _, x := range a {
		if x < 0 {
			continue
		}
		am[x]++
	}
	for _, x := range b {
		if x < 0 {
			continue
		}
		bm[x]++
	}
	if len(am) != len(bm) {
		return false
	}
	for k, v := range am {
		if bm[k] != v {
			return false
		}
	}
	return true
}

func TestVPTreeMatchesBruteSmall(t *testing.T) {
	// small synthetic dataset
	dim := 14
	n := 200
	points := make([]float32, n*dim)
	rng := rand.New(rand.NewSource(1))
	for i := range points {
		points[i] = float32(rng.Float64())
	}
	tree := BuildVPTree(points, n, dim, 16)
	for trial := 0; trial < 50; trial++ {
		q := make([]float64, dim)
		for i := range q {
			q[i] = rng.Float64()
		}
		got := tree.SearchK(q, points, dim)
		want := bruteK5(q, points, dim)
		if !sameNeighborSet(got, want) {
			t.Fatalf("trial %d mismatch got=%v want=%v", trial, got, want)
		}
	}
}
