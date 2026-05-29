package fraud

import (
	"testing"
)

func TestVectorizeGoldenLegit(t *testing.T) {
	n := Norm{
		MaxAmount:            10000,
		MaxInstallments:      12,
		AmountVsAvgRatio:     10,
		MaxMinutes:           1440,
		MaxKm:                1000,
		MaxTxCount24h:        20,
		MaxMerchantAvgAmount: 10000,
	}
	nf := NewNormFast(n)
	mcc, err := LoadMCCTable("../../resources/mcc_risk.json")
	if err != nil {
		t.Fatal(err)
	}

	req := &Request{}
	req.Transaction.Amount = 41.12
	req.Transaction.Installments = 2
	req.Transaction.RequestedAt = "2026-03-11T18:45:53Z"
	req.Customer.AvgAmount = 82.24
	req.Customer.TxCount24h = 3
	req.Customer.KnownMerchants = []string{"MERC-003", "MERC-016"}
	req.Merchant.ID = "MERC-016"
	req.Merchant.MCC = "5411"
	req.Merchant.AvgAmount = 60.25
	req.Terminal.IsOnline = false
	req.Terminal.CardPresent = true
	req.Terminal.KmFromHome = 29.23
	req.LastTransaction = nil

	want := [14]float64{0.0041, 0.1667, 0.05, 0.7826, 0.3333, -1, -1, 0.0292, 0.15, 0, 1, 0, 0.15, 0.006}
	var got [14]float64
	if err := VectorizeQuery(req, nf, mcc, &got); err != nil {
		t.Fatal(err)
	}
	for i := range want {
		if absDiff(got[i], want[i]) > 0.002 {
			t.Fatalf("dim %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func absDiff(a, b float64) float64 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}
