# Product Requirements Document — Sales Radar (v2)

**Document:** `PRD-Sales-Radar-v2.md` (repository root)  
**Document status:** Living PRD aligned with the codebase and UI as of **April 2026**.  
**Product:** Internal web application **Sales Radar** (`cmd/web`) backed by **SQLite**, sharing pipeline logic with the `salesradar` CLI.

---

## 1. Purpose & vision

Sales Radar helps operators and sellers **discover, score, and review B2B leads** in one place. A single **Generate Leads** action runs the full pipeline (discovery → extraction → ICP → dedup → status → review), **replaces persisted lead rows** with the latest run’s output, and surfaces results in a **filterable list** with quick **detail review** (drawer + optional full page).

**Principles**

- **Trust the pipeline:** Explanations, readiness, and actions are derived from structured review data, not ad-hoc edits in the list UI.
- **Operational clarity:** Run metrics, integration usage, and per-source breakdown are visible on the **Debug / Ops** page when a run is persisted.
- **Restraint in UI:** Brand color is used for actions, selection, and focus—not full-screen color washes.

---

## 2. Goals & non-goals

### 2.1 Goals

| ID | Goal |
|----|------|
| G1 | Run multi-source discovery with configurable **source toggles** and optional API keys (Google CSE, Apollo). |
| G2 | Persist **one coherent lead table** for the UI (`leads` + related rows), refreshed on each successful **Generate Leads** run. |
| G3 | Support **sales workflows**: sort/filter/search leads; open **detail drawer** from a row; optional **full-page detail** for a lead ID. |
| G4 | Export **CSV** that respects the **same filters** as the list view (`GET /export.csv`). |
| G5 | Provide **Settings** for discovery source enablement and **Debug** for pipeline run telemetry (DB-backed, not URL-only). |

### 2.2 Non-goals (current product)

- **No in-app lead delete** (row menu may show Delete as explicitly disabled).
- **No multi-user auth / RBAC** in the shipped `cmd/web` server.
- **No live Odoo write** from the quality-gated web path (Odoo client is a noop in the web pipeline path; CRM push is part of domain design but not required for UI acceptance here).
- **Full discovery engine relational schema** (`companies`, `company_snapshots`, etc.) is documented separately; the **UI read path** today is the denormalized **`leads`** table populated by `store.ReplaceAll`.

---

## 3. Users & primary scenarios

| Persona | Needs |
|---------|--------|
| **Sales / SDR** | Scan prioritized leads, open details, understand “why this company,” export a slice to CSV. |
| **RevOps / operator** | Toggle discovery sources, run generation, verify integrations and run stats on Debug. |
| **Engineering** | Inspect pipeline breakdown, provider status, and persisted `RunDebugJSON` for a run. |

**Core flows**

1. **First run:** Open app → **Settings** (optional) → **Lead List** → **Generate Leads** → review KPIs and list; consult **Debug** if something fails or counts look wrong.
2. **Daily use:** Filter/sort/search → open drawer → read decision / pitch / why-now → export or work through “Contact” vs “Research first” rows.
3. **Sidebar:** Expand/collapse navigation; preference stored in `localStorage` (`salesradar.sidebarCollapsed`).

---

## 4. Functional requirements

### 4.1 Application shell & navigation

| Req ID | Requirement |
|--------|-------------|
| NAV-1 | **Root** `/` redirects to `/leads`. |
| NAV-2 | Persistent **sidebar** on Lead List and Settings: **Lead List**, **Settings**; navigation matches current route (active state). |
| NAV-3 | **Collapsible sidebar:** collapsed width shows **icons + collapse control only** (no logo); expanded shows **Sales Radar logo** (image asset) without a framed “card” around the logo—logo sits on sidebar background with padding only. |
| NAV-4 | Toggle updates `html.sidebar-collapsed` and persists collapse state. |

### 4.2 Visual design system (implemented)

Brand is anchored on **primary blue `#216ab7`**, used sparingly.

| Token | Value | Usage |
|-------|--------|--------|
| Primary | `#216ab7` | Primary buttons, links, active nav, key accents |
| Primary hover | `#1b5a9b` | Button/link hover |
| Primary active | `#174c83` | Button active press |
| Primary soft | `#eaf3fb` | Selected rows, soft surfaces, focus rings (with border) |
| Primary accent | `#c7dbef` | Light borders/hover borders on cards |
| Page background | `#f8fafc` | App background |
| Card / sidebar bg | `#ffffff` | Surfaces |
| Text primary | `#0f172a` | Body emphasis |
| Text secondary | `#64748b` | Labels, muted text |

