FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod ./
COPY . .

RUN mkdir -p /data \
    && cp resources/references.json.gz resources/normalization.json resources/mcc_risk.json /data/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/prepare ./cmd/prepare \
    && /out/prepare -in /data/references.json.gz -out /data/refs.bin -tree /data/tree.bin

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /data
COPY --from=build /data/normalization.json /data/mcc_risk.json /data/refs.bin /data/tree.bin ./
COPY --from=build /out/api /api

ENV REFS_PATH=/data/refs.bin \
    TREE_PATH=/data/tree.bin \
    NORM_PATH=/data/normalization.json \
    MCC_PATH=/data/mcc_risk.json \
    PORT=8080 \
    LISTEN_ADDR=:8080

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
