# Sales Radar — Test Scenarios

**Version:** April 2026 · **Scope:** Automated tests in repo + recommended manual / integration checks

---

## 1. Automated tests (Go)

Packages with `*_test.go` (run: `go test ./...` from repo root):

| Package | Focus |
|---------|--------|
| `internal/store` | Migrations, migration version parsing |
| `internal/discovery` | Discovery behavior / mocks |
| `internal/icp` | Scoring / catalog |
| `internal/review` | Review lead building |
| `internal/deduplication` | Dedup classification |
| `internal/extraction` | Parsing / extraction |
| `internal/enrichment` | Enrichment layer |
| `internal/status` | Status assignment |
| `internal/crm` | CRM mapping |
| `internal/companycheck` | Company checks |
| `internal/normalization` | Entity resolution |
| `cmd/salesradar` | CSV output tests |

**Note:** There is **no** dedicated `internal/api` HTTP handler test file yet; API behavior is covered indirectly via integration **manual** scenarios below.

---

## 2. Manual — API smoke (`cmd/api` or `cmd/web`)

**Prerequisites:** `go run ./cmd/api -addr :8080` (or `cmd/web`), DB path writable.

| # | Scenario | Steps | Expected |
|---|----------|--------|----------|
| A1 | Health | `GET /health` | `200`, JSON `status: ok` |
| A2 | Leads list | `GET /api/v1/leads` | `200`, `items` array, `meta.total_in_db` |
| A3 | Filter echo | `GET /api/v1/leads?sort=company&order=desc` | `filter_echo` reflects params |
| A4 | Lead by ID | `GET /api/v1/leads/1` | `200` + `lead`, or `404` JSON if missing |
| A5 | Settings | `GET /api/v1/settings` | `discovery_sources`, `icp`, `catalogs` |
| A6 | Pipeline run | `POST /api/v1/pipeline/run` | `200`, `rows_persisted`, `stats` populated |
| A7 | Export | `GET /api/v1/export.csv` | `200`, `text/csv`, attachment filename |
| A8 | Debug | `GET /api/v1/debug` | `200`, structured debug (may be empty if no run) |
| A9 | Error shape | `GET /api/v1/leads/999999999` | `404`, `{ "error": { "code", "message" } }` |

---

## 3. Manual — Legacy HTML (`cmd/web`)

| # | Scenario | Expected |
|---|----------|----------|
| H1 | `GET /` | Redirect to `/leads` |
| H2 | `GET /leads` | HTML list, 200 |
| H3 | `POST /run` with `Accept: application/json` | JSON stats |
| H4 | `GET /export.csv` | CSV download |
| H5 | `GET /settings` | Settings form |

---

## 4. Manual — Next.js (`frontend/`)

**Prerequisites:** `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8080` in `.env.local`; API running.

| # | Scenario | Expected |
|---|----------|----------|
| N1 | `npm run dev`, open `/` | Redirect to `/leads` |
| N2 | `/leads` | **200** (not Next 404); shows table or inline error if API down |
| N3 | Apply filters | URL updates; data reflects filters |
| N4 | Generate leads | Pipeline runs; URL gains pipeline query params; KPIs when echoed |
| N5 | Export CSV | File downloads; columns match filtered set |
| N6 | `/leads/{id}` | Detail loads or API error message |
| N7 | Monorepo root | Dev server run from **`frontend/`**; `next.config` pins `turbopack.root` to avoid `/leads` **404** |

---

## 5. End-to-end business scenarios

| ID | Story | Verify |
|----|--------|--------|
| E1 | **First run** | No rows → Generate → list populated; Debug shows run; KPIs present (URL or `summary.last_run`). |
| E2 | **Settings** | Toggle off Apollo → run still completes (skipped enrichment); badges consistent. |
| E3 | **ICP exclusion** | Configure ICP → low/no match leads increase **rejected** in stats. |
| E4 | **Export parity** | Set filters → export CSV → row count matches list count for same query. |
| E5 | **Idempotent replace** | Run twice → DB reflects **latest** run only (no duplicate generations in list). |

---

## 6. Non-functional checks

| Area | Check |
|------|--------|
| **SQLite locking** | Concurrent read during write: `busy_timeout` in DSN (`store.Open`). |
| **Long pipeline** | Reverse proxy timeout > worst-case run time in production. |
| **Secrets** | No `SALESRADAR_*` keys in frontend bundle (only `NEXT_PUBLIC_*` for URL). |

---

## 7. Regression risks (when changing code)

- **`internal/api/request/listfilter.go`** — any new filter key must be added to **HTML form**, **Next filters**, **export**, and **API** docs.
- **`store.ReplaceAll`** — schema or transaction changes need migration tests + E2E run.
- **`pipeline.RunStats` JSON** — Debug page and Next pipeline URL merge depend on field names (`snake_case`).

---

*Last updated April 2026.*
