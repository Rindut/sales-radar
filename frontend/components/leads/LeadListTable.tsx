import Link from "next/link";

import type { Lead } from "@/lib/api-types";
import {
  actionLabel,
  displayDomain,
  priorityBand,
  priorityLabel,
  readinessKind,
  readinessLabel,
  signalPreview,
  sourceLabel,
} from "@/lib/lead-display";

function ReadinessBadge({ lead }: { lead: Lead }) {
  const k = readinessKind(lead);
  const cls =
    k === "ready"
      ? "border-emerald-200 bg-emerald-50 text-emerald-900"
      : k === "almost"
        ? "border-amber-200 bg-amber-50 text-amber-950"
        : "border-slate-200 bg-slate-100 text-slate-700";
  return (
    <span
      className={`inline-flex whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-semibold ${cls}`}
    >
      {readinessLabel(lead)}
    </span>
  );
}

function PriorityPill({ lead }: { lead: Lead }) {
  const b = priorityBand(lead);
  const cls =
    b === "high"
      ? "border-emerald-200 bg-emerald-50 text-emerald-900"
      : b === "medium"
        ? "border-amber-200 bg-amber-50 text-amber-900"
        : "border-slate-200 bg-slate-100 text-slate-600";
  return (
    <span
      className={`inline-flex whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-bold ${cls}`}
    >
      {priorityLabel(lead)}
    </span>
  );
}

function ActionLabel({ lead }: { lead: Lead }) {
  const a = actionLabel(lead);
  const cls =
    a === "Contact now"
      ? "text-emerald-800"
      : a === "Ignore"
        ? "text-slate-500"
        : "text-slate-700";
  return <span className={`text-xs font-medium ${cls}`}>{a}</span>;
}

export function LeadListTable({ items }: { items: Lead[] }) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white shadow-sm">
      <table className="min-w-[900px] w-full border-collapse text-left text-sm">
        <thead>
          <tr className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
            <th className="px-3 py-2">#</th>
            <th className="px-3 py-2">Company</th>
            <th className="px-3 py-2">Industry</th>
            <th className="px-3 py-2">Source</th>
            <th className="px-3 py-2">Signal</th>
            <th className="px-3 py-2">Readiness</th>
            <th className="px-3 py-2">Priority</th>
            <th className="px-3 py-2">Action</th>
          </tr>
        </thead>
        <tbody>
          {items.map((lead, i) => (
            <tr
              key={lead.id}
              className="border-b border-slate-100 hover:bg-primary-soft/40"
            >
              <td className="px-3 py-3 align-top text-xs tabular-nums text-slate-400">
                {i + 1}
              </td>
              <td className="px-3 py-3 align-top">
                <Link
                  href={`/leads/${lead.id}`}
                  className="font-medium text-primary hover:underline"
                >
                  {lead.company?.trim() || "—"}
                </Link>
                <div className="mt-0.5 font-mono text-xs text-slate-500">
                  {displayDomain(lead)}
                </div>
              </td>
              <td className="px-3 py-3 align-top text-slate-700">
                {lead.industry?.trim() || "—"}
              </td>
              <td className="px-3 py-3 align-top text-xs text-slate-700">
                {sourceLabel(lead)}
              </td>
              <td className="max-w-[220px] px-3 py-3 align-top text-xs text-slate-600">
                <span title={lead.why_now || undefined}>{signalPreview(lead)}</span>
              </td>
              <td className="px-3 py-3 align-top">
                <ReadinessBadge lead={lead} />
              </td>
              <td className="px-3 py-3 align-top">
                <PriorityPill lead={lead} />
              </td>
              <td className="px-3 py-3 align-top">
                <ActionLabel lead={lead} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
