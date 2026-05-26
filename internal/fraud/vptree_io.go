package fraud

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const treeMagic = "VPT1"

// WriteTree writes a VP-tree built from the same point set as Store (indices only).
func WriteTree(path string, root *vpNode) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, []byte(treeMagic)); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	var ver uint32 = 1
	if err := binary.Write(f, binary.LittleEndian, ver); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := writeNode(f, root); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func writeNode(w io.Writer, n *vpNode) error {
	if n == nil {
		return errors.New("nil node")
	}
	if n.leaf != nil {
		if err := binary.Write(w, binary.LittleEndian, byte(1)); err != nil {
			return err
		}
		c := uint32(len(n.leaf))
		if err := binary.Write(w, binary.LittleEndian, c); err != nil {
			return err
		}
		for _, idx := range n.leaf {
			if err := binary.Write(w, binary.LittleEndian, idx); err != nil {
				return err
			}
		}
		return nil
	}
	if err := binary.Write(w, binary.LittleEndian, byte(0)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, n.vantage); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, n.mu); err != nil {
		return err
	}
	if err := writeNode(w, n.left); err != nil {
		return err
	}
	return writeNode(w, n.right)
}

// LoadTree reads a tree file produced by WriteTree.
func LoadTree(path string) (*vpNode, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 8 || string(b[0:4]) != treeMagic {
		return nil, fmt.Errorf("bad tree magic")
	}
	r := bytesReader{b: b[4:]}
	var ver uint32
	if err := binary.Read(&r, binary.LittleEndian, &ver); err != nil {
		return nil, err
	}
	if ver != 1 {
		return nil, fmt.Errorf("bad tree version %d", ver)
	}
	node, err := readNode(&r)
	if err != nil {
		return nil, err
	}
	if len(r.b) != 0 {
		return nil, fmt.Errorf("trailing bytes %d", len(r.b))
	}
	return node, nil
}

type bytesReader struct {
	b []byte
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if len(r.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.b)
	r.b = r.b[n:]
	return n, nil
}

func readNode(r io.Reader) (*vpNode, error) {
	var kind byte
	if err := binary.Read(r, binary.LittleEndian, &kind); err != nil {
		return nil, err
	}
	if kind == 1 {
		var c uint32
		if err := binary.Read(r, binary.LittleEndian, &c); err != nil {
			return nil, err
		}
		leaf := make([]uint32, c)
		for i := range leaf {
			if err := binary.Read(r, binary.LittleEndian, &leaf[i]); err != nil {
				return nil, err
			}
		}
		return &vpNode{leaf: leaf}, nil
	}
	if kind != 0 {
		return nil, fmt.Errorf("bad node kind %d", kind)
	}
	var vantage uint32
	var mu float64
	if err := binary.Read(r, binary.LittleEndian, &vantage); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &mu); err != nil {
		return nil, err
	}
	left, err := readNode(r)
	if err != nil {
		return nil, err
	}
	right, err := readNode(r)
	if err != nil {
		return nil, err
	}
	return &vpNode{vantage: vantage, mu: mu, left: left, right: right}, nil
}
