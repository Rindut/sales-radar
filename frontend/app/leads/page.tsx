import Link from "next/link";
import { Suspense } from "react";

import { ExportCsvButton } from "@/components/leads/ExportCsvButton";
import { GenerateLeadsButton } from "@/components/leads/GenerateLeadsButton";
import { LeadFiltersForm } from "@/components/leads/LeadFiltersForm";
import { LeadKpiStrip } from "@/components/leads/LeadKpiStrip";
import { LeadListTable } from "@/components/leads/LeadListTable";
import { ApiError, apiJson } from "@/lib/api-client";
import type { LeadsListResponse } from "@/lib/api-types";
import { resolveSearchParams, toQueryString } from "@/lib/search-params";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

export default async function LeadsPage({ searchParams }: Props) {
  const sp = await resolveSearchParams(searchParams);
  const qs = toQueryString(sp);
  const path = qs ? `/api/v1/leads?${qs}` : "/api/v1/leads";

  let data: LeadsListResponse | null = null;
  let errorMessage: string | null = null;

  try {
    data = await apiJson<LeadsListResponse>(path);
  } catch (e) {
    errorMessage =
      e instanceof ApiError
        ? e.message
        : e instanceof Error
          ? e.message
          : "Failed to load leads";
  }

  const lastRun = data?.summary?.last_run;
  const totalShown = data?.pagination?.returned ?? data?.items?.length ?? 0;
  const totalInDB = data?.meta?.total_in_db ?? 0;
  const pipelineHasRun = data?.meta?.pipeline_has_run ?? false;
  const industries = data?.meta?.industries ?? [];

  const showNoDataHint = !lastRun && !pipelineHasRun;
  const showEmptyFilterHint =
    !lastRun && pipelineHasRun && totalInDB === 0 && totalShown === 0;

  const debugHref = qs ? `/debug?${qs}` : "/debug";

  return (
    <>
      <header className="border-b border-slate-200 bg-white px-6 py-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="text-lg font-semibold text-slate-900">Lead list</h1>
            <p className="mt-1 text-sm text-slate-600">
              Leads prioritized by ICP fit.
            </p>
          </div>
          <Suspense
            fallback={
              <div className="h-10 w-44 animate-pulse rounded-lg bg-slate-200" />
            }
          >
            <GenerateLeadsButton />
          </Suspense>
        </div>
      </header>

      <div className="space-y-6 p-6">
        {errorMessage ? (
          <div
            className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
            role="alert"
          >
            {errorMessage}
          </div>
        ) : data ? (
          <>
            <LeadKpiStrip
              lastRun={lastRun}
              showNoDataHint={showNoDataHint}
              showEmptyFilterHint={showEmptyFilterHint}
            />

            <LeadFiltersForm searchParams={sp} industries={industries} />

            <section aria-labelledby="leads-heading">
              <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <h2 id="leads-heading" className="text-base font-semibold text-slate-900">
                  Leads
                </h2>
                <div className="flex flex-wrap items-center gap-3">
                  <Suspense
                    fallback={
                      <div className="h-9 w-28 animate-pulse rounded bg-slate-200" />
                    }
                  >
                    <ExportCsvButton enabled={totalShown > 0} />
                  </Suspense>
                  <span className="text-sm text-slate-600">
                    <strong className="text-slate-900">{totalShown}</strong> lead(s)
                    shown
                  </span>
                  <Link
                    href={debugHref}
                    className="text-xs font-medium text-primary hover:underline"
                  >
                    [Debug]
                  </Link>
                </div>
              </div>

              {totalShown === 0 ? (
                <p className="rounded-lg border border-dashed border-slate-200 bg-white px-4 py-8 text-center text-sm text-slate-500">
                  {showNoDataHint || showEmptyFilterHint
                    ? "Adjust filters or generate leads to see results here."
                    : "No leads match the current filters."}
                </p>
              ) : (
                <LeadListTable items={data.items} />
              )}
            </section>
          </>
        ) : null}
      </div>
    </>
  );
}
