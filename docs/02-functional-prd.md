# Sales Radar — Functional PRD (Consolidated)

**Version:** April 2026 · **Audience:** Product, engineering, QA · **Normative sources:** `PRD-Sales-Radar-v2.md`, `internal/*`, `frontend/app/*`

This document merges requirements for the **legacy HTML UI** (`cmd/web/templates`), the **JSON API** (`internal/api`), and the **Next.js** app (`frontend/`). Where behavior differs only by transport (HTML vs JSON), it is stated once under **behavior**, with **surface notes**.

---

## 1. Navigation & shell

| ID | Requirement | HTML | Next | API |
|----|-------------|------|------|-----|
| NAV-1 | `/` redirects to lead list entry (`/leads`). | ✓ | ✓ | — |
| NAV-2 | Persistent sidebar: Lead list, Settings; Debug accessible from list meta / nav. | ✓ | ✓ | — |
| NAV-3 | HTML: collapsible sidebar, logo asset, `localStorage` key `salesradar.sidebarCollapsed`. | ✓ | Optional / simplified shell | — |
| NAV-4 | Active route indication for current page. | ✓ | ✓ | — |

---

## 2. Settings

| ID | Requirement | Notes |
|----|-------------|--------|
| SET-1 | Discovery source toggles: Google, Seed, Website crawl, Job signal, Apollo, LinkedIn (`domain.DiscoverySourceToggles`). | Persisted in SQLite via `store.SetDiscoverySourceToggles`. |
| SET-2 | ICP form: target industries (catalog), region focus, signals, weights, employee bounds, sub-50 rule, etc. (`store.ICPFormSettings`). | Normalized on read/write (`internal/store/icp_form.go`). |
| SET-3 | Integration **badges** (configured / missing / disabled) for operators. | HTML: template badges; API: `GET /api/v1/settings` includes `catalogs`. |
| SET-4 | **HTML:** `GET/POST /settings` (form post, redirect). | Legacy |
| SET-5 | **API:** `GET /api/v1/settings`, `PUT /api/v1/settings` (JSON body mirrors GET shape). | New |
| SET-6 | **Next:** `/settings` page; Phase 4+ may add full PUT UX. | Scaffold / evolving |

---

## 3. Generate Leads (pipeline run)

| ID | Requirement | Notes |
|----|-------------|--------|
| RUN-1 | Invoke `pipeline.RunWithQualityGate` with `DefaultRunParams()`, toggles + ICP from DB. | Shared by `/run`, `POST /api/v1/pipeline/run`. |
| RUN-2 | Map output through `store.FromStaged` + `store.ReplaceAll`; persist `RunStats` JSON into pipeline run record. | Single coherent snapshot per success. |
| RUN-3 | **HTML:** `POST /run` → redirect to `/leads?…` with run summary query params **or** JSON if `Accept: application/json`. | Legacy list merges URL client-side. |
| RUN-4 | **API:** `POST /api/v1/pipeline/run` → JSON `{ run, stats, provider_statuses, rows_persisted }`. | Next **Generate** button merges stats into `/leads` URL query for KPI echo. |
| RUN-5 | Candidate limits: default **40**, cap **100** (`MaxLeadsPerRunDefault`, `MaxLeadsPerRunCap`). | Product constants |

---

## 4. Lead list

| ID | Requirement | Notes |
|----|-------------|--------|
| LST-1 | Filters aligned with `store.ListFilter`: `q`, `icp_match`, `lead_status`, `sales_status`, `industry`, `action`, `sort`, `order`. | `internal/api/request/listfilter.go` shared with HTML paths. |
| LST-2 | Sort: `priority` (default), `confidence`, `completeness`, `action`, `company`. | |
| LST-3 | KPI strip when run summary present (HTML: URL params; API: `summary.last_run` when query contains legacy keys; Next merges pipeline keys after run). | |
| LST-4 | **Readiness** badge (Ready / Almost ready / Not ready), **priority** pill (High / Medium / Low), **action** label, **signal** preview (why-now / strength). | Logic mirrored in `frontend/lib/lead-display.ts` for Next. |
| LST-5 | **HTML:** lead **cards**, **drawer** on row click, row menu, active row styling. | Full parity in PRD v2 |
| LST-6 | **Next:** table layout + link to detail; drawer optional / Phase 4. | |
| LST-7 | **Export CSV** uses **same query string** as list (`/export.csv` or `/api/v1/export.csv`). | |
| LST-8 | Link to **Debug** with optional query echo. | |
| LST-9 | Empty states: no pipeline run yet; DB empty; no filter matches. | |

---

## 5. Lead detail

| ID | Requirement |
|----|-------------|
| DET-1 | Single lead by ID (`store.Get`); read-only fields; brand-consistent styling. |
| DET-2 | **HTML:** `/leads/{id}` template. |
| DET-3 | **API:** `GET /api/v1/leads/{id}` → `{ "lead": {…} }`. |
| DET-4 | **Next:** `/leads/[id]` (Phase 4 can deepen layout). |
| DET-5 | Invalid ID → **404** (HTML not found; API JSON error envelope). |

---

## 6. CSV export

| ID | Requirement |
|----|-------------|
| EXP-1 | Same filter parsing as list. |
| EXP-2 | Column set includes company, domains, scoring, statuses, reasons, `source_trace`, integration flags, timestamps (see `internal/api/exportcsv` / legacy handler). |
| EXP-3 | Filename `leads_export_YYYYMMDD.csv` (UTC). |

---

## 7. Debug / Ops

| ID | Requirement |
|----|-------------|
| DBG-1 | Load `store.LatestPipelineRun`; decode `RunDebugJSON` → `pipeline.RunStats` when present. |
| DBG-2 | Show run metadata, integration rows (Google / Apollo / LinkedIn), per-source breakdown, provider statuses. |
| DBG-3 | **API:** `GET /api/v1/debug` JSON (`internal/api/debugview` + handlers). |
| DBG-4 | **HTML:** `/debug` template. |
| DBG-5 | **Next:** `/debug` scaffold. |

---

## 8. Non-functional

| Area | Expectation |
|------|-------------|
| **Deployment** | Go: single binary + SQLite file path; flags `-addr`, `-db`. Next: Vercel or Node; `NEXT_PUBLIC_API_BASE_URL` for API origin. |
| **Security** | No default auth; CORS on `cmd/api` when browser calls cross-origin. Secrets only on API host (`SALESRADAR_*`). |
| **Performance** | List is full result set (no paging v1); indexed SQLite queries in `store.List`. |
| **A11y** | HTML: keyboard rows, drawer Escape, focus rings; Next: follow same patterns as components evolve. |

---

## 9. Backlog (not required for “current” acceptance)

- AuthN/AuthZ, audit logs.
- Per-lead delete/discard with DB update.
- Editable leads / CRM sync from UI.
- API pagination (`limit`/`offset`).
- Full drawer parity on Next list.

---

*Consolidated April 2026. For the original HTML-only PRD narrative, see `PRD-Sales-Radar-v2.md`.*
