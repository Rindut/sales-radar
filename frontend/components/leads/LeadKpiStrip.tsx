import type { PipelineSummaryNumbers } from "@/lib/api-types";

export function LeadKpiStrip({
  lastRun,
  showNoDataHint,
  showEmptyFilterHint,
}: {
  lastRun: PipelineSummaryNumbers | undefined;
  showNoDataHint: boolean;
  showEmptyFilterHint: boolean;
}) {
  if (lastRun) {
    return (
      <div className="space-y-3">
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm">
            <div className="text-xs font-medium text-slate-500">Total leads</div>
            <div className="text-2xl font-semibold tabular-nums text-slate-900">
              {lastRun.rows_stored}
            </div>
          </div>
          <div className="rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm">
            <div className="text-xs font-medium text-slate-500">Contact-ready</div>
            <div className="text-2xl font-semibold tabular-nums text-emerald-800">
              {lastRun.contact_ready}
            </div>
          </div>
          <div className="rounded-lg border border-slate-200 bg-white px-4 py-3 shadow-sm">
            <div className="text-xs font-medium text-slate-500">Pending review</div>
            <div className="text-2xl font-semibold tabular-nums text-amber-900">
              {lastRun.research_first}
            </div>
          </div>
        </div>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-600">
          <span>
            <strong className="text-slate-700">Rejected:</strong> {lastRun.rejected}
          </span>
          <span>
            <strong className="text-slate-700">Duplicates removed:</strong>{" "}
            {lastRun.duplicates_removed}
          </span>
          <span>
            <strong className="text-slate-700">Merged:</strong> {lastRun.semantic_merged}
          </span>
        </div>
      </div>
    );
  }

  if (showNoDataHint) {
    return (
      <p className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
        No data yet. Click <strong>Generate leads</strong> to start discovering companies.
      </p>
    );
  }

  if (showEmptyFilterHint) {
    return (
      <p className="rounded-lg border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
        No leads found. Try enabling more sources in Settings or adjusting filters.
      </p>
    );
  }

  return null;
}
