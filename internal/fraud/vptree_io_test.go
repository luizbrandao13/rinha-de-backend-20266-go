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
	var q [14]float32
	for trial := 0; trial < 20; trial++ {
		for i := range q {
			q[i] = float32(trial*i%13) / 13
		}
		a := root.SearchK(&q, points, dim)
		b := got.SearchK(&q, points, dim)
		if a != b {
			t.Fatalf("trial %d mismatch %v vs %v", trial, a, b)
		}
	}
}

func TestForestRoundtripIO(t *testing.T) {
	dim := 14
	n := 800
	points := make([]float32, n*dim)
	for i := range points {
		points[i] = float32(i%23) / 23
		if i%dim == 5 && i/dim%3 == 0 {
			points[i] = -1
		}
	}
	forest := BuildPartitionedForest(points, n, dim, 32)
	path := t.TempDir() + "/forest.bin"
	if err := WriteTreeForest(path, forest); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTreeForest(path)
	if err != nil {
		t.Fatal(err)
	}
	var q [14]float32
	for i := range q {
		q[i] = 0.42
	}
	part := partitionTag(q[11] >= 0.5, q[5] != -1)
	a := forest[part].SearchK(&q, points, dim)
	b := got[part].SearchK(&q, points, dim)
	if !sameNeighborSet(a, b) {
		t.Fatalf("mismatch %v vs %v", a, b)
	}
}
