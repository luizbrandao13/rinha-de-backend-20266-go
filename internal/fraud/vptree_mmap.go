package fraud

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// MmapForest is a VP-tree forest backed by a mmap'd tree.bin (no heap nodes).
type MmapForest struct {
	data  []byte
	roots [4]int
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
		if off+13 > len(data) {
			return 0, errors.New("internal header truncated")
		}
		left := off + 13
		ls, err := nodeSize(data, left)
		if err != nil {
			return 0, err
		}
		rs, err := nodeSize(data, left+ls)
		if err != nil {
			return 0, err
		}
		return 13 + ls + rs, nil
	default:
		return 0, fmt.Errorf("bad node kind %d", data[off])
	}
}

// SearchK runs exact k=5 over all partitions.
func (f *MmapForest) SearchK(q *[14]int16, points []int16, dim int) [5]int32 {
	var h neighborHeap
	seen := make(map[int]struct{}, 4)
	for _, root := range f.roots {
		if root < 0 {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		searchMmapRec(f.data, root, q, points, dim, &h)
	}
	return h.snapshot()
}

func searchMmapRec(data []byte, off int, q *[14]int16, points []int16, dim int, h *neighborHeap) {
	if off >= len(data) {
		return
	}
	switch data[off] {
	case 1:
		c := binary.LittleEndian.Uint32(data[off+1:])
		base := off + 5
		for i := uint32(0); i < c; i++ {
			idx := binary.LittleEndian.Uint32(data[base+int(i)*4:])
			d2 := distSq14I16(q, points, dim, int(idx))
			h.push(d2, int32(idx))
		}
	case 0:
		vantage := binary.LittleEndian.Uint32(data[off+1:])
		mu := math.Float64frombits(binary.LittleEndian.Uint64(data[off+5 : off+13]))
		left := off + 13
		ls, err := nodeSize(data, left)
		if err != nil {
			return
		}
		right := left + ls

		d0 := math.Sqrt(distSq14I16(q, points, dim, int(vantage)))
		h.push(d0*d0, int32(vantage))

		if d0 < mu {
			searchMmapRec(data, left, q, points, dim, h)
			tau := math.Sqrt(h.worst())
			if math.IsInf(tau, +1) {
				tau = math.MaxFloat64
			}
			if tau > mu-d0 {
				searchMmapRec(data, right, q, points, dim, h)
			}
		} else {
			searchMmapRec(data, right, q, points, dim, h)
			tau := math.Sqrt(h.worst())
			if math.IsInf(tau, +1) {
				tau = math.MaxFloat64
			}
			if tau > d0-mu {
				searchMmapRec(data, left, q, points, dim, h)
			}
		}
	}
}
