package fraud

import (
	"os"
	"testing"
)

func TestMmapForestMatchesBruteOnSample(t *testing.T) {
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
	n, err := LoadNorm("../../resources/normalization.json")
	if err != nil {
		t.Fatal(err)
	}
	mcc, err := LoadMCCTable("../../resources/mcc_risk.json")
	if err != nil {
		t.Fatal(err)
	}
	nf := NewNormFast(n)

	limit := 50
	if st.N() < limit {
		limit = st.N()
	}
	for i := 0; i < limit; i++ {
		// synthetic query from row i with small perturbation
		var q [14]int16
		base := i * st.dim
		for j := 0; j < st.dim; j++ {
			q[j] = st.pointsI16[base+j]
		}
		q[0]++
		got := forest.SearchK(&q, st.pointsI16, st.dim)
		want := bruteI16(&q, st)
		if !sameNeighborSet(got, want) {
			t.Fatalf("trial %d mmap=%v brute=%v", i, got, want)
		}
		_ = nf
		_ = mcc
	}
}
