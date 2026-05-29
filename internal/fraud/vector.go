package fraud

import (
	"time"
)

// Request mirrors the POST /fraud-score JSON (only fields needed for scoring).
type Request struct {
	Transaction struct {
		Amount       float64 `json:"amount"`
		Installments int     `json:"installments"`
		RequestedAt  string  `json:"requested_at"`
	} `json:"transaction"`
	Customer struct {
		AvgAmount      float64  `json:"avg_amount"`
		TxCount24h     int      `json:"tx_count_24h"`
		KnownMerchants []string `json:"known_merchants"`
	} `json:"customer"`
	Merchant struct {
		ID        string  `json:"id"`
		MCC       string  `json:"mcc"`
		AvgAmount float64 `json:"avg_amount"`
	} `json:"merchant"`
	Terminal struct {
		IsOnline    bool    `json:"is_online"`
		CardPresent bool    `json:"card_present"`
		KmFromHome  float64 `json:"km_from_home"`
	} `json:"terminal"`
	LastTransaction *struct {
		Timestamp     string  `json:"timestamp"`
		KmFromCurrent float64 `json:"km_from_current"`
	} `json:"last_transaction"`
}

// VectorizeF32 builds the 14-dimensional vector as float32 (search hot path).
func VectorizeF32(r *Request, n NormFast, mcc *MCCTable, out *[14]float32) error {
	t, err := time.Parse(time.RFC3339, r.Transaction.RequestedAt)
	if err != nil {
		return err
	}
	t = t.UTC()

	hour := float32(t.Hour()) / 23.0

	wd := int(t.Weekday())
	var dow int
	if wd == int(time.Sunday) {
		dow = 6
	} else {
		dow = wd - 1
	}
	day := float32(dow) / 6.0

	avg := r.Customer.AvgAmount
	if avg <= 0 {
		avg = 1e-9
	}
	amountVsAvg := (r.Transaction.Amount / avg) * n.InvAmountVsAvgRatio

	var minSince, kmLast float32
	if r.LastTransaction == nil {
		minSince = -1
		kmLast = -1
	} else {
		tLast, err := time.Parse(time.RFC3339, r.LastTransaction.Timestamp)
		if err != nil {
			return err
		}
		tLast = tLast.UTC()
		minutes := t.Sub(tLast).Minutes()
		if minutes < 0 {
			minutes = 0
		}
		minSince = clamp01f32(float32(minutes) * float32(n.InvMaxMinutes))
		kmLast = clamp01f32(float32(r.LastTransaction.KmFromCurrent) * float32(n.InvMaxKm))
	}

	unknown := float32(1)
	for _, m := range r.Customer.KnownMerchants {
		if m == r.Merchant.ID {
			unknown = 0
			break
		}
	}

	online := float32(0)
	if r.Terminal.IsOnline {
		online = 1
	}
	card := float32(0)
	if r.Terminal.CardPresent {
		card = 1
	}

	out[0] = clamp01f32(float32(r.Transaction.Amount) * float32(n.InvMaxAmount))
	out[1] = clamp01f32(float32(r.Transaction.Installments) * float32(n.InvMaxInstallments))
	out[2] = clamp01f32(float32(amountVsAvg))
	out[3] = hour
	out[4] = day
	out[5] = minSince
	out[6] = kmLast
	out[7] = clamp01f32(float32(r.Terminal.KmFromHome) * float32(n.InvMaxKm))
	out[8] = clamp01f32(float32(r.Customer.TxCount24h) * float32(n.InvMaxTxCount24h))
	out[9] = online
	out[10] = card
	out[11] = unknown
	out[12] = mcc.Risk(r.Merchant.MCC)
	out[13] = clamp01f32(float32(r.Merchant.AvgAmount) * float32(n.InvMaxMerchantAvgAmount))
	return nil
}

// Vectorize builds the 14-dimensional vector per REGRAS_DE_DETECCAO.md (float64).
func Vectorize(r *Request, n Norm, mcc map[string]float64, out []float64) error {
	if len(out) != 14 {
		panic("fraud: out must be length 14")
	}

	t, err := time.Parse(time.RFC3339, r.Transaction.RequestedAt)
	if err != nil {
		return err
	}
	t = t.UTC()

	// hour 0..23 -> /23
	hour := float64(t.Hour()) / 23.0

	// Monday=0 .. Sunday=6 -> /6
	wd := int(t.Weekday()) // Sunday=0 in Go
	var dow int
	if wd == int(time.Sunday) {
		dow = 6
	} else {
		dow = wd - 1
	}
	day := float64(dow) / 6.0

	avg := r.Customer.AvgAmount
	if avg <= 0 {
		avg = 1e-9
	}
	amountVsAvg := (r.Transaction.Amount / avg) / n.AmountVsAvgRatio

	var minSince, kmLast float64
	if r.LastTransaction == nil {
		minSince = -1
		kmLast = -1
	} else {
		tLast, err := time.Parse(time.RFC3339, r.LastTransaction.Timestamp)
		if err != nil {
			return err
		}
		tLast = tLast.UTC()
		minutes := t.Sub(tLast).Minutes()
		if minutes < 0 {
			minutes = 0
		}
		minSince = clamp01(minutes / n.MaxMinutes)
		kmLast = clamp01(r.LastTransaction.KmFromCurrent / n.MaxKm)
	}

	unknown := 1.0
	for _, m := range r.Customer.KnownMerchants {
		if m == r.Merchant.ID {
			unknown = 0
			break
		}
	}

	online := 0.0
	if r.Terminal.IsOnline {
		online = 1
	}
	card := 0.0
	if r.Terminal.CardPresent {
		card = 1
	}

	out[0] = clamp01(r.Transaction.Amount / n.MaxAmount)
	out[1] = clamp01(float64(r.Transaction.Installments) / n.MaxInstallments)
	out[2] = clamp01(amountVsAvg)
	out[3] = hour
	out[4] = day
	out[5] = minSince
	out[6] = kmLast
	out[7] = clamp01(r.Terminal.KmFromHome / n.MaxKm)
	out[8] = clamp01(float64(r.Customer.TxCount24h) / n.MaxTxCount24h)
	out[9] = online
	out[10] = card
	out[11] = unknown
	out[12] = mccRisk(mcc, r.Merchant.MCC)
	out[13] = clamp01(r.Merchant.AvgAmount / n.MaxMerchantAvgAmount)

	return nil
}

func mccRisk(m map[string]float64, mcc string) float64 {
	if m == nil {
		return 0.5
	}
	v, ok := m[mcc]
	if !ok {
		return 0.5
	}
	return v
}

// FraudScoreFromNeighbors returns fraud_score = fraudCount / 5.
func FraudScoreFromNeighbors(fraudCount int) float64 {
	return float64(fraudCount) / 5.0
}

// Approved applies the fixed threshold from the challenge.
func Approved(fraudScore float64) bool {
	return fraudScore < 0.6
}

// NeighborFraudCount counts fraud labels among k=5 neighbor indices.
func NeighborFraudCount(labels []byte, neighbors [5]int32) int {
	c := 0
	for _, idx := range neighbors {
		if idx < 0 {
			continue
		}
		if labels[idx] != 0 {
			c++
		}
	}
	return c
}

func clamp01f32(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
