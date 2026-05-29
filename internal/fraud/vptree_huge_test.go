package fraud

import (
	"os"
	"testing"
	"time"
)

func TestVPHugeBuildTime(t *testing.T) {
	if os.Getenv("RUN_VP_HUGE") == "" {
		t.Skip("set RUN_VP_HUGE=1 to run 3M VP-tree build timing")
	}
	n := 3_000_000
	dim := 14
	points := make([]float64, n*dim)
	for i := 0; i < n; i++ {
		for j := 0; j < dim; j++ {
			points[i*dim+j] = float64((i+j)%997) / 997
		}
	}
	t0 := time.Now()
	BuildVPTree(points, n, dim, 64)
	t.Logf("build %d points in %s", n, time.Since(t0))
}
