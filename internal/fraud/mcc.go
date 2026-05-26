package fraud

import (
	"encoding/json"
	"os"
)

const defaultMCCRisk = 0.5

func LoadMCCRisk(path string) (map[string]float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]float64
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
