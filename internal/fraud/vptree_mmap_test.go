package fraud

import (
	"os"
	"testing"
)

func TestMmapForestMatchesBruteI16(t *testing.T) {
	path := os.Getenv("REFS_BIN")
	treePath := os.Getenv("TREE_BIN")
	if path == "" || treePath == "" {
		t.Skip("set REFS_BIN and TREE_BIN")
	}
	st, err := LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.pointsI16) == 0 {
		t.Skip("need RNF3 refs")
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

	req := &Request{}
	req.Transaction.Amount = 441.59
	req.Transaction.Installments = 1
	req.Transaction.RequestedAt = "2027-07-09T16:31:06Z"
	req.Customer.AvgAmount = 883.18
	req.Customer.TxCount24h = 1
	req.Customer.KnownMerchants = []string{"MERC-004", "MERC-017"}
	req.Merchant.ID = "MERC-004"
	req.Merchant.MCC = "5411"
	req.Merchant.AvgAmount = 302.78
	req.Terminal.IsOnline = false
	req.Terminal.CardPresent = true
	req.Terminal.KmFromHome = 33.8814492067
	req.LastTransaction = &struct {
		Timestamp     string  `json:"timestamp"`
		KmFromCurrent float64 `json:"km_from_current"`
	}{Timestamp: "2027-06-04T14:14:22Z", KmFromCurrent: 18.4353521556}

	var q [14]int16
	if err := VectorizeQueryI16(req, nf, mcc, &q); err != nil {
		t.Fatal(err)
	}
	got := forest.SearchK(&q, st.pointsI16, st.dim)
	want := bruteI16(&q, st)
	if !sameNeighborSet(got, want) {
		t.Fatalf("mmap=%v brute=%v", got, want)
	}
}

func bruteI16(q *[14]int16, st *Store) [5]int32 {
	var h neighborHeap
	for idx := 0; idx < st.N(); idx++ {
		d2 := distSq14I16(q, st.pointsI16, st.dim, idx)
		h.push(d2, int32(idx))
	}
	return h.snapshot()
}
