# Sales Radar — Product Overview

**Version:** April 2026 · **Status:** Living document · **Code:** `salesradar` module, repository *BAWANA Sales Radar*

---

## 1. What Sales Radar is

**Sales Radar** is an internal B2B tool for **discovering, scoring, and reviewing sales leads** in one workflow. Operators run a single **Generate Leads** action that executes a full pipeline—multi-source discovery → extraction → ICP scoring → deduplication → status → human-readable review—and **persists a fresh snapshot** of qualified leads to a local database for list, detail, export, and ops/debug views.

**Who it serves**

| Persona | Outcome |
|---------|---------|
| **Sales / SDR** | Prioritized leads with clear **action** (Contact / Research first / Ignore), narrative context (why fit, why now, sales angle), and CSV export for outreach workflows. |
| **RevOps / operator** | Toggles for discovery sources, visibility into **pipeline run stats** and integration usage on Debug. |
| **Engineering** | Same domain logic in **Go** (`internal/*`), consumable via legacy **server-rendered UI**, **JSON API**, or **CLI**. |

---

## 2. Product principles

1. **Trust the pipeline** — List and detail views reflect **derived** readiness and actions from structured review data, not arbitrary spreadsheet edits in the UI.
2. **Operational clarity** — Run metrics, per-source breakdown, and provider statuses are **persisted** (`RunDebugJSON`) and surfaced on Debug/Ops.
3. **Restrained UI** — Brand blue (`#216ab7`) is used for **actions, links, focus, and key accents**, not full-screen color blocks.

---

## 3. Scope

### In scope (current)

- Multi-source **discovery** (Google CSE, seed, website crawl, job signal, Apollo enrichment, LinkedIn URLs via Apollo) controlled by **Settings** toggles and environment configuration.
- **Quality-gated** pipeline run with configurable **ICP** settings persisted in SQLite.
- **Replace-all** persistence model: each successful run **replaces** the main `leads` table (plus related rows) with the new snapshot.
- **Lead list**: search, filters, sort, KPIs after a run, CSV export matching filters, link to Debug.
- **Lead detail**: read-only full view per lead ID.
- **Settings**: discovery sources + ICP form (catalog-driven).
- **Debug / Ops**: latest run metadata, integration matrix, discovery breakdown.
- **JSON API** (`/api/v1/*`) for the **Next.js** frontend (`frontend/`) and automation.
- **CLI** (`cmd/salesradar`) for JSON/CSV/CRM-shaped output without the web server.

### Out of scope (as shipped)

- Multi-user **authentication / RBAC** in default servers (`cmd/web`, `cmd/api`).
- **In-app delete** of leads (menu may show Delete as disabled).
- **Live Odoo write** on the quality-gated web path (Odoo client is a **noop** there; CRM types remain in the domain model).
- Offset/limit **pagination** in API list (full filtered set returned today).

---

## 4. Delivery surfaces

| Surface | Role |
|---------|------|
| **`cmd/web`** | Monolith: embedded HTML templates + static assets + **same JSON API** mounted under `/api/v1`. Default `:8080`, SQLite `data/salesradar.db`. |
| **`cmd/api`** | API-only binary for **`api.sales.bawana.xyz`** (or local dev); optional **CORS** flag. |
| **`frontend/`** | **Next.js 15** App Router + Tailwind; targets **`sales.bawana.xyz`**; uses `NEXT_PUBLIC_API_BASE_URL`. |
| **`cmd/salesradar`** | Pipeline to stdout (no DB replace for the default flow described in CLI help). |

Canonical product requirements for the **original** HTML UI live in **`PRD-Sales-Radar-v2.md`**; this overview aligns with that PRD and the **current** split-frontend architecture.

---

## 5. Key business rules (summary)

- Default **candidate budget** per run: **`MaxLeadsPerRunDefault` = 40**, hard cap **`MaxLeadsPerRunCap` = 100** (`internal/domain/constants.go`).
- **Google / Apollo / LinkedIn** hosts are **not** treated as official company domains in copy and domain rules where applicable.
- Export filename pattern: **`leads_export_YYYYMMDD.csv`** (UTC).

---

## 6. Glossary

| Term | Meaning |
|------|---------|
| **ICP** | Ideal Customer Profile — match tier (`high` / `medium` / `low` / `partial` / `no`), scoring weights, segment exclusions. |
| **Quality gate** | Pipeline stage that drops rows that must not be stored; reflected in **rejected** counts. |
| **ReplaceAll** | Transactional wipe + insert of lead rows after a successful run (`store.ReplaceAll`). |
| **RunDebugJSON** | JSON blob of `pipeline.RunStats` attached to the latest `pipeline_runs` row for Debug. |

---

## 7. Related documents

| Document | Content |
|----------|---------|
| `PRD-Sales-Radar-v2.md` | Detailed functional PRD for the legacy HTML product. |
| `docs/02-functional-prd.md` | Consolidated functional view (HTML + API + Next). |
| `docs/03-system-architecture.md` | Components and deployment. |
| `docs/04-ui-ux-notes.md` | Visual and interaction patterns. |
| `docs/05-api-and-data-model.md` | Endpoints and schema. |
| `docs/06-test-scenarios.md` | Verification scenarios. |
| `docs/discovery_engine_schema.md` | Deeper persistence/discovery schema reference. |

---

*Last updated: April 2026 — aligned with Go 1.22+, Next.js 15 frontend, and `/api/v1` handlers in `internal/api`.*
