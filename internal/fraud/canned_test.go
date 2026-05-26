package fraud

import (
	"encoding/json"
	"testing"
)

func TestCannedFraudScoreJSON(t *testing.T) {
	for frauds := 0; frauds <= 5; frauds++ {
		var got struct {
			Approved   bool    `json:"approved"`
			FraudScore float64 `json:"fraud_score"`
		}
		if err := json.Unmarshal(CannedFraudScoreJSON[frauds], &got); err != nil {
			t.Fatal(err)
		}
		fs := FraudScoreFromNeighbors(frauds)
		if got.FraudScore != fs {
			t.Fatalf("frauds=%d score want %v got %v", frauds, fs, got.FraudScore)
		}
		if got.Approved != Approved(fs) {
			t.Fatalf("frauds=%d approved want %v got %v", frauds, Approved(fs), got.Approved)
		}
	}
}
