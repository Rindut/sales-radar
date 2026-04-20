# Sales Radar — API & Data Model

**Version:** April 2026 · **Implementation:** `internal/api`, `internal/store`, `internal/api/dto`

---

## 1. API surface overview

All JSON routes are registered by **`api.Register(mux, db)`** (`internal/api/server.go`). Method-aware patterns require **Go 1.22+** (`GET /api/v1/leads/{id}`).

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` | Liveness: `{ "status":"ok", "service":"sales-radar-api" }` |
| `GET` | `/api/v1/leads` | Filtered list + meta + optional `summary.last_run` |
| `GET` | `/api/v1/leads/{id}` | Single lead |
| `GET` | `/api/v1/settings` | Discovery toggles + ICP + catalogs |
| `PUT` | `/api/v1/settings` | Replace discovery + ICP (JSON body) |
| `POST` | `/api/v1/pipeline/run` | Run pipeline + `ReplaceAll` |
| `GET` | `/api/v1/debug` | Debug JSON (latest run + breakdown) |
| `GET` | `/api/v1/export.csv` | CSV export (same filters as list) |

**Legacy HTML-only routes** (still on `cmd/web`): `/`, `/leads`, `/leads/{id}`, `/settings`, `/run`, `/export.csv`, `/debug`, `/static/*`.

**Error envelope (JSON):**

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

Written by `internal/api/jsonerr`.

---

## 2. Query parameters — list / export

Shared parser: **`request.ParseListFilter`** (`internal/api/request/listfilter.go`).

| Param | Meaning |
|-------|---------|
| `q` | Search string (company / context) |
| `icp_match` | `high` \| `medium` \| `low` \| empty = any |
| `lead_status` | e.g. `new`, `needs_review`, `discarded` |
| `sales_status` | e.g. `qualified`, `partial_data`, `needs_manual_review` |
| `industry` | Exact match string |
| `action` | `Contact` \| `Research first` \| `Ignore` |
| `sort` | `priority` (default), `confidence`, `completeness`, `action`, `company` |
| `order` | `asc` (default) or `desc` |

**Pipeline echo params** (optional, for KPIs / `summary.last_run`): `candidates`, `stored`, `enriched`, `contact_ready`, `research_first`, `rejected`, `dupes`, `merged`, `int_g`, `int_a`, `int_l`, `providers`, `breakdown`, `bd_total`, `bd_ok`, `mode`, `src` — see `dto.OptionalPipelineSummaryFromQuery`.

---

## 3. Response shapes (JSON)

### 3.1 `GET /api/v1/leads`

```json
{
  "items": [ /* Lead */ ],
  "pagination": { "total": 0, "returned": 0 },
  "summary": { "last_run": { /* PipelineSummaryNumbers */ } },
  "meta": {
    "pipeline_has_run": true,
    "total_in_db": 0,
    "industries": ["banking", "..."]
  },
  "filter_echo": { "q": "", "sort": "priority", "order": "asc" }
}
```

`Lead` fields (snake_case) — see `internal/api/dto/dto.go` type `Lead`: `id`, `company`, `industry`, `icp_match`, `priority_score`, `action`, `sales_status`, `lead_status`, `why_now`, `source_trace`, `used_google`, etc.

### 3.2 `GET /api/v1/leads/{id}`

```json
{ "lead": { /* Lead */ } }
```

404 if not found.

### 3.3 `POST /api/v1/pipeline/run`

```json
{
  "run": {
    "run_uuid": "...",
    "started_at": "RFC3339",
    "finished_at": "...",
    "status": "...",
    "discovery_mode": "..."
  },
  "stats": { /* pipeline.RunStats */ },
  "provider_statuses": [ /* discovery.ProviderStatus */ ],
  "rows_persisted": 0
}
```

### 3.4 `GET /api/v1/settings`

```json
{
  "discovery_sources": {
    "google": true,
    "seed": true,
    "website_crawl": true,
    "job_signal": true,
    "apollo": true,
    "linkedin": true
  },
  "icp": { /* ICPForm */ },
  "catalogs": {
    "industries": [{ "id", "label", "helper" }],
    "signals": [...],
    "regions": [...],
    "weights": ["..."]
  }
}
```

### 3.5 `GET /api/v1/debug`

Structured debug payload: run meta, `summary`, `provider_rows`, `breakdown_rows`, `integration_rows`, flags — see `dto.DebugResponse` in `internal/api/dto/dto.go`.

---

## 4. Core persistence — conceptual model

### 4.1 `store.Lead` (UI row)

Primary fields used in API/UI: identifiers; **company**; **industry**; **icp_match**; **duplicate_status**; **lead_status**; **sales_status**; **action**; **priority_score**; **data_completeness**; **confidence**; narratives (**summary**, **reason_for_fit**, **why_now**, **why_now_strength**, **sales_angle**, **accept_explanation**); **reasons**[], **missing_optional**[]; **source**, **source_ref**; **official_domain**, **website_domain**, **linkedin_url**, **country_region**; **employee_size**, **size**; **sales_ready**; **source_trace**[]; **used_google/apollo/linkedin**; **created_at** (RFC3339 in JSON).

### 4.2 Pipeline run

- Table **`pipeline_runs`**: `run_uuid`, timestamps, `status`, `discovery_mode`, **`run_debug_json`** (serialized `pipeline.RunStats`).

### 4.3 Replace semantics

- **`store.ReplaceAll`**: deletes prior lead-related rows, inserts new snapshot, finalizes run row — **one** coherent generation visible to UI per successful run.

---

## 5. Environment variables (backend)

| Variable | Role |
|----------|------|
| `SALESRADAR_GOOGLE_API_KEY`, `SALESRADAR_GOOGLE_CX` | Google Custom Search |
| `SALESRADAR_APOLLO_API_KEY` | Apollo enrichment |
| `DISCOVERY_MODE` | Discovery mode string |
| `SALESRADAR_ENABLE_WEBSITE_CRAWL`, `SALESRADAR_ENABLE_JOB_SIGNAL` | Feature toggles (see `internal/discovery`) |
| `SALESRADAR_USE_MOCK_DISCOVERY` | Deterministic mock (tests / offline) |

**Frontend:** `NEXT_PUBLIC_API_BASE_URL` — base URL of Go API (no trailing slash).

---

## 6. CORS (`cmd/api`)

Optional flag **`-cors <origin>`** sets `Access-Control-Allow-Origin` and handles `OPTIONS` for browser clients (e.g. Next on `localhost:3000` or `https://sales.bawana.xyz`).

---

*Last updated April 2026. For SQL-level discovery schema, see `docs/discovery_engine_schema.md`.*
