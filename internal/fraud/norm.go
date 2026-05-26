package fraud

import (
	"encoding/json"
	"os"
)

// Norm holds constants from normalization.json.
type Norm struct {
	MaxAmount            float64 `json:"max_amount"`
	MaxInstallments      float64 `json:"max_installments"`
	AmountVsAvgRatio     float64 `json:"amount_vs_avg_ratio"`
	MaxMinutes           float64 `json:"max_minutes"`
	MaxKm                float64 `json:"max_km"`
	MaxTxCount24h        float64 `json:"max_tx_count_24h"`
	MaxMerchantAvgAmount float64 `json:"max_merchant_avg_amount"`
}

func LoadNorm(path string) (Norm, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Norm{}, err
	}
	var n Norm
	if err := json.Unmarshal(b, &n); err != nil {
		return Norm{}, err
	}
	return n, nil
}
