import { PageHeader } from "@/components/PageHeader";
import { ApiError, apiJson } from "@/lib/api-client";
import type { DebugResponse } from "@/lib/api-types";

export default async function DebugPage() {
  let data: DebugResponse | null = null;
  let errorMessage: string | null = null;

  try {
    data = await apiJson<DebugResponse>("/api/v1/debug");
  } catch (e) {
    errorMessage =
      e instanceof ApiError
        ? e.message
        : e instanceof Error
          ? e.message
          : "Failed to load debug info";
  }

  return (
    <>
      <PageHeader
        title="Pipeline debug"
        description="Latest pipeline run with provider/source diagnostics."
      />
      <div className="p-6">
        {errorMessage ? (
          <div
            className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
            role="alert"
          >
            {errorMessage}
          </div>
        ) : data ? (
          <div className="space-y-4">
            <section className="rounded-lg border border-slate-200 bg-white p-4 text-sm">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium text-slate-900">Run outcome:</span>
                <span
                  className={`inline-flex rounded-full border px-2 py-0.5 text-xs font-semibold ${
                    data.run_outcome === "success"
                      ? "border-emerald-200 bg-emerald-50 text-emerald-900"
                      : data.run_outcome === "partial_success"
                        ? "border-amber-200 bg-amber-50 text-amber-900"
                        : "border-red-200 bg-red-50 text-red-900"
                  }`}
                >
                  {data.run_outcome || "unknown"}
                </span>
              </div>
              <p className="mt-2 text-slate-700">
                {data.summary?.pipeline_text || "No summary yet. Run Generate Leads first."}
              </p>
              {data.run_error_message ? (
                <p className="mt-2 text-red-800">Run error: {data.run_error_message}</p>
              ) : null}
              {data.run ? (
                <p className="mt-2 text-xs text-slate-500">
                  {data.run.run_uuid} · {data.run.status} ·{" "}
                  {data.run.run_outcome || "unknown"}
                </p>
              ) : null}
            </section>

            <section className="rounded-lg border border-slate-200 bg-white p-4 text-sm">
              <h3 className="font-semibold text-slate-900">Website Crawl</h3>
              <div className="mt-2 grid gap-2 sm:grid-cols-2">
                <p>
                  <span className="text-slate-500">Enabled in settings:</span>{" "}
                  {data.website_crawl_enabled ? "yes" : "no"}
                </p>
                <p>
                  <span className="text-slate-500">Firecrawl configured:</span>{" "}
                  {data.website_crawl_configured ? "yes" : "no"}
                </p>
                <p>
                  <span className="text-slate-500">Status:</span>{" "}
                  {data.website_crawl?.status || "—"}
                </p>
                <p>
                  <span className="text-slate-500">Reason code:</span>{" "}
                  {data.website_crawl?.reason_code || "—"}
                </p>
                <p>
                  <span className="text-slate-500">Pages:</span>{" "}
                  {data.website_crawl?.pages_succeeded ?? 0}/
                  {data.website_crawl?.pages_attempted ?? 0}
                </p>
              </div>
              {data.website_crawl?.reason_message ? (
                <p className="mt-2 text-slate-700">{data.website_crawl.reason_message}</p>
              ) : null}
              {data.website_crawl?.skip_reason ? (
                <p className="mt-2 text-amber-800">Skip reason: {data.website_crawl.skip_reason}</p>
              ) : null}
              {data.website_crawl?.error_message ? (
                <p className="mt-1 text-red-800">Error: {data.website_crawl.error_message}</p>
              ) : null}
            </section>

            <section className="rounded-lg border border-slate-200 bg-white p-4 text-sm">
              <h3 className="font-semibold text-slate-900">Website Crawl Funnel (Firecrawl-only, pre-filter to stored)</h3>
              <p className="mt-1 text-xs text-slate-600">
                Raw intake shows Website Crawl / Firecrawl candidate volume before internal filtering.
              </p>
              <div className="mt-2 overflow-x-auto">
                <table className="min-w-[760px] w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500">
                      <th className="py-1 pr-2">Firecrawl raw intake</th>
                      <th className="py-1 pr-2">After domain validation</th>
                      <th className="py-1 pr-2">After dedupe</th>
                      <th className="py-1 pr-2">After ICP filter</th>
                      <th className="py-1 pr-2">After quality gate</th>
                      <th className="py-1">Stored</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="border-b border-slate-100">
                      <td className="py-1 pr-2">{data.website_crawl_funnel?.firecrawl_raw_candidates ?? 0}</td>
                      <td className="py-1 pr-2">{data.website_crawl_funnel?.firecrawl_after_domain_validation ?? 0}</td>
                      <td className="py-1 pr-2">{data.website_crawl_funnel?.firecrawl_after_dedupe ?? 0}</td>
                      <td className="py-1 pr-2">{data.website_crawl_funnel?.firecrawl_after_icp_filter ?? 0}</td>
                      <td className="py-1 pr-2">{data.website_crawl_funnel?.firecrawl_after_quality_gate ?? 0}</td>
                      <td className="py-1">{data.website_crawl_funnel?.firecrawl_stored ?? 0}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div className="mt-3 overflow-x-auto">
                <table className="min-w-[760px] w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500">
                      <th className="py-1 pr-2">Drop-off reason</th>
                      <th className="py-1">Count</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">no valid domain</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_no_valid_domain ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">duplicate</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_duplicate ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">industry mismatch</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_industry_mismatch ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">region mismatch</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_region_mismatch ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">employee range mismatch</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_employee_range_mismatch ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">low confidence</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_low_confidence ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">low signal quality</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_low_signal_quality ?? 0}</td></tr>
                    <tr className="border-b border-slate-100"><td className="py-1 pr-2">quality gate (other)</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_quality_gate ?? 0}</td></tr>
                    <tr><td className="py-1 pr-2">other</td><td className="py-1">{data.website_crawl_funnel?.drop_off_reasons?.dropped_other ?? 0}</td></tr>
                  </tbody>
                </table>
              </div>
            </section>

            <section className="rounded-lg border border-slate-200 bg-white p-4 text-sm">
              <h3 className="font-semibold text-slate-900">Provider execution</h3>
              <div className="mt-2 overflow-x-auto">
                <table className="min-w-[980px] w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500">
                      <th className="py-1 pr-2">Source</th>
                      <th className="py-1 pr-2">Provider</th>
                      <th className="py-1 pr-2">Status</th>
                      <th className="py-1 pr-2">Reason code</th>
                      <th className="py-1 pr-2">Cfg</th>
                      <th className="py-1 pr-2">Enabled</th>
                      <th className="py-1 pr-2">Pages</th>
                      <th className="py-1 pr-2">Candidates</th>
                      <th className="py-1 pr-2">Budget</th>
                      <th className="py-1">Reason / Error</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(data.provider_details || []).map((p) => (
                      <tr key={p.source_key} className="border-b border-slate-100 align-top">
                        <td className="py-1 pr-2 font-mono">{p.source_key}</td>
                        <td className="py-1 pr-2">{p.provider_name}</td>
                        <td className="py-1 pr-2">{p.status}</td>
                        <td className="py-1 pr-2">{p.reason_code || "—"}</td>
                        <td className="py-1 pr-2">{p.configured ? "yes" : "no"}</td>
                        <td className="py-1 pr-2">{p.enabled_by_settings ? "yes" : "no"}</td>
                        <td className="py-1 pr-2">{p.pages_succeeded}/{p.pages_attempted}</td>
                        <td className="py-1 pr-2">
                          ok {p.candidates_success} · skip {p.candidates_skipped} · fail {p.candidates_failed}
                        </td>
                        <td className="py-1 pr-2">
                          {p.budget_used_sec}/{p.budget_limit_sec}s · rows skipped {p.budget_rows_skipped}
                        </td>
                        <td className="py-1">
                          {p.reason_message || p.skip_reason || p.error_message || "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

            <section className="rounded-lg border border-slate-200 bg-white p-4 text-sm">
              <h3 className="font-semibold text-slate-900">Source breakdown</h3>
              <div className="mt-2 overflow-x-auto">
                <table className="min-w-[760px] w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500">
                      <th className="py-1 pr-2">Source</th>
                      <th className="py-1 pr-2">Status</th>
                      <th className="py-1 pr-2">Generated</th>
                      <th className="py-1 pr-2">Stored</th>
                      <th className="py-1 pr-2">Skipped</th>
                      <th className="py-1 pr-2">Failed</th>
                      <th className="py-1 pr-2">Kept</th>
                      <th className="py-1 pr-2">Qualified</th>
                      <th className="py-1 pr-2">Conversion</th>
                      <th className="py-1">Reason / Error</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(data.breakdown_rows || []).map((r) => (
                      <tr key={r.source_name} className="border-b border-slate-100">
                        <td className="py-1 pr-2 font-mono">{r.source_name}</td>
                        <td className="py-1 pr-2">{r.status}</td>
                        <td className="py-1 pr-2">{r.generated}</td>
                        <td className="py-1 pr-2">{r.stored}</td>
                        <td className="py-1 pr-2">{r.skipped}</td>
                        <td className="py-1 pr-2">{r.failed}</td>
                        <td className="py-1 pr-2">{r.kept}</td>
                        <td className="py-1 pr-2">{r.qualified}</td>
                        <td className="py-1 pr-2">{r.conversion}</td>
                        <td className="py-1">{r.skip_reason || r.last_error || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </div>
        ) : null}
      </div>
    </>
  );
}
