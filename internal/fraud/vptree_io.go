package fraud

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const treeMagic = "VPT1"
const treeMagicV2 = "VPT2"
const numPartitions = 4

// WriteTree writes a single VP-tree (legacy VPT1).
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

// WriteTreeForest writes four partition VP-trees (VPT2).
func WriteTreeForest(path string, forest [4]*vpNode) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, []byte(treeMagicV2)); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	var ver uint32 = 2
	if err := binary.Write(f, binary.LittleEndian, ver); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	for i := 0; i < numPartitions; i++ {
		if err := writeNode(f, forest[i]); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func writeNode(w io.Writer, n *vpNode) error {
	if n == nil {
		// empty partition: leaf with zero entries
		if err := binary.Write(w, binary.LittleEndian, byte(1)); err != nil {
			return err
		}
		var c uint32
		return binary.Write(w, binary.LittleEndian, c)
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

// LoadTreeForest reads VPT2 (preferred) or VPT1 (single tree used for all partitions).
func LoadTreeForest(path string) ([4]*vpNode, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return [4]*vpNode{}, err
	}
	if len(b) < 8 {
		return [4]*vpNode{}, errors.New("tree file too small")
	}
	magic := string(b[0:4])
	r := bytesReader{b: b[4:]}
	var ver uint32
	if err := binary.Read(&r, binary.LittleEndian, &ver); err != nil {
		return [4]*vpNode{}, err
	}
	switch magic {
	case treeMagicV2:
		if ver != 2 {
			return [4]*vpNode{}, fmt.Errorf("bad tree version %d", ver)
		}
		var forest [4]*vpNode
		for i := 0; i < numPartitions; i++ {
			node, err := readNode(&r)
			if err != nil {
				return [4]*vpNode{}, err
			}
			forest[i] = node
		}
		if len(r.b) != 0 {
			return [4]*vpNode{}, fmt.Errorf("trailing bytes %d", len(r.b))
		}
		return forest, nil
	case treeMagic:
		if ver != 1 {
			return [4]*vpNode{}, fmt.Errorf("bad tree version %d", ver)
		}
		node, err := readNode(&r)
		if err != nil {
			return [4]*vpNode{}, err
		}
		return [4]*vpNode{node, node, node, node}, nil
	default:
		return [4]*vpNode{}, fmt.Errorf("bad tree magic %q", magic)
	}
}

// LoadTree reads a legacy single-tree file.
func LoadTree(path string) (*vpNode, error) {
	forest, err := LoadTreeForest(path)
	if err != nil {
		return nil, err
	}
	return forest[0], nil
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
		if c == 0 {
			return nil, nil
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