**Interaction:** Primary buttons, active sidebar item, links, input/select focus rings, keyboard focus on nav/menus/drawer, **active list row** when its drawer is open (left accent + soft background), and **info-style** badges (e.g. Settings) use the primary scale—not every panel.

### 4.3 Settings (`GET/POST /settings`)

| Req ID | Requirement |
|--------|-------------|
| SET-1 | Display **discovery source toggles**: Google, Seed, Website crawl, Job signal, Apollo, LinkedIn (domain: `DiscoverySourceToggles`). |
| SET-2 | **POST** saves toggles via `store.SetDiscoverySourceToggles` and redirects back to Settings. |
| SET-3 | Show human-readable **status badges** for integration readiness (e.g. Apollo key present vs missing), consistent with info/warn/danger badge pattern. |

### 4.4 Generate Leads (`POST /run`)

| Req ID | Requirement |
|--------|-------------|
| RUN-1 | Invoke **`pipeline.RunWithQualityGate`** with `pipeline.DefaultRunParams()` and **source toggles** loaded from DB. |
| RUN-2 | On success, map prepared rows through **`store.FromStaged`** + **`store.ReplaceAll`**, persisting a JSON **run stats** payload for later debug (`RunStats` / `RunDebugJSON`). |
| RUN-3 | **HTML:** redirect to `/leads` with query parameters summarizing the run (candidates, enriched, contact_ready, research_first, rejected, dupes, merged, stored, integration flags, provider JSON, breakdown JSON, discovery mode/source, etc.). |
| RUN-4 | **JSON:** If `Accept: application/json` (and XHR/fetch pattern as implemented), return JSON for clients that merge URL without full navigation. |
| RUN-5 | Default candidate budget per run is **`domain.MaxLeadsPerRunDefault` (40)** with cap **`MaxLeadsPerRunCap` (100)** per domain constants (pipeline enforcement). |

### 4.5 Lead list (`GET /leads`)

| Req ID | Requirement |
|--------|-------------|
| LST-1 | Render **KPI / summary** area using URL run params when present, plus **DB-backed** signals: e.g. whether any pipeline run exists, total row count. |
| LST-2 | Support **full-text-style search** and structured filters aligned with `store.ListFilter`: query `q`, `icp_match`, `lead_status`, `sales_status`, `industry`, `action`, `sort`, `order` (asc/desc). |
| LST-3 | **Sort columns** include priority, confidence, completeness, action (Contact ordering), company name. |
| LST-4 | **Empty states** are context-specific (no data yet vs no rows vs no filter matches), driven by server state, not static placeholder only. |
| LST-5 | Each data row is a **lead card**: company, industry, signal preview, **readiness** badge, **priority** pill, **action** label (Contact emphasized with primary color), row menu (kebab). |
| LST-6 | **Drawer** opens from row click/keyboard; shows decision callout, sales context, why-now, structured fields, badges; **backdrop** closes panel; **Escape** closes. |
| LST-7 | **Active row styling:** while the drawer is open for a lead, that row shows **primary soft fill + left border accent**; clearing selection on close. |
| LST-8 | **Export CSV** control passes through current filter query string so export matches the visible subset (see §4.7). |
| LST-9 | Link to **`/debug`** (“Debug” / ops) available from the list meta area for operators. |

### 4.6 Lead detail page (`GET /leads/{id}`)

| Req ID | Requirement |
|--------|-------------|
| DET-1 | Render a **print-friendly / shareable** HTML detail view for a single stored lead (`store.Get`). |
| DET-2 | Use the same **brand tokens** for links and primary callout styling as the main app. |
| DET-3 | **404** for invalid or missing IDs. |

### 4.7 CSV export (`GET /export.csv`)

| Req ID | Requirement |
|--------|-------------|
| EXP-1 | Apply **`parseListFilter`** (same as list page) so columns and row set match user expectations. |
| EXP-2 | Emit agreed column headers including company, domains, scoring, status fields, explanations, reasons, missing optional, source metadata, `source_trace`, integration flags, timestamps. |
| EXP-3 | Filename pattern: `leads_export_YYYYMMDD.csv` (UTC). |

### 4.8 Debug / Ops (`GET /debug`)

| Req ID | Requirement |
|--------|-------------|
| DBG-1 | Load **`store.LatestPipelineRun`**; if none, show empty/early state. |
| DBG-2 | Decode **persisted `RunDebugJSON`** into `pipeline.RunStats` when present; surface decode errors without crashing the page. |
| DBG-3 | Show **run metadata** (run id, UUID, timestamps, status, discovery mode) when available. |
| DBG-4 | Show **integration matrix** (Google, Apollo, LinkedIn) with config hints and “last run” interpretation from stats. |
| DBG-5 | Show **per-source breakdown** and **provider status** rows consistent with `RunStats` / discovery debug helpers in `main.go`. |

