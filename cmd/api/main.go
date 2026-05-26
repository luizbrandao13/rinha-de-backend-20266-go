package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/luizbrandao13/rinha-de-backend-20266-go/internal/fraud"
)

type fraudResp struct {
	Approved   bool    `json:"approved"`
	FraudScore float64 `json:"fraud_score"`
}

func main() {
	refPath := getenv("REFS_PATH", "/data/refs.bin")
	treePath := getenv("TREE_PATH", "/data/tree.bin")
	normPath := getenv("NORM_PATH", "/data/normalization.json")
	mccPath := getenv("MCC_PATH", "/data/mcc_risk.json")
	addr := getenv("LISTEN_ADDR", ":"+getenv("PORT", "8080"))

	log.Printf("loading references from %s (tree %s)", refPath, treePath)
	t0 := time.Now()
	eng, err := fraud.NewEngine(refPath, treePath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("index ready in %s", time.Since(t0))

	norm, err := fraud.LoadNorm(normPath)
	if err != nil {
		log.Fatal(err)
	}
	mcc, err := fraud.LoadMCCRisk(mccPath)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/fraud-score", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
		var req fraud.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		ok, score, err := eng.Evaluate(&req, norm, mcc)
		if err != nil {
			http.Error(w, "unprocessable", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(fraudResp{Approved: ok, FraudScore: score})
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
