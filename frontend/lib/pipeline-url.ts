/**
 * Merge POST /api/v1/pipeline/run stats into the /leads URL (parity with legacy list.html).
 */

export const PIPELINE_QUERY_KEYS = [
  "candidates",
  "enriched",
  "contact_ready",
  "research_first",
  "rejected",
  "dupes",
  "merged",
  "stored",
  "int_g",
  "int_a",
  "int_l",
  "providers",
  "breakdown",
  "bd_total",
  "bd_ok",
  "mode",
  "src",
] as const;

export type PipelineRunStatsJSON = {
  candidates_found: number;
  enriched: number;
  contact_ready: number;
  research_first: number;
  rejected: number;
  duplicates_removed: number;
  semantic_merged: number;
  rows_stored: number;
  integration_google_used?: boolean;
  integration_apollo_used?: boolean;
  integration_linkedin_used?: boolean;
  provider_statuses?: unknown[];
  source_breakdown?: unknown[];
  breakdown_generated_total?: number;
  breakdown_matches_total?: boolean;
  discovery_mode?: string;
  discovery_source?: string;
};

/** Returns pathname + search for Next.js router, e.g. `/leads?...` */
export function mergePipelineRunIntoPath(
  currentPathname: string,
  currentSearch: string,
  stats: PipelineRunStatsJSON,
  rowsPersisted: number
): string {
  const u = new URLSearchParams(currentSearch);
  for (const k of PIPELINE_QUERY_KEYS) {
    u.delete(k);
  }
  u.set("candidates", String(stats.candidates_found));
  u.set("enriched", String(stats.enriched));
  u.set("contact_ready", String(stats.contact_ready));
  u.set("research_first", String(stats.research_first));
  u.set("rejected", String(stats.rejected));
  u.set("dupes", String(stats.duplicates_removed));
  u.set("merged", String(stats.semantic_merged));
  u.set("stored", String(rowsPersisted));
  u.set("int_g", stats.integration_google_used ? "1" : "0");
  u.set("int_a", stats.integration_apollo_used ? "1" : "0");
  u.set("int_l", stats.integration_linkedin_used ? "1" : "0");
  u.set("providers", JSON.stringify(stats.provider_statuses ?? []));
  u.set("breakdown", JSON.stringify(stats.source_breakdown ?? []));
  u.set("bd_total", String(stats.breakdown_generated_total ?? 0));
  u.set("bd_ok", String(stats.breakdown_matches_total ?? false));
  u.set("mode", String(stats.discovery_mode ?? ""));
  u.set("src", String(stats.discovery_source ?? ""));
  const q = u.toString();
  return q ? `${currentPathname}?${q}` : currentPathname;
}
