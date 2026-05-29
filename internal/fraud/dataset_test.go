package fraud

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestReferencesChecksumMatchesTestData(t *testing.T) {
	root := findRepoRoot(t)
	gzPath := filepath.Join(root, "resources", "references.json.gz")
	tdPath := filepath.Join(root, "test", "test-data.json")

	want, err := expectedReferencesChecksum(tdPath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sha256GzipFile(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("references checksum mismatch:\n  test-data expects %s\n  gzip -d resources/references.json.gz gives %s\n  (compare SHA256 of the uncompressed JSON, not the .gz file)", want, got)
	}
}

func expectedReferencesChecksum(testDataPath string) (string, error) {
	b, err := os.ReadFile(testDataPath)
	if err != nil {
		return "", err
	}
	var meta struct {
		Checksum string `json:"references_checksum_sha256"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return "", err
	}
	if meta.Checksum == "" {
		return "", os.ErrInvalid
	}
	return meta.Checksum, nil
}

func sha256GzipFile(gzPath string) (string, error) {
	f, err := os.Open(gzPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()
	h := sha256.New()
	if _, err := io.Copy(h, gr); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "test", "test-data.json")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
