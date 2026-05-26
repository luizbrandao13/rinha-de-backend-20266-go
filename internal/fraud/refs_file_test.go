package fraud

import (
	"os"
	"testing"
	"time"
)

func TestBuildFromRefsFile(t *testing.T) {
	path := os.Getenv("REFS_BIN_PATH")
	if path == "" {
		t.Skip("set REFS_BIN_PATH to refs.bin")
	}
	st, err := LoadStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	BuildVPTree(st.Points(), st.N(), st.Dim(), 64)
	t.Logf("built in %s", time.Since(t0))
}
