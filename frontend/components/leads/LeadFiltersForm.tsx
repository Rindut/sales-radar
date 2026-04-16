import Link from "next/link";

import { PIPELINE_QUERY_KEYS } from "@/lib/pipeline-url";
import { spGet } from "@/lib/search-params";

type FilterValues = {
  q: string;
  icp_match: string;
  sales_status: string;
  action: string;
  lead_status: string;
  industry: string;
  sort: string;
  order: string;
};

function buildFilterValues(
  sp: Record<string, string | string[] | undefined>
): FilterValues {
  return {
    q: spGet(sp, "q"),
    icp_match: spGet(sp, "icp_match"),
    sales_status: spGet(sp, "sales_status"),
    action: spGet(sp, "action"),
    lead_status: spGet(sp, "lead_status"),
    industry: spGet(sp, "industry"),
    sort: spGet(sp, "sort") || "priority",
    order: spGet(sp, "order") || "asc",
  };
}

export function buildPreservePipelineHidden(
  sp: Record<string, string | string[] | undefined>
): { name: string; value: string }[] {
  const out: { name: string; value: string }[] = [];
  for (const k of PIPELINE_QUERY_KEYS) {
    const v = spGet(sp, k);
    if (v) out.push({ name: k, value: v });
  }
  return out;
}

export function LeadFiltersForm({
  searchParams,
  industries,
}: {
  searchParams: Record<string, string | string[] | undefined>;
  industries: string[];
}) {
  const v = buildFilterValues(searchParams);
  const preserve = buildPreservePipelineHidden(searchParams);

  return (
    <details className="group rounded-lg border border-slate-200 bg-white shadow-sm">
      <summary className="cursor-pointer list-none px-4 py-3 text-sm font-medium text-slate-800 [&::-webkit-details-marker]:hidden">
        <span className="inline-flex items-center gap-2">
          <span aria-hidden>▸</span>
          Filters &amp; sorting
        </span>
      </summary>
      <div className="space-y-4 border-t border-slate-200 p-4">
        <form method="get" action="/leads" className="space-y-4">
          {preserve.map((h) => (
            <input key={h.name} type="hidden" name={h.name} value={h.value} />
          ))}
          <label className="block">
            <span className="mb-1 block text-xs font-medium text-slate-600">
              Search company or domain
            </span>
            <input
              name="q"
              type="search"
              defaultValue={v.q}
              autoComplete="off"
              placeholder="Search…"
              className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-primary focus:border-primary focus:ring-1"
            />
          </label>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <label className="block text-xs font-medium text-slate-600">
              ICP match
              <select
                name="icp_match"
                defaultValue={v.icp_match}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="">any</option>
                <option value="high">high</option>
                <option value="medium">medium</option>
                <option value="low">low</option>
              </select>
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Sales status
              <select
                name="sales_status"
                defaultValue={v.sales_status}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="">any</option>
                <option value="qualified">qualified</option>
                <option value="partial_data">partial_data</option>
                <option value="needs_manual_review">needs_manual_review</option>
              </select>
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Action
              <select
                name="action"
                defaultValue={v.action}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="">any</option>
                <option value="Contact">Contact</option>
                <option value="Research first">Research first</option>
                <option value="Ignore">Ignore</option>
              </select>
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Lead status
              <select
                name="lead_status"
                defaultValue={v.lead_status}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="">any</option>
                <option value="new">new</option>
                <option value="needs_review">needs_review</option>
                <option value="discarded">discarded</option>
              </select>
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Industry
              <select
                name="industry"
                defaultValue={v.industry}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="">any</option>
                {industries.map((ind) => (
                  <option key={ind} value={ind}>
                    {ind}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Sort by
              <select
                name="sort"
                defaultValue={v.sort}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="priority">priority score</option>
                <option value="action">action (Contact first)</option>
                <option value="company">company</option>
                <option value="completeness">data completeness</option>
                <option value="confidence">confidence</option>
              </select>
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Order
              <select
                name="order"
                defaultValue={v.order === "desc" ? "desc" : "asc"}
                className="mt-1 w-full rounded-lg border border-slate-200 bg-white px-2 py-2 text-sm text-slate-900"
              >
                <option value="asc">ascending</option>
                <option value="desc">descending</option>
              </select>
            </label>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="submit"
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-hover"
            >
              Apply filters
            </button>
            <Link
              href="/leads"
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            >
              Clear filters
            </Link>
          </div>
        </form>
      </div>
    </details>
  );
}
