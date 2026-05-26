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

## Ranking oficial (prévia) e parâmetros de pontuação

A prévia de resultados da edição 2026 está em **[https://rinhadebackend.com.br/](https://rinhadebackend.com.br/)** (tabela carregada no browser a partir dos dados publicados no repositório de preview). Serve como referência visual para **p99**, falhas e **score final** face a outras submissões.

Para calibrar a tua solução com a **mesma lógica de pontuação** que o site resume, usa o k6 deste repositório (gera `test/results.json` com o detalhe numérico):

```bash
# Com a API a ouvir em localhost:9999 (ex.: docker compose up)
K6_NO_USAGE_REPORT=true k6 run test/test.js
cat test/results.json | jq
```

Constantes usadas no cálculo (espelho de `test/test.js` / [docs/br/AVALIACAO.md](docs/br/AVALIACAO.md)):

| Parâmetro | Valor | Significado (resumo) |
|-----------|-------|------------------------|
| `T_MAX_MS` | 1000 | Referência de latência no termo log do score p99 |
| `P99_MIN_MS` | 1 | Piso de p99 em ms (satura o ganho de latência; entradas no topo do ranking ficam ~aqui) |
| `P99_MAX_MS` | 2000 | Acima disso, score de p99 vai a **−3000** (corte) |
| `K` | 1000 | Escala do termo log de p99 e de detecção |
| `EPSILON_MIN` | 0.001 | Piso para ε na componente de taxa de erro |
| `BETA` | 300 | Peso da penalidade absoluta em função de E |
| `TX_CORTE` | 0.15 (15%) | Se a taxa de falhas ultrapassar, score de detecção = **−3000** |
| Pesos em **E** | FP:1, FN:3, HTTP:5 | Erro ponderado para ε e penalidade |
| Score total | p99 + detecção | Cada parte varia tipicamente entre **−3000** e **+3000** |

Objetivo prático alinhado ao topo do ranking: **p99 o mais baixo possível** (idealmente na ordem de **1 ms** ou menos, sujeito ao hardware do teste oficial) mantendo **FP/FN/erros HTTP** baixos para não disparar o corte de 15% nem a penalidade de detecção.

## Nota sobre o dataset de testes

O ficheiro `test/test-data.json` inclui um `references_checksum_sha256` que pode não coincidir com o `resources/references.json.gz` deste clone. A lógica segue sempre o dataset de referência presente em `resources/`.
