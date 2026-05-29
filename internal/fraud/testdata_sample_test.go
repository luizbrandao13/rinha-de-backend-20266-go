package fraud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectionAgainstTestDataSample(t *testing.T) {
	if os.Getenv("RUN_TESTDATA_SAMPLE") == "" {
		t.Skip("set RUN_TESTDATA_SAMPLE=1 to compare against test/test-data.json")
	}
	root := findRepoRoot(t)
	refPath := os.Getenv("REFS_BIN")
	treePath := os.Getenv("TREE_BIN")
	if refPath == "" {
		refPath = filepath.Join(root, "resources", "references.json.gz")
	}
	if treePath == "" {
		treePath = filepath.Join(os.TempDir(), "missing-tree.bin")
	}
	eng, err := NewEngine(refPath, treePath, filepath.Join(root, "resources", "normalization.json"), filepath.Join(root, "resources", "mcc_risk.json"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "test", "test-data.json"))
	if err != nil {
		t.Fatal(err)
	}
	var td struct {
		Entries []struct {
			Request          Request `json:"request"`
			ExpectedApproved bool    `json:"expected_approved"`
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
	bruteBad := 0
	for i := 0; i < limit; i++ {
		e := td.Entries[i]
		ok, _, _, err := eng.Evaluate(&e.Request)
		if err != nil {
			t.Fatalf("entry %d: %v", i, err)
		}
		bok, _, _, err := eng.BruteEvaluate(&e.Request)
		if err != nil {
			t.Fatalf("brute entry %d: %v", i, err)
		}
		if bok != e.ExpectedApproved {
			bruteBad++
		}
		if ok != e.ExpectedApproved {
			bad++
			if bad <= 3 {
				t.Logf("vp mismatch idx=%d expected=%v vp=%v brute=%v", i, e.ExpectedApproved, ok, bok)
			}
		}
	}
	if bruteBad > 0 {
		t.Fatalf("brute: %d/%d mismatches vs test-data expected_approved", bruteBad, limit)
	}
	if bad > 0 {
		t.Fatalf("vp-tree: %d/%d mismatches vs test-data", bad, limit)
	}
}
