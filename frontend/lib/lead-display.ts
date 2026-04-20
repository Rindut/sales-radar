/**
 * View helpers mirroring cmd/web template logic (readiness, priority, labels).
 */

import type { Lead } from "./api-types";

export type ReadinessKind = "ready" | "almost" | "not";

export function readinessKind(lead: Lead): ReadinessKind {
  if (lead.lead_status === "discarded") return "not";
  const action = (lead.action || "").trim();
  if (action === "Contact") return "ready";
  if (action === "Research first") return "almost";
  if (action === "Ignore") return "not";
  if (lead.sales_ready === true) return "ready";
  const ss = (lead.sales_status || "").trim();
  if (ss === "qualified") return "almost";
  if (ss === "partial_data" || ss === "needs_manual_review") return "almost";
  return "not";
}

export function readinessLabel(lead: Lead): string {
  const k = readinessKind(lead);
  if (k === "ready") return "Ready";
  if (k === "almost") return "Almost ready";
  return "Not ready";
}

export function priorityBand(lead: Lead): "high" | "medium" | "low" {
  const icp = (lead.icp_match || "").toLowerCase();
  if (icp === "high" || lead.priority_score >= 70) return "high";
  if (icp === "medium" || lead.priority_score >= 40) return "medium";
  return "low";
}

export function priorityLabel(lead: Lead): string {
  const b = priorityBand(lead);
  if (b === "high") return "High";
  if (b === "medium") return "Medium";
  return "Low";
}

export function actionLabel(lead: Lead): string {
  const a = (lead.action || "").trim();
  if (a === "Contact") return "Contact now";
  if (a === "Research first") return "Research first";
  if (a === "Ignore") return "Ignore";
  if (!a) return "—";
  return a;
}

function traceToLabel(trace: string): string {
  switch (trace.toLowerCase()) {
    case "google_discovery":
      return "Google";
    case "seed_discovery":
      return "Seed";
    case "directory_discovery":
      return "Directory";
    case "website_crawl_discovery":
      return "Website crawl";
    case "job_signal_discovery":
      return "Job signal";
    case "mock_discovery":
      return "Mock";
    case "apollo_enrichment":
      return "Apollo";
    case "linkedin_validation":
    case "linkedin_signal":
      return "LinkedIn";
    case "company_website_check":
      return "";
    default:
      return "";
  }
}

function sourceEnumLabel(src: string): string {
  switch ((src || "").toLowerCase()) {
    case "google":
      return "Google";
    case "linkedin":
      return "LinkedIn";
    case "apollo":
      return "Apollo";
    case "company_website":
      return "Company website";
    case "job_portal":
      return "Job portal";
    default:
      return src || "";
  }
}

export function sourceLabel(lead: Lead): string {
  const traces = lead.source_trace || [];
  const labels: string[] = [];
  const seen = new Set<string>();
  for (const t of traces) {
    const s = traceToLabel(String(t).trim());
    if (!s || seen.has(s)) continue;
    seen.add(s);
    labels.push(s);
  }
  if (labels.length > 0) return labels.join(" · ");
  return sourceEnumLabel(lead.source || "") || "—";
}

const MAX_SIGNAL = 52;

function truncateSignalText(s: string): string {
  const runes = Array.from(s);
  if (runes.length <= MAX_SIGNAL) return s;
  return runes.slice(0, MAX_SIGNAL).join("") + "…";
}

function primaryWebsiteEnrichmentText(lead: Lead): string | null {
  const status = (lead.website_enrichment_status || "").toLowerCase().trim();
  const signals = (lead.website_enrichment_signals || "").trim();
  const summary = (lead.website_enrichment_summary || "").trim();
  if (status === "success" || status === "legacy_fallback") {
    const crawl = signals || summary;
    if (crawl) return crawl;
  }
  if ((status === "failed" || status === "skipped") && summary) {
    return summary;
  }
  return null;
}

/** Website crawl / Firecrawl (or legacy HTTP) copy when enrichment produced usable text. */
export function signalPreview(lead: Lead): string {
  const crawl = primaryWebsiteEnrichmentText(lead);
  if (crawl) return truncateSignalText(crawl);
  const s = (lead.why_now || "").trim();
  if (!s) {
    const w = (lead.why_now_strength || "").toLowerCase().trim();
    if (w === "high") return "Strong urgency";
    if (w === "medium") return "Moderate urgency";
    if (w === "low") return "—";
    return "—";
  }
  return truncateSignalText(s);
}

/** Full text for hover: website signals/summary plus pipeline why-now when present. */
export function signalPreviewTooltip(lead: Lead): string | undefined {
  const parts: string[] = [];
  const signals = (lead.website_enrichment_signals || "").trim();
  const summary = (lead.website_enrichment_summary || "").trim();
  if (signals) parts.push(signals);
  if (summary && summary !== signals) parts.push(summary);
  const wn = (lead.why_now || "").trim();
  if (wn) parts.push(wn);
  if (parts.length === 0) return undefined;
  return parts.join(" — ");
}

export function displayDomain(lead: Lead): string {
  const o = (lead.official_domain || "").trim();
  if (o) return o;
  const w = (lead.website_domain || "").trim();
  if (w) return w;
  return "—";
}
