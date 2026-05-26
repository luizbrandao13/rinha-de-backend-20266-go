package fraud

import "os"

// NewEngine loads references and optionally a pre-built VP-tree.
// If treePath is empty or the file is missing, the tree is built at startup (slow for 3M points).
func NewEngine(refPath, treePath string) (*Engine, error) {
	st, err := LoadStore(refPath)
	if err != nil {
		return nil, err
	}
	var root *vpNode
	if treePath != "" {
		if _, err := os.Stat(treePath); err == nil {
			root, err = LoadTree(treePath)
			if err != nil {
				return nil, err
			}
			return &Engine{store: st, tree: root}, nil
		}
	}
	root = BuildVPTree(st.points, st.n, st.dim, defaultLeafCap)
	return &Engine{store: st, tree: root}, nil
}

// Engine performs kNN (k=5) over the reference store.
type Engine struct {
	store *Store
	tree  *vpNode
}

// Evaluate runs vectorization + kNN and returns approval, fraud_score, and fraud neighbor count in {0..5}.
func (e *Engine) Evaluate(req *Request, norm Norm, mcc map[string]float64) (approved bool, fraudScore float64, fraudNeighbors int, err error) {
	var q [14]float64
	vec := q[:]
	if err := Vectorize(req, norm, mcc, vec); err != nil {
		return false, 0, 0, err
	}
	neighbors := e.tree.SearchK(vec, e.store.points, e.store.dim)
	fc := NeighborFraudCount(e.store.labels, neighbors)
	fs := FraudScoreFromNeighbors(fc)
	return Approved(fs), fs, fc, nil
}
