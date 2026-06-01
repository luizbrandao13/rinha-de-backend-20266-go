package fraud

import (
	"math"
	"math/rand"
	"sort"
)

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

const defaultLeafCap = 128

// BuildPartitionedForest builds four VP-trees keyed by partitionTag (unknown_merchant, has_last_tx).
func BuildPartitionedForest(points []float64, n, dim, leafCap int) [4]*vpNode {
	if leafCap <= 0 {
		leafCap = defaultLeafCap
	}
	var parts [4][]uint32
	for i := 0; i < n; i++ {
		p := partitionFromVector(points, dim, i)
		parts[p] = append(parts[p], uint32(i))
	}
	var forest [4]*vpNode
	for p := 0; p < 4; p++ {
		if len(parts[p]) == 0 {
			continue
		}
		forest[p] = buildIndexList(parts[p], points, dim, leafCap)
	}
	return forest
}

// BuildPartitionedForestI16 builds VP-trees over RNF3 int16 refs (same metric as query path).
func BuildPartitionedForestI16(points []int16, n, dim, leafCap int) [4]*vpNode {
	if leafCap <= 0 {
		leafCap = defaultLeafCap
	}
	var parts [4][]uint32
	for i := 0; i < n; i++ {
		p := partitionFromI16(points, dim, i)
		parts[p] = append(parts[p], uint32(i))
	}
	var forest [4]*vpNode
	for p := 0; p < 4; p++ {
		if len(parts[p]) == 0 {
			continue
		}
		forest[p] = buildIndexListI16(parts[p], points, dim, leafCap)
	}
	return forest
}

func buildIndexListI16(indices []uint32, points []int16, dim, leafCap int) *vpNode {
	cp := append([]uint32(nil), indices...)
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	tmp := make([]float64, len(cp))
	aux := make([]float64, len(cp))
	return buildRecRangeI16(cp, 0, len(cp), points, dim, leafCap, tmp, aux, rng)
}

func buildRecRangeI16(buf []uint32, lo, hi int, points []int16, dim, leafCap int, tmp, aux []float64, rng *rand.Rand) *vpNode {
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
		d := math.Sqrt(distSqRowsI16(points, dim, int(vantage), int(seg[i])))
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
		node.left = buildRecRangeI16(buf, lo+1, lo+mid, points, dim, leafCap, tmp, aux, rng)
		node.right = buildRecRangeI16(buf, lo+mid, hi, points, dim, leafCap, tmp, aux, rng)
		return node
	}

	node := &vpNode{vantage: vantage, mu: mu}
	node.left = buildRecRangeI16(buf, lo+1, lo+1+split, points, dim, leafCap, tmp, aux, rng)
	node.right = buildRecRangeI16(buf, lo+1+split, hi, points, dim, leafCap, tmp, aux, rng)
	return node
}

func buildIndexList(indices []uint32, points []float64, dim, leafCap int) *vpNode {
	cp := append([]uint32(nil), indices...)
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	tmp := make([]float64, len(cp))
	aux := make([]float64, len(cp))
	return buildRecRange(cp, 0, len(cp), points, dim, leafCap, tmp, aux, rng)
}

// BuildVPTree builds a single VP-tree over all rows (legacy / tests).
func BuildVPTree(points []float64, n, dim, leafCap int) *vpNode {
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

func buildRecRange(buf []uint32, lo, hi int, points []float64, dim, leafCap int, tmp, aux []float64, rng *rand.Rand) *vpNode {
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
		d := math.Sqrt(distSqRows(points, dim, int(vantage), int(seg[i])))
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

// SearchForestKF32 runs exact k=5 over float32 reference vectors (mmap hot path).
func SearchForestKF32(forest [4]*vpNode, q *[14]float64, points []float32, dim int) [5]int32 {
	var h neighborHeap
	seen := make(map[*vpNode]struct{}, 4)
	for _, root := range forest {
		if root == nil {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		root.searchRecF32(q, points, dim, &h)
	}
	return h.snapshot()
}

// SearchForestK runs exact k=5 over the union of all trees (merge top distances).
func SearchForestK(forest [4]*vpNode, q *[14]float64, points []float64, dim int) [5]int32 {
	var h neighborHeap
	seen := make(map[*vpNode]struct{}, 4)
	for _, root := range forest {
		if root == nil {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		root.searchRec(q, points, dim, &h)
	}
	return h.snapshot()
}

// SearchK returns indices of k=5 nearest neighbors within this tree only.
func (root *vpNode) SearchK(q *[14]float64, points []float64, dim int) [5]int32 {
	if root == nil {
		return [5]int32{-1, -1, -1, -1, -1}
	}
	var h neighborHeap
	root.searchRec(q, points, dim, &h)
	return h.snapshot()
}

func (n *vpNode) searchRec(q *[14]float64, points []float64, dim int, h *neighborHeap) {
	if n == nil {
		return
	}
	if n.leaf != nil {
		for _, idx := range n.leaf {
			d2 := distSq14(q, points, dim, int(idx))
			h.push(d2, int32(idx))
		}
		return
	}

	d0 := math.Sqrt(distSq14(q, points, dim, int(n.vantage)))
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

func (n *vpNode) searchRecF32(q *[14]float64, points []float32, dim int, h *neighborHeap) {
	if n == nil {
		return
	}
	if n.leaf != nil {
		for _, idx := range n.leaf {
			d2 := distSq14F32(q, points, dim, int(idx))
			h.push(d2, int32(idx))
		}
		return
	}

	d0 := math.Sqrt(distSq14F32(q, points, dim, int(n.vantage)))
	d0sq := d0 * d0
	h.push(d0sq, int32(n.vantage))

	tau := math.Sqrt(h.worst())
	if math.IsInf(tau, +1) {
		tau = math.MaxFloat64
	}

	if d0 < n.mu {
		n.left.searchRecF32(q, points, dim, h)
		tau = math.Sqrt(h.worst())
		if tau > n.mu-d0 {
			n.right.searchRecF32(q, points, dim, h)
		}
	} else {
		n.right.searchRecF32(q, points, dim, h)
		tau = math.Sqrt(h.worst())
		if tau > d0-n.mu {
			n.left.searchRecF32(q, points, dim, h)
		}
	}
}
