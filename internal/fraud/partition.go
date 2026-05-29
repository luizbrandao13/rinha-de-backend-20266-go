package fraud

// partitionTag matches the 2-bit split used by top submissions:
// (unknown_merchant << 1) | has_last_transaction.
func partitionTag(unknownMerchant bool, hasLastTx bool) int {
	tag := 0
	if unknownMerchant {
		tag |= 2
	}
	if hasLastTx {
		tag |= 1
	}
	return tag
}

func partitionFromRequest(r *Request) int {
	unknown := true
	for _, m := range r.Customer.KnownMerchants {
		if m == r.Merchant.ID {
			unknown = false
			break
		}
	}
	return partitionTag(unknown, r.LastTransaction != nil)
}

func partitionFromVector(points []float64, dim, row int) int {
	base := row * dim
	unknown := points[base+11] >= 0.5
	hasLast := points[base+5] != -1
	return partitionTag(unknown, hasLast)
}
