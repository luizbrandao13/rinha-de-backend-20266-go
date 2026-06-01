package fraud

import (
	"os"
	"testing"
	"time"
)

func BenchmarkMmapSearchVPT3(b *testing.B) {
	path := os.Getenv("REFS_BIN")
	treePath := os.Getenv("TREE_BIN")
	if path == "" || treePath == "" {
		b.Skip("set REFS_BIN and TREE_BIN")
	}
	st, err := LoadStore(path)
	if err != nil {
		b.Fatal(err)
	}
	forest, err := LoadTreeForestMmap(treePath)
	if err != nil {
		b.Fatal(err)
	}
	if !forest.fastSeek {
		b.Skip("need VPT3 tree")
	}
	var qf [14]float64
	for i := range qf {
		qf[i] = float64(i) / 14
	}
	part := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qf[0] = float64(i%997) / 997
		forest.SearchKQFPartition(&qf, st.pointsI16, st.dim, part)
	}
}

func TestMmapSearchLatencySample(t *testing.T) {
	if os.Getenv("RUN_SEARCH_BENCH") == "" {
		t.Skip("set RUN_SEARCH_BENCH=1")
	}
	path := os.Getenv("REFS_BIN")
	treePath := os.Getenv("TREE_BIN")
	if path == "" || treePath == "" {
		t.Skip("set REFS_BIN and TREE_BIN")
	}
	st, err := LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	forest, err := LoadTreeForestMmap(treePath)
	if err != nil {
		t.Fatal(err)
	}
	var qf [14]float64
	for i := range qf {
		qf[i] = 0.42
	}
	const trials = 200
	part := 1
	var total time.Duration
	for i := 0; i < trials; i++ {
		qf[0] = float64(i) / trials
		t0 := time.Now()
		forest.SearchKQFPartition(&qf, st.pointsI16, st.dim, part)
		total += time.Since(t0)
	}
	avg := total / trials
	t.Logf("avg search: %v (fastSeek=%v)", avg, forest.fastSeek)
	if avg > 50*time.Millisecond {
		t.Fatalf("search too slow: %v", avg)
	}
}