---

## 5. Data & persistence (UI-relevant)

### 5.1 Lead row (conceptual)

The UI **`store.Lead`** includes: identifiers; **company**; **industry**; size/employee; **ICP match**; duplicate status; **lead status** (`new`, `needs_review`, `discarded`, …); **sales status**; **action** (Contact / Research first / Ignore); **priority score**; **data completeness**; **confidence**; narrative fields (**summary**, **reason for fit**, **why now**, **why now strength**, **sales angle**, **accept explanation**); **missing optional** list; **reasons** list; **source** / **source ref**; **official domain**, **website domain**, **LinkedIn URL**, region; **sales ready** bool; **source trace**; **used_google / used_apollo / used_linkedin** flags; **created_at**.

### 5.2 Pipeline run record

Each **Generate Leads** execution persists run metadata and optional **debug JSON** for Debug page analytics (counts, breakdown, provider statuses, integration flags).

### 5.3 Discovery engine schema (reference)

Normalized **discovery engine** tables and snapshot strategy are described in **`docs/discovery_engine_schema.md`**. The **web UI list** may read from the simplified **`leads`** projection maintained by the app’s store layer; PRD acceptance for list/export is defined against **what the UI queries**, not against every internal migration table.

---

## 6. Integrations & configuration

| Integration | Role |
|-------------|------|
| **Google Custom Search** | Live discovery when API key + CX env vars set (`googlesearch.ConfigFromEnv`). |
| **Apollo** | Enrichment by domain when `SALESRADAR_APOLLO_API_KEY` set; LinkedIn company URLs may flow from Apollo. |
| **LinkedIn** | Not site-scraped; URLs validated when returned by enrichment. |
| **Seed / website crawl / job signal** | Additional discovery modes per toggles and orchestrator. |

Settings and Debug copy should **never** treat google.com / linkedin.com / apollo.io as **official company domains** (enforced in domain rules and UI messaging where applicable).

---

## 7. Pipeline semantics (summary)

High-level stages (see `internal/pipeline/run_result.go` and related packages):

1. **Discovery** — `discovery.DiscoverWithStatus` produces candidates + provider statuses + mode/source.  
2. **Extraction** — Structured fields from raw context.  
3. **Enrichment** — Apollo/domain enrichment where configured.  
4. **ICP** — Match tier, reasons, score, recommended action.  
5. **Dedup** — Exact / semantic classification (stats feed duplicates removed / merged).  
6. **Status** — Lead status assignment (`internal/status`).  
7. **Review** — **`internal/review`** builds human-readable review payload and acceptance explanations.  
8. **Quality gate** — Rows that should not be stored are filtered before `ReplaceAll` (rejected counts reflected in stats).

**StagedOdooLead** / **Odoo** types remain in the domain model; the web path uses a **noop** CRM client so the UI does not depend on Odoo uptime.

---

## 8. Non-functional requirements

| Area | Expectation |
|------|-------------|
| **Deployment** | Single binary + SQLite file; default listen `:8080`, DB path `data/salesradar.db` (configurable flags). |
| **Performance** | List queries use indexed/filtered SQL via `store.List`; avoid N+1 in templates (embed detail blobs in list as implemented). |
| **Accessibility** | Keyboard operable list rows, drawer close, focus visible styles, semantic roles for menus. |
| **Theming** | CSS variables in templates; Inter font; consistent spacing and radii; reduced-motion respected for sidebar transition. |

---

## 9. Open items / backlog (not in PRD as shipped)

Items below are **not** required for “implemented” status but are natural follow-ons:

- User authentication and audit logging.
- Per-lead **delete** or **discard** from UI with DB update.
- Editable fields / CRM sync from the web UI.
- **IncludeIDs** or saved views (if added to `ListFilter`, document query param contract).
- Unifying list reads exclusively on `company_snapshots` without `leads` projection.

---

## 10. Document history

| Version | Summary |
|---------|---------|
| 2026-04 v2 | Canonical filename **`PRD-Sales-Radar-v2.md`** at repo root; content matches **quality-gated web pipeline**, **Settings toggles**, **list + drawer + detail + export + debug**, **collapsible sidebar & logo behavior**, and **blue brand theme (`#216ab7`)**. |

---

*This document (`PRD-Sales-Radar-v2.md`) is the v2 source of truth for product intent; implementation details live in code and in `docs/discovery_engine_schema.md` where noted.*
