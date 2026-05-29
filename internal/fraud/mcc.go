package fraud

import (
	"encoding/json"
	"os"
)

const defaultMCCRiskF32 = float32(0.5)

// MCCTable is a read-only MCC risk lookup (hot path friendly).
type MCCTable struct {
	byCode      map[string]float32
	defaultRisk float32
}

func LoadMCCTable(path string) (*MCCTable, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	t := &MCCTable{
		byCode:      make(map[string]float32, len(raw)),
		defaultRisk: defaultMCCRiskF32,
	}
	for k, v := range raw {
		t.byCode[k] = float32(v)
	}
	return t, nil
}

func (t *MCCTable) Risk(mcc string) float32 {
	if t == nil {
		return defaultMCCRiskF32
	}
	if v, ok := t.byCode[mcc]; ok {
		return v
	}
	return t.defaultRisk
}

// LoadMCCRisk remains for compatibility; prefer MCCTable in the hot path.
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
