/**
 * Builds distinct "Summary" vs "Why this lead fits" copy from the same API lead.
 * When summary and reason_for_fit are duplicated (common from the pipeline), Summary
 * becomes a short sales decision snapshot; the long ICP narrative stays under Why.
 */

import type { Lead } from "@/lib/api-types";
import { actionLabel, priorityLabel, readinessLabel } from "@/lib/lead-display";

function norm(s: string): string {
  return s.trim().toLowerCase().replace(/\s+/g, " ");
}

/** True when API sent the same (or nearly same) text for summary and ICP reason. */
export function isDuplicateSummaryAndReason(lead: Lead): boolean {
  const a = norm(lead.summary ?? "");
  const b = norm(lead.reason_for_fit ?? "");
  if (!a || !b) return false;
  if (a === b) return true;
  if (a.length >= 40 && b.length >= 40 && (a.includes(b) || b.includes(a))) {
    return true;
  }
  return false;
}

/**
 * Short, decision-oriented snapshot for the sales team (English).
 * Uses only fields already on the lead — no backend changes.
 */
export function buildSalesDecisionSummary(lead: Lead): string {
  const company = (lead.company ?? "").trim() || "—";
  const industry = (lead.industry ?? "").trim();
  const line1 = industry ? `${company} · ${industry}` : company;

  const icp = (lead.icp_match ?? "").trim() || "—";
  const act = actionLabel(lead);
  const ready = readinessLabel(lead);
  const pri = priorityLabel(lead);
  const score = lead.priority_score;

  const line2 = `ICP match: ${icp} · Priority: ${pri} (${score}/100) · Next move: ${act} · Readiness: ${ready}.`;

  const line3 = `What to do: ${act}. ${
    ready === "Ready"
      ? "Good to engage."
      : ready === "Almost ready"
        ? "Gather one more proof point if needed before heavy outreach."
        : "Skip for now unless strategy shifts."
  }`;

  return [line1, line2, line3].join("\n\n");
}

/** Text shown in the Summary section (English source before optional ID translation). */
export function summaryForDrawer(lead: Lead): string {
  const sum = (lead.summary ?? "").trim();

  if (isDuplicateSummaryAndReason(lead)) {
    return buildSalesDecisionSummary(lead);
  }

  if (!sum) {
    return buildSalesDecisionSummary(lead);
  }

  const focus = `Decision focus: ${actionLabel(lead)} · ${readinessLabel(lead)} · ${priorityLabel(lead)} priority (${lead.priority_score}/100).`;
  return `${focus}\n\n${sum}`;
}

/** Text shown under "Why this lead fits" — ICP narrative only. */
export function whyFitForDrawer(lead: Lead): string {
  return (lead.reason_for_fit ?? "").trim();
}
