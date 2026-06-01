package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/luizbrandao13/rinha-de-backend-20266-go/internal/fraud"
	"github.com/valyala/fasthttp"
)

func main() {
	if v := os.Getenv("GOMAXPROCS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			runtime.GOMAXPROCS(n)
		}
	} else {
		runtime.GOMAXPROCS(1)
	}

	refPath := getenv("REFS_PATH", "/data/refs.bin")
	treePath := getenv("TREE_PATH", "/data/tree.bin")
	normPath := getenv("NORM_PATH", "/data/normalization.json")
	mccPath := getenv("MCC_PATH", "/data/mcc_risk.json")
	addr := getenv("LISTEN_ADDR", ":"+getenv("PORT", "8080"))

	log.Printf("loading references from %s (tree %s)", refPath, treePath)
	t0 := time.Now()
	eng, err := fraud.NewEngine(refPath, treePath, normPath, mccPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("index ready in %s", time.Since(t0))

	srv := &fasthttp.Server{
		Handler:                       makeHandler(eng),
		ReadBufferSize:                4096,
		WriteBufferSize:               256,
		MaxRequestBodySize:            256 << 10,
		DisableHeaderNamesNormalizing: true,
		NoDefaultServerHeader:         true,
		ReduceMemoryUsage:             true,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe(addr))
}

func makeHandler(eng *fraud.Engine) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/ready":
			if !ctx.IsGet() {
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
				return
			}
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBodyString("ok")
		case "/fraud-score":
			if !ctx.IsPost() {
				ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
				return
			}
			var req fraud.Request
			if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				ctx.SetBodyString("bad json")
				return
			}
			_, _, frauds, err := eng.Evaluate(&req)
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusUnprocessableEntity)
				ctx.SetBodyString("unprocessable")
				return
			}
			if frauds < 0 || frauds > 5 {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				return
			}
			ctx.SetContentType("application/json")
			ctx.SetBody(fraud.CannedFraudScoreJSON[frauds])
		default:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
		}
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
