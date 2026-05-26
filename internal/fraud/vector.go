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

// Vectorize builds the 14-dimensional vector per REGRAS_DE_DETECCAO.md.
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
		return defaultMCCRisk
	}
	v, ok := m[mcc]
	if !ok {
		return defaultMCCRisk
	}
	return v
}

func distanceSquared14to32(q []float64, p []float32) float64 {
	var s float64
	for i := 0; i < 14; i++ {
		d := q[i] - float64(p[i])
		s += d * d
	}
	return s
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
