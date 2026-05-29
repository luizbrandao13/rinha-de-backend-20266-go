package fraud

import "os"

// Engine performs partitioned kNN (k=5) over the reference store.
type Engine struct {
	store *Store
	trees [4]*vpNode
	norm  NormFast
	mcc   *MCCTable
}

// NewEngine loads references and a VP-tree forest (VPT2) or legacy single tree.
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
	var trees [4]*vpNode
	if treePath != "" {
		if _, err := os.Stat(treePath); err == nil {
			trees, err = LoadTreeForest(treePath)
			if err != nil {
				return nil, err
			}
		}
	}
	if trees[0] == nil && trees[1] == nil && trees[2] == nil && trees[3] == nil {
		forest := BuildPartitionedForest(st.Points(), st.N(), st.Dim(), defaultLeafCap)
		trees = forest
	}
	return &Engine{
		store: st,
		trees: trees,
		norm:  NewNormFast(n),
		mcc:   mcc,
	}, nil
}

// Evaluate runs vectorization + exact kNN (k=5) over all reference partitions.
func (e *Engine) Evaluate(req *Request) (approved bool, fraudScore float64, fraudNeighbors int, err error) {
	var q [14]float64
	if err := VectorizeQuery(req, e.norm, e.mcc, &q); err != nil {
		return false, 0, 0, err
	}
	neighbors := SearchForestK(e.trees, &q, e.store.points, e.store.dim)
	fc := NeighborFraudCount(e.store.labels, neighbors)
	fs := FraudScoreFromNeighbors(fc)
	return Approved(fs), fs, fc, nil
}
