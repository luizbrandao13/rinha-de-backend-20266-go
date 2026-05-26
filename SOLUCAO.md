# Solução Go — Rinha de Backend 2026 (detecção de fraude)

Implementação da API exigida pelo desafio: `GET /ready`, `POST /fraud-score`, vetor 14D conforme [docs/br/REGRAS_DE_DETECCAO.md](docs/br/REGRAS_DE_DETECCAO.md), kNN com **k = 5** em distância euclidiana, `fraud_score = fraudes nos vizinhos / 5`, `approved = fraud_score < 0.6`.

## Componentes

- **`cmd/api`**: servidor HTTP (JSON, corpo limitado a 256 KiB).
- **`cmd/prepare`**: converte `references.json.gz` para `refs.bin` e gera `tree.bin` (VP-tree serializado).
- **`internal/fraud`**: normalização, VP-tree, carregamento dos dados.

## Desenvolvimento local

```bash
go test ./...
go vet ./...

# gerar índice (usa resources/references.json.gz)
go run ./cmd/prepare -in resources/references.json.gz -out /tmp/refs.bin -tree /tmp/tree.bin

REFS_PATH=/tmp/refs.bin TREE_PATH=/tmp/tree.bin \
NORM_PATH=resources/normalization.json MCC_PATH=resources/mcc_risk.json \
LISTEN_ADDR=:9999 go run ./cmd/api
```

## Docker (LB + 2 APIs)

Requer Docker. Na raiz do repositório:

```bash
docker compose up --build
```

O Nginx escuta na porta **9999** e faz round-robin para duas réplicas da API (porta interna 8080). Limites no `docker-compose.yml` respeitam **1 CPU / 350 MB** no total.

## Nota sobre o dataset de testes

O ficheiro `test/test-data.json` inclui um `references_checksum_sha256` que pode não coincidir com o `resources/references.json.gz` deste clone. A lógica segue sempre o dataset de referência presente em `resources/`.
