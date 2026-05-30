package main

import (
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/luizbrandao13/rinha-de-backend-20266-go/internal/fraud"
)

const magic = "RNF3"

type refRec struct {
	Vector []float64 `json:"vector"`
	Label  string    `json:"label"`
}

func main() {
	inPath := flag.String("in", "resources/references.json.gz", "input references.json.gz")
	outPath := flag.String("out", "refs.bin", "output binary")
	treePath := flag.String("tree", "tree.bin", "output VP-tree index")
	skipTree := flag.Bool("skip-tree", false, "only convert references, do not build VP-tree")
	flag.Parse()

	if err := convert(*inPath, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *skipTree {
		return
	}
	fmt.Fprintf(os.Stderr, "loading %s for VP-tree...\n", *outPath)
	st, err := fraud.LoadStore(*outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "building partitioned VP-trees n=%d...\n", st.N())
	t0 := time.Now()
	forest := fraud.BuildPartitionedForest(st.Points(), st.N(), st.Dim(), 128)
	if err := fraud.WriteTreeForest(*treePath, forest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "tree written to %s in %s\n", *treePath, time.Since(t0))
}

func convert(inPath, outPath string) error {
	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(inPath), ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()
		r = gr
	}

	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return fmt.Errorf("expected array start, got %v", tok)
	}

	tmpPath := outPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	hdr := make([]byte, 16)
	copy(hdr[0:4], []byte(magic))
	binary.LittleEndian.PutUint32(hdr[4:8], 3) // version
	binary.LittleEndian.PutUint32(hdr[8:12], 0)
	binary.LittleEndian.PutUint32(hdr[12:16], 14)
	if _, err := out.Write(hdr); err != nil {
		return err
	}

	bufF := make([]byte, 14*2)
	labels := make([]byte, 0, 1<<20)
	var n uint32
	for dec.More() {
		var rec refRec
		if err := dec.Decode(&rec); err != nil {
			return err
		}
		if len(rec.Vector) != 14 {
			return fmt.Errorf("vector len %d at row %d", len(rec.Vector), n)
		}
		for i, v := range rec.Vector {
			qv := fraud.QuantizeDim(v)
			binary.LittleEndian.PutUint16(bufF[i*2:(i+1)*2], uint16(qv))
		}
		if _, err := out.Write(bufF); err != nil {
			return err
		}
		var lb byte
		switch rec.Label {
		case "fraud":
			lb = 1
		case "legit":
			lb = 0
		default:
			return fmt.Errorf("unknown label %q at %d", rec.Label, n)
		}
		labels = append(labels, lb)
		n++
		if n%250_000 == 0 {
			fmt.Fprintf(os.Stderr, "converted %d rows\n", n)
		}
	}
	if _, err := out.Write(labels); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if tok, err = dec.Token(); err != nil {
		return err
	} else if d, ok := tok.(json.Delim); !ok || d != ']' {
		return fmt.Errorf("expected array end, got %v", tok)
	}

	if _, err := out.Seek(8, io.SeekStart); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	var nbuf [4]byte
	binary.LittleEndian.PutUint32(nbuf[:], n)
	if _, err := out.Write(nbuf[:]); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "done: %d rows -> %s\n", n, outPath)
	return nil
}
