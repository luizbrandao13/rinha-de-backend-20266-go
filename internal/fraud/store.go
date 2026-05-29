package fraud

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const storeMagicV1 = "RNF1"
const storeMagicV2 = "RNF2"

// Store holds loaded reference vectors and labels (refs.bin).
type Store struct {
	data      []byte // mmap backing store
	n         int
	dim       int
	pointsF32 []float32
	pointsF64 []float64 // RNF2 only, or lazy materialization
	labels    []byte
}

func (s *Store) N() int   { return s.n }
func (s *Store) Dim() int { return s.dim }

// PointsF32 returns the row-major float32 matrix (RNF1 mmap view).
func (s *Store) PointsF32() []float32 { return s.pointsF32 }

// Points returns float64 rows for index build/tests (not used on hot API path).
func (s *Store) Points() []float64 {
	if s.pointsF64 != nil {
		return s.pointsF64
	}
	if len(s.pointsF32) == 0 {
		return nil
	}
	out := make([]float64, len(s.pointsF32))
	for i, v := range s.pointsF32 {
		out[i] = float64(v)
	}
	return out
}

// LoadStore memory-maps refs.bin (RNF1 float32 or RNF2 float64).
func LoadStore(path string) (*Store, error) {
	data, err := mmapRead(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 16 {
		return nil, errors.New("refs file too small")
	}
	magic := string(data[0:4])
	ver := binary.LittleEndian.Uint32(data[4:8])
	n := int(binary.LittleEndian.Uint32(data[8:12]))
	dim := int(binary.LittleEndian.Uint32(data[12:16]))
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
		if len(data) < offLab+n {
			return nil, fmt.Errorf("truncated file: need %d bytes", offLab+n)
		}
		pointsF64 := bytesToFloat64Slice(data[offVec:offLab])
		return &Store{
			data:      data,
			n:         n,
			dim:       dim,
			pointsF64: pointsF64,
			labels:    data[offLab : offLab+n],
		}, nil
	case storeMagicV1:
		if ver != 1 {
			return nil, fmt.Errorf("unsupported RNF1 version %d", ver)
		}
		vecBytes := n * dim * 4
		offVec := 16
		offLab := offVec + vecBytes
		if len(data) < offLab+n {
			return nil, fmt.Errorf("truncated file: need %d bytes", offLab+n)
		}
		pointsF32 := bytesToFloat32Slice(data[offVec:offLab])
		return &Store{
			data:      data,
			n:         n,
			dim:       dim,
			pointsF32: pointsF32,
			labels:    data[offLab : offLab+n],
		}, nil
	default:
		return nil, fmt.Errorf("bad magic %q", magic)
	}
}

func mmapRead(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size <= 0 {
		return nil, errors.New("empty refs file")
	}
	if size > 1<<31-1 {
		return nil, fmt.Errorf("refs file too large: %d", size)
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	return data, nil
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
