package fraud

import (
	"encoding/json"
	"os"
)

const defaultMCCRisk = 0.5

// MCCTable is a read-only MCC risk lookup (hot path friendly).
type MCCTable struct {
	byCode      map[string]float64
	defaultRisk float64
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
		byCode:      make(map[string]float64, len(raw)),
		defaultRisk: defaultMCCRisk,
	}
	for k, v := range raw {
		t.byCode[k] = v
	}
	return t, nil
}

func (t *MCCTable) Risk(mcc string) float64 {
	if t == nil {
		return defaultMCCRisk
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
