package fraud

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func TestStoreRowAlignment(t *testing.T) {
	path := os.Getenv("REFS_BIN")
	if path == "" {
		t.Skip("set REFS_BIN to a prepared refs.bin")
	}
	st, err := LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.N() <= 811572 {
		t.Skip("refs too small")
	}
	points := st.Points()
	dim := st.Dim()
	idx := 811572
	base := idx * dim
	row := points[base : base+dim]
	for _, v := range row {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < -1.01 || v > 1.01 {
			t.Fatalf("row %d looks misaligned: %v", idx, row)
		}
	}
}

func TestStoreRoundtripSmall(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/refs.bin"
	n, dim := 3, 14
	labels := []byte{0, 1, 0}
	points := make([]float64, n*dim)
	for i := range points {
		points[i] = float64(i) / 100
	}
	if err := writeStoreFixture(path, n, dim, points, labels); err != nil {
		t.Fatal(err)
	}
	st, err := LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.N() != n || st.Dim() != dim {
		t.Fatalf("n=%d dim=%d", st.N(), st.Dim())
	}
	got := st.Points()
	for i := range points {
		if math.Abs(got[i]-points[i]) > 1e-5 {
			t.Fatalf("points[%d]=%v want %v", i, got[i], points[i])
		}
	}
}

func writeStoreFixture(path string, n, dim int, points []float64, labels []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hdr := make([]byte, 16)
	copy(hdr[0:4], []byte(storeMagicV1))
	binary.LittleEndian.PutUint32(hdr[4:8], 1)
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(n))
	binary.LittleEndian.PutUint32(hdr[12:16], uint32(dim))
	if _, err := f.Write(hdr); err != nil {
		return err
	}
	buf := make([]byte, dim*4)
	for i := 0; i < n; i++ {
		for j := 0; j < dim; j++ {
			binary.LittleEndian.PutUint32(buf[j*4:(j+1)*4], math.Float32bits(float32(points[i*dim+j])))
		}
		if _, err := f.Write(buf); err != nil {
			return err
		}
	}
	_, err = f.Write(labels)
	return err
}
