package fraud

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"unsafe"
)

const storeMagicV1 = "RNF1"
const storeMagicV2 = "RNF2"

// Store holds loaded reference vectors and labels (refs.bin).
type Store struct {
	raw    []byte
	n      int
	dim    int
	points []float64 // len n*dim
	labels []byte    // len n
}

func (s *Store) N() int   { return s.n }
func (s *Store) Dim() int { return s.dim }

// Points returns the row-major float64 matrix (length n*dim).
func (s *Store) Points() []float64 { return s.points }

// LoadStore reads RNF2 (float64) or legacy RNF1 (float32 promoted to float64).
func LoadStore(path string) (*Store, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < 16 {
		return nil, errors.New("refs file too small")
	}
	magic := string(raw[0:4])
	ver := binary.LittleEndian.Uint32(raw[4:8])
	n := int(binary.LittleEndian.Uint32(raw[8:12]))
	dim := int(binary.LittleEndian.Uint32(raw[12:16]))
	if n <= 0 || dim != 14 {
		return nil, fmt.Errorf("invalid header n=%d dim=%d", n, dim)
	}
	switch magic {
	case storeMagicV2:
		if ver != 2 {
			return nil, fmt.Errorf("unsupported RNF2 version %d", ver)
		}
		vecBytes := n * dim * 8
		offVec := 16
		offLab := offVec + vecBytes
		if len(raw) < offLab+n {
			return nil, fmt.Errorf("truncated file: need %d bytes", offLab+n)
		}
		points := bytesToFloat64Slice(raw[offVec:offLab])
		labels := raw[offLab : offLab+n]
		return &Store{raw: raw, n: n, dim: dim, points: points, labels: labels}, nil
	case storeMagicV1:
		if ver != 1 {
			return nil, fmt.Errorf("unsupported RNF1 version %d", ver)
		}
		vecBytes := n * dim * 4
		offVec := 16
		offLab := offVec + vecBytes
		if len(raw) < offLab+n {
			return nil, fmt.Errorf("truncated file: need %d bytes", offLab+n)
		}
		f32 := bytesToFloat32Slice(raw[offVec:offLab])
		points := make([]float64, len(f32))
		for i, v := range f32 {
			points[i] = float64(v)
		}
		labels := raw[offLab : offLab+n]
		return &Store{raw: raw, n: n, dim: dim, points: points, labels: labels}, nil
	default:
		return nil, fmt.Errorf("bad magic %q", magic)
	}
}

func bytesToFloat32Slice(b []byte) []float32 {
	if len(b) == 0 {
		return nil
	}
	if len(b)%4 != 0 {
		panic("unaligned float32 buffer")
	}
	return unsafe.Slice((*float32)(unsafe.Pointer(&b[0])), len(b)/4)
}

func bytesToFloat64Slice(b []byte) []float64 {
	if len(b) == 0 {
		return nil
	}
	if len(b)%8 != 0 {
		panic("unaligned float64 buffer")
	}
	return unsafe.Slice((*float64)(unsafe.Pointer(&b[0])), len(b)/8)
}
