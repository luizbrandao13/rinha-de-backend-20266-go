package fraud

// CannedFraudScoreJSON returns pre-serialized POST /fraud-score bodies for each
// possible fraud neighbor count in {0..5} (exact kNN with k=5).
var CannedFraudScoreJSON [6][]byte

func init() {
	// fraud_score = i/5; approved = fraud_score < 0.6
	CannedFraudScoreJSON[0] = []byte(`{"approved":true,"fraud_score":0}`)
	CannedFraudScoreJSON[1] = []byte(`{"approved":true,"fraud_score":0.2}`)
	CannedFraudScoreJSON[2] = []byte(`{"approved":true,"fraud_score":0.4}`)
	CannedFraudScoreJSON[3] = []byte(`{"approved":false,"fraud_score":0.6}`)
	CannedFraudScoreJSON[4] = []byte(`{"approved":false,"fraud_score":0.8}`)
	CannedFraudScoreJSON[5] = []byte(`{"approved":false,"fraud_score":1}`)
}
