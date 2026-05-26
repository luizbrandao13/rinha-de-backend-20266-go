package fraud

import (
	"math"
	"math/rand"
	"sort"
)

func row(points []float32, dim int, i uint32) []float32 {
	base := int(i) * dim
	return points[base : base+dim]
}

func distSqRow(points []float32, dim int, i, j uint32) float64 {
	bi := int(i) * dim
	bj := int(j) * dim
	var s float64
	for k := 0; k < dim; k++ {
		d := float64(points[bi+k]) - float64(points[bj+k])
		s += d * d
	}
	return s
}

func distRow(points []float32, dim int, i, j uint32) float64 {
	return math.Sqrt(distSqRow(points, dim, i, j))
}

// --- keep k=5 smallest squared distances ---

type neighborHeap struct {
	d [5]float64
	i [5]int32
	n int
}

func (h *neighborHeap) worst() float64 {
	if h.n == 0 {
		return math.Inf(+1)
	}
	w := 0
	for j := 1; j < h.n; j++ {
		if h.d[j] > h.d[w] {
			w = j
		}
	}
	return h.d[w]
}

func (h *neighborHeap) push(distSq float64, idx int32) {
	if h.n < 5 {
		h.d[h.n] = distSq
		h.i[h.n] = idx
		h.n++
		return
	}
	w := 0
	for j := 1; j < 5; j++ {
		if h.d[j] > h.d[w] {
			w = j
		}
	}
	if distSq < h.d[w] {
		h.d[w] = distSq
		h.i[w] = idx
	}
}

func (h *neighborHeap) snapshot() [5]int32 {
	var i [5]int32
	copy(i[:], h.i[:h.n])
	for j := h.n; j < 5; j++ {
		i[j] = -1
	}
	return i
}

// --- VP-tree (Euclidean; mu in distance space, heap in dist^2) ---

type vpNode struct {
	vantage uint32
	mu      float64 // median distance ||p - vantage||
	left    *vpNode
	right   *vpNode
	leaf    []uint32
}

const defaultLeafCap = 64

// BuildVPTree builds a VP-tree over row-major points (n rows, dim cols).
func BuildVPTree(points []float32, n, dim, leafCap int) *vpNode {
	if leafCap <= 0 {
		leafCap = defaultLeafCap
	}
	indices := make([]uint32, n)
	for i := range indices {
		indices[i] = uint32(i)
	}
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})
	tmp := make([]float64, n)
	aux := make([]float64, n)
	return buildRecRange(indices, 0, n, points, dim, leafCap, tmp, aux, rng)
}

func buildRecRange(buf []uint32, lo, hi int, points []float32, dim, leafCap int, tmp, aux []float64, rng *rand.Rand) *vpNode {
	m := hi - lo
	if m <= leafCap {
		cp := make([]uint32, m)
		copy(cp, buf[lo:hi])
		return &vpNode{leaf: cp}
	}
	if m > 1 {
		j := lo + rng.Intn(m)
		buf[lo], buf[j] = buf[j], buf[lo]
	}
	vantage := buf[lo]
	if m == 1 {
		return &vpNode{leaf: []uint32{vantage}}
	}

	restLen := m - 1
	seg := buf[lo+1 : hi]
	for i := 0; i < restLen; i++ {
		d := distRow(points, dim, vantage, seg[i])
		if math.IsNaN(d) || math.IsInf(d, 0) {
			d = 1e12
		}
		tmp[i] = d
	}
	copy(aux[:restLen], tmp[:restLen])
	sort.Float64s(aux[:restLen])
	mu := aux[restLen/2]

	split := partitionByMu(tmp[:restLen], seg, mu)
	if split == 0 || split == restLen {
		mid := 1 + restLen/2
		node := &vpNode{vantage: vantage, mu: mu}
		node.left = buildRecRange(buf, lo+1, lo+mid, points, dim, leafCap, tmp, aux, rng)
		node.right = buildRecRange(buf, lo+mid, hi, points, dim, leafCap, tmp, aux, rng)
		return node
	}

	node := &vpNode{vantage: vantage, mu: mu}
	node.left = buildRecRange(buf, lo+1, lo+1+split, points, dim, leafCap, tmp, aux, rng)
	node.right = buildRecRange(buf, lo+1+split, hi, points, dim, leafCap, tmp, aux, rng)
	return node
}

// partitionByMu partitions parallel tmp and ids so tmp[0:split] < mu and tmp[split:] >= mu.
func partitionByMu(tmp []float64, ids []uint32, mu float64) int {
	a, b := 0, len(tmp)-1
	for a <= b {
		for a <= b && tmp[a] < mu {
			a++
		}
		for a <= b && tmp[b] >= mu {
			b--
		}
		if a < b {
			tmp[a], tmp[b] = tmp[b], tmp[a]
			ids[a], ids[b] = ids[b], ids[a]
			a++
			b--
		}
	}
	return a
}

// SearchK returns indices of k=5 nearest neighbors (squared Euclidean distance).
func (root *vpNode) SearchK(q []float64, points []float32, dim int) [5]int32 {
	var h neighborHeap
	root.searchRec(q, points, dim, &h)
	return h.snapshot()
}

func (n *vpNode) searchRec(q []float64, points []float32, dim int, h *neighborHeap) {
	if n == nil {
		return
	}
	if n.leaf != nil {
		for _, idx := range n.leaf {
			d2 := distanceSquared14to32(q, row(points, dim, idx))
			h.push(d2, int32(idx))
		}
		return
	}

	d0 := math.Sqrt(distanceSquared14to32(q, row(points, dim, n.vantage)))
	d0sq := d0 * d0
	h.push(d0sq, int32(n.vantage))

	tau := math.Sqrt(h.worst())
	if math.IsInf(tau, +1) {
		tau = math.MaxFloat64
	}

	if d0 < n.mu {
		n.left.searchRec(q, points, dim, h)
		tau = math.Sqrt(h.worst())
		if tau > n.mu-d0 {
			n.right.searchRec(q, points, dim, h)
		}
	} else {
		n.right.searchRec(q, points, dim, h)
		tau = math.Sqrt(h.worst())
		if tau > d0-n.mu {
			n.left.searchRec(q, points, dim, h)
		}
	}
}
