# Discovery Engine Schema (Production)

## Tables and relationships

- `pipeline_runs` is the top-level execution record for each **Generate Leads** run.
- `raw_candidates` stores source output **before** normalization/dedup; each row belongs to one `pipeline_runs` row.
- `companies` is the canonical entity table after normalization + dedup/entity resolution.
- `company_sources` preserves per-run attribution from company to one or more discovery sources (optionally linked back to `raw_candidates`).
- `company_signals` stores intent/why-now evidence per company per run.
- `company_scores` stores versioned scoring results (`scoring_version`) per company per run.
- `company_snapshots` is the denormalized UI read model (one row per company per run per scoring version) to avoid heavy joins.

## Cardinality

- `pipeline_runs` 1 -> N `raw_candidates`
- `pipeline_runs` 1 -> N `company_sources`
- `pipeline_runs` 1 -> N `company_signals`
- `pipeline_runs` 1 -> N `company_scores`
- `pipeline_runs` 1 -> N `company_snapshots`
- `companies` 1 -> N `company_sources`
- `companies` 1 -> N `company_signals`
- `companies` 1 -> N `company_scores`
- `companies` 1 -> N `company_snapshots`

## Performance strategy

- UI reads primarily from `company_snapshots`.
- Required filter indexes are present on:
  - `company_snapshots.action`
  - `company_snapshots.priority_score`
  - `company_snapshots.industry`
  - `company_snapshots.pipeline_run_id`
- Additional indexes support run debugging and source-level drilldowns.

## Migration files

- `db/migrations/001_discovery_engine_schema.sql`
- `db/migrations/002_discovery_engine_indexes.sql`

