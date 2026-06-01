package fraud

import (
	"errors"
	"os"
)

// Engine performs partitioned kNN (k=5) over mmap'd references and VP index.
type Engine struct {
	store  *Store
	forest *MmapForest
	norm   NormFast
	mcc    *MCCTable
}

// NewEngine loads mmap'd references and VP-tree forest.
func NewEngine(refPath, treePath, normPath, mccPath string) (*Engine, error) {
	st, err := LoadStore(refPath)
	if err != nil {
		return nil, err
	}
	n, err := LoadNorm(normPath)
	if err != nil {
		return nil, err
	}
	mcc, err := LoadMCCTable(mccPath)
	if err != nil {
		return nil, err
	}
	var forest *MmapForest
	if treePath != "" {
		if _, err := os.Stat(treePath); err == nil {
			forest, err = LoadTreeForestMmap(treePath)
			if err != nil {
				return nil, err
			}
		}
	}
	if forest == nil {
		return nil, errors.New("tree.bin required")
	}
	return &Engine{
		store:  st,
		forest: forest,
		norm:   NewNormFast(n),
		mcc:    mcc,
	}, nil
}

// Evaluate runs vectorization + exact kNN (k=5).
func (e *Engine) Evaluate(req *Request) (approved bool, fraudScore float64, fraudNeighbors int, err error) {
	var q [14]int16
	if err := VectorizeQueryI16(req, e.norm, e.mcc, &q); err != nil {
		return false, 0, 0, err
	}
	var qf [14]float64
	dequantQuery(&q, &qf, e.store.dim)
	part := partitionFromRequest(req)
	neighbors := e.forest.SearchKQFPartition(&qf, e.store.pointsI16, e.store.dim, part)
	fc := NeighborFraudCount(e.store.labels, neighbors)
	fs := FraudScoreFromNeighbors(fc)
	return Approved(fs), fs, fc, nil
}
