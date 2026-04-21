"use client";

import { useMemo, useState } from "react";

import { LeadDetailDrawer } from "@/components/leads/LeadDetailDrawer";
import type { Lead } from "@/lib/api-types";
import {
  actionLabel,
  displayDomain,
  priorityBand,
  priorityLabel,
  sourceLabel,
} from "@/lib/lead-display";

type PriorityBand = "high" | "medium" | "low";

function priorityRank(lead: Lead): number {
  const b = priorityBand(lead);
  if (b === "high") return 3;
  if (b === "medium") return 2;
  return 1;
}

function sortByPriority(items: Lead[]): Lead[] {
  return [...items].sort((a, b) => {
    const bandDiff = priorityRank(b) - priorityRank(a);
    if (bandDiff !== 0) return bandDiff;
    const scoreDiff = (b.priority_score || 0) - (a.priority_score || 0);
    if (scoreDiff !== 0) return scoreDiff;
    return (a.company || "").localeCompare(b.company || "");
  });
}

function PriorityPill({ lead }: { lead: Lead }) {
  const b = priorityBand(lead);
  const cls =
    b === "high"
      ? "border-emerald-200 bg-emerald-50 text-emerald-800"
      : b === "medium"
        ? "border-amber-200 bg-amber-50 text-amber-800"
        : "border-slate-200 bg-slate-100 text-slate-600";
  return (
    <div className="flex items-center gap-2">
      <span
        className={`inline-flex whitespace-nowrap rounded-full border px-2.5 py-0.5 text-xs font-bold ${cls}`}
      >
        {priorityLabel(lead)}
      </span>
      <span className="text-xs font-semibold tabular-nums text-slate-700">
        {lead.priority_score}/100
      </span>
    </div>
  );
}

function ActionLabel({ lead }: { lead: Lead }) {
  return (
    <span className="text-[11px] font-medium text-slate-500">
      {actionLabel(lead)}
    </span>
  );
}

function rowStyle(lead: Lead): string {
  const b = priorityBand(lead);
  if (b === "high") return "bg-emerald-50/60 hover:bg-emerald-50";
  if (b === "medium") return "bg-amber-50/45 hover:bg-amber-50/70";
  return "opacity-60 hover:bg-slate-50";
}

function leftAccent(lead: Lead): string {
  const b = priorityBand(lead);
  if (b === "high") return "border-l-[3px] border-emerald-500";
  if (b === "medium") return "border-l-[3px] border-amber-400";
  return "border-l-[3px] border-slate-200";
}

function groupTitle(band: PriorityBand): string {
  if (band === "high") return "High Priority";
  if (band === "medium") return "Medium Priority";
  return "Low Priority";
}

function groupTitleClass(band: PriorityBand): string {
  if (band === "high") return "text-emerald-800";
  if (band === "medium") return "text-amber-800";
  return "text-slate-500";
}

export function LeadListTable({ items }: { items: Lead[] }) {
  const [drawerLead, setDrawerLead] = useState<Lead | null>(null);

  const sorted = useMemo(() => sortByPriority(items), [items]);
  const grouped = useMemo(() => {
    const out: Record<PriorityBand, Lead[]> = { high: [], medium: [], low: [] };
    for (const lead of sorted) {
      out[priorityBand(lead)].push(lead);
    }
    return out;
  }, [sorted]);

  if (items.length === 0) {
    return null;
  }

  return (
    <>
      <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white shadow-sm">
        <table className="min-w-[980px] w-full border-collapse text-left text-sm">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
              <th className="px-3 py-2">#</th>
              <th className="px-3 py-2">Company</th>
              <th className="px-3 py-2">ICP Priority</th>
              <th className="px-3 py-2">Industry</th>
              <th className="px-3 py-2">Source</th>
              <th className="px-3 py-2">Action</th>
            </tr>
          </thead>
          {(["high", "medium", "low"] as PriorityBand[]).map((band) => {
            const leads = grouped[band];
            if (leads.length === 0) return null;
            return (
              <tbody key={band}>
                <tr className="border-y border-slate-200 bg-white">
                  <td colSpan={6} className={`px-3 py-2 text-xs font-semibold uppercase tracking-[0.4px] ${groupTitleClass(band)}`}>
                    {groupTitle(band)}
                  </td>
                </tr>
                {leads.map((lead, i) => (
                  <tr
                    key={lead.id}
                    className={`border-b border-slate-100 ${rowStyle(lead)}`}
                  >
                    <td className={`px-3 py-3 align-top text-xs tabular-nums text-slate-400 ${leftAccent(lead)}`}>
                      {sorted.indexOf(lead) + 1}
                    </td>
                    <td className="px-3 py-3 align-top">
                      <button
                        type="button"
                        onClick={() => setDrawerLead(lead)}
                        className="text-left font-medium text-primary hover:underline"
                      >
                        {lead.company?.trim() || "—"}
                      </button>
                      <div className="mt-0.5 font-mono text-xs text-slate-500">
                        {displayDomain(lead)}
                      </div>
                    </td>
                    <td className="px-3 py-3 align-top">
                      <PriorityPill lead={lead} />
                    </td>
                    <td className="px-3 py-3 align-top text-slate-700">
                      {lead.industry?.trim() || "—"}
                    </td>
                    <td className="px-3 py-3 align-top text-xs text-slate-700">
                      {sourceLabel(lead)}
                    </td>
                    <td className="px-3 py-3 align-top">
                      <ActionLabel lead={lead} />
                    </td>
                  </tr>
                ))}
              </tbody>
            );
          })}
        </table>
      </div>
      <LeadDetailDrawer lead={drawerLead} onClose={() => setDrawerLead(null)} />
    </>
  );
}
