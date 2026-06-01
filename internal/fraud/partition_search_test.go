package fraud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPartitionSearchMatchesGlobalBrute(t *testing.T) {
	if os.Getenv("RUN_TESTDATA_SAMPLE") == "" {
		t.Skip("set RUN_TESTDATA_SAMPLE=1")
	}
	root := findRepoRoot(t)
	refPath := os.Getenv("REFS_BIN")
	treePath := os.Getenv("TREE_BIN")
	if refPath == "" || treePath == "" {
		t.Skip("set REFS_BIN and TREE_BIN")
	}
	st, err := LoadStore(refPath)
	if err != nil {
		t.Fatal(err)
	}
	forest, err := LoadTreeForestMmap(treePath)
	if err != nil {
		t.Fatal(err)
	}
	n, err := LoadNorm(filepath.Join(root, "resources", "normalization.json"))
	if err != nil {
		t.Fatal(err)
	}
	mcc, err := LoadMCCTable(filepath.Join(root, "resources", "mcc_risk.json"))
	if err != nil {
		t.Fatal(err)
	}
	nf := NewNormFast(n)

	b, err := os.ReadFile(filepath.Join(root, "test", "test-data.json"))
	if err != nil {
		t.Fatal(err)
	}
	var td struct {
		Entries []struct {
			Request Request `json:"request"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(b, &td); err != nil {
		t.Fatal(err)
	}
	limit := 500
	if len(td.Entries) < limit {
		limit = len(td.Entries)
	}
	bad := 0
	for i := 0; i < limit; i++ {
		req := &td.Entries[i].Request
		var q [14]int16
		if err := VectorizeQueryI16(req, nf, mcc, &q); err != nil {
			t.Fatal(err)
		}
		var qf [14]float64
		dequantQuery(&q, &qf, st.dim)
		part := partitionFromRequest(req)
		got := forest.SearchKQFPartition(&qf, st.pointsI16, st.dim, part)
		want := bruteI16(&q, st)
		if !sameNeighborSet(got, want) {
			bad++
			if bad <= 3 {
				t.Logf("idx=%d part=%d got=%v want=%v", i, part, got, want)
			}
		}
	}
	if bad > 0 {
		t.Fatalf("partition search: %d/%d neighbor mismatches vs global brute", bad, limit)
	}
}
