package fraud

import (
	"testing"
)

func TestVPTreeRoundtripIO(t *testing.T) {
	dim := 14
	n := 500
	points := make([]float32, n*dim)
	for i := range points {
		points[i] = float32(i%17) / 17
	}
	root := BuildVPTree(points, n, dim, 32)
	path := t.TempDir() + "/tree.bin"
	if err := WriteTree(path, root); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTree(path)
	if err != nil {
		t.Fatal(err)
	}
	rng := make([]float64, dim)
	for trial := 0; trial < 20; trial++ {
		for i := range rng {
			rng[i] = float64(trial*i%13) / 13
		}
		a := root.SearchK(rng, points, dim)
		b := got.SearchK(rng, points, dim)
		if a != b {
			t.Fatalf("trial %d mismatch %v vs %v", trial, a, b)
		}
	}
}
