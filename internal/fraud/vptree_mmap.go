package fraud

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// MmapForest is a VP-tree forest backed by a mmap'd tree.bin (no heap nodes).
type MmapForest struct {
	data     []byte
	roots    [4]int
	fastSeek bool // VPT3 stores left subtree size in internal nodes
}

// LoadTreeForestMmap memory-maps tree.bin for zero-copy traversal.
func LoadTreeForestMmap(path string) (*MmapForest, error) {
	data, err := mmapRead(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 8 {
		return nil, errors.New("tree file too small")
	}
	magic := string(data[0:4])
	ver := binary.LittleEndian.Uint32(data[4:8])
	switch magic {
	case treeMagicV3:
		if ver != 3 {
			return nil, fmt.Errorf("bad tree version %d", ver)
		}
		f := &MmapForest{data: data, fastSeek: true}
		off := 8
		for i := 0; i < numPartitions; i++ {
			if off >= len(data) {
				return nil, errors.New("truncated forest")
			}
			f.roots[i] = off
			sz, err := nodeSizeV3(data, off)
			if err != nil {
				return nil, err
			}
			off += sz
		}
		if off != len(data) {
			return nil, fmt.Errorf("trailing bytes %d", len(data)-off)
		}
		return f, nil
	case treeMagicV2:
		if ver != 2 {
			return nil, fmt.Errorf("bad tree version %d", ver)
		}
		f := &MmapForest{data: data}
		off := 8
		for i := 0; i < numPartitions; i++ {
			if off >= len(data) {
				return nil, errors.New("truncated forest")
			}
			f.roots[i] = off
			sz, err := nodeSize(data, off)
			if err != nil {
				return nil, err
			}
			off += sz
		}
		if off != len(data) {
			return nil, fmt.Errorf("trailing bytes %d", len(data)-off)
		}
		return f, nil
	case treeMagic:
		if ver != 1 {
			return nil, fmt.Errorf("bad tree version %d", ver)
		}
		f := &MmapForest{data: data}
		for i := range f.roots {
			f.roots[i] = 8
		}
		return f, nil
	default:
		return nil, fmt.Errorf("bad tree magic %q", magic)
	}
}

func nodeSizeV3(data []byte, off int) (int, error) {
	if off >= len(data) {
		return 0, errors.New("node out of range")
	}
	switch data[off] {
	case 1:
		if off+5 > len(data) {
			return 0, errors.New("leaf header truncated")
		}
		c := binary.LittleEndian.Uint32(data[off+1:])
		end := off + 5 + int(c)*4
		if end > len(data) {
			return 0, errors.New("leaf truncated")
		}
		return end - off, nil
	case 0:
		if off+internalHeaderV3 > len(data) {
			return 0, errors.New("internal header truncated")
		}
		ls := int(binary.LittleEndian.Uint32(data[off+13 : off+17]))
		left := off + internalHeaderV3
		right := left + ls
		if right > len(data) {
			return 0, errors.New("left subtree out of range")
		}
		rs, err := nodeSizeV3(data, right)
		if err != nil {
			return 0, err
		}
		return internalHeaderV3 + ls + rs, nil
	default:
		return 0, fmt.Errorf("bad node kind %d", data[off])
	}
}

func nodeSize(data []byte, off int) (int, error) {
	if off >= len(data) {
		return 0, errors.New("node out of range")
	}
	switch data[off] {
	case 1:
		if off+5 > len(data) {
			return 0, errors.New("leaf header truncated")
		}
		c := binary.LittleEndian.Uint32(data[off+1:])
		end := off + 5 + int(c)*4
		if end > len(data) {
			return 0, errors.New("leaf truncated")
		}
		return end - off, nil
	case 0:
		if off+internalHeaderV2 > len(data) {
			return 0, errors.New("internal header truncated")
		}
		left := off + internalHeaderV2
		ls, err := nodeSize(data, left)
		if err != nil {
			return 0, err
		}
		rs, err := nodeSize(data, left+ls)
		if err != nil {
			return 0, err
		}
		return internalHeaderV2 + ls + rs, nil
	default:
		return 0, fmt.Errorf("bad node kind %d", data[off])
	}
}

func childOffsets(data []byte, off int, fastSeek bool) (left, right int, ok bool) {
	if off >= len(data) || data[off] != 0 {
		return 0, 0, false
	}
	if fastSeek {
		if off+internalHeaderV3 > len(data) {
			return 0, 0, false
		}
		ls := int(binary.LittleEndian.Uint32(data[off+13 : off+17]))
		left = off + internalHeaderV3
		return left, left + ls, true
	}
	if off+internalHeaderV2 > len(data) {
		return 0, 0, false
	}
	left = off + internalHeaderV2
	ls, err := nodeSize(data, left)
	if err != nil {
		return 0, 0, false
	}
	return left, left + ls, true
}

// SearchK runs exact k=5 over all partitions.
func (f *MmapForest) SearchK(q *[14]int16, points []int16, dim int) [5]int32 {
	var qf [14]float64
	dequantQuery(q, &qf, dim)
	return f.SearchKQF(&qf, points, dim)
}

// SearchKQF runs exact k=5 with a pre-dequantized query vector.
func (f *MmapForest) SearchKQF(q *[14]float64, points []int16, dim int) [5]int32 {
	var h neighborHeap
	var seen [4]int
	nseen := 0
	for _, root := range f.roots {
		if root < 0 {
			continue
		}
		dup := false
		for j := 0; j < nseen; j++ {
			if seen[j] == root {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		seen[nseen] = root
		nseen++
		searchMmapRec(f, root, q, points, dim, &h)
	}
	return h.snapshot()
}

// SearchKQFPartition runs exact k=5 within one partition tree (~750k refs).
func (f *MmapForest) SearchKQFPartition(q *[14]float64, points []int16, dim, part int) [5]int32 {
	if part < 0 || part >= len(f.roots) || f.roots[part] < 0 {
		return [5]int32{-1, -1, -1, -1, -1}
	}
	var h neighborHeap
	searchMmapRec(f, f.roots[part], q, points, dim, &h)
	return h.snapshot()
}

func searchMmapRec(f *MmapForest, off int, q *[14]float64, points []int16, dim int, h *neighborHeap) {
	data := f.data
	if off >= len(data) {
		return
	}
	switch data[off] {
	case 1:
		c := binary.LittleEndian.Uint32(data[off+1:])
		base := off + 5
		for i := uint32(0); i < c; i++ {
			idx := binary.LittleEndian.Uint32(data[base+int(i)*4:])
			d2 := distSq14QF(q, points, dim, int(idx))
			h.push(d2, int32(idx))
		}
	case 0:
		vantage := binary.LittleEndian.Uint32(data[off+1:])
		mu := math.Float64frombits(binary.LittleEndian.Uint64(data[off+5 : off+13]))
		left, right, ok := childOffsets(data, off, f.fastSeek)
		if !ok {
			return
		}

		d0sq := distSq14QF(q, points, dim, int(vantage))
		d0 := math.Sqrt(d0sq)
		h.push(d0sq, int32(vantage))

		if d0 < mu {
			searchMmapRec(f, left, q, points, dim, h)
			tau := math.Sqrt(h.worst())
			if math.IsInf(tau, +1) {
				tau = math.MaxFloat64
			}
			if tau > mu-d0 {
				searchMmapRec(f, right, q, points, dim, h)
			}
		} else {
			searchMmapRec(f, right, q, points, dim, h)
			tau := math.Sqrt(h.worst())
			if math.IsInf(tau, +1) {
				tau = math.MaxFloat64
			}
			if tau > d0-mu {
				searchMmapRec(f, left, q, points, dim, h)
			}
		}
	}
}
