package fraud

// NormFast holds precomputed reciprocals for faster vectorization.
type NormFast struct {
	Norm
	InvMaxAmount            float64
	InvMaxInstallments      float64
	InvAmountVsAvgRatio     float64
	InvMaxMinutes           float64
	InvMaxKm                float64
	InvMaxTxCount24h        float64
	InvMaxMerchantAvgAmount float64
}

func NewNormFast(n Norm) NormFast {
	return NormFast{
		Norm:                    n,
		InvMaxAmount:            1 / n.MaxAmount,
		InvMaxInstallments:      1 / n.MaxInstallments,
		InvAmountVsAvgRatio:     1 / n.AmountVsAvgRatio,
		InvMaxMinutes:           1 / n.MaxMinutes,
		InvMaxKm:                1 / n.MaxKm,
		InvMaxTxCount24h:        1 / n.MaxTxCount24h,
		InvMaxMerchantAvgAmount: 1 / n.MaxMerchantAvgAmount,
	}
}
