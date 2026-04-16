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
        description="Last run metadata and integration status (stub — full view in Phase 4)."
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
          <div className="max-w-2xl space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm">
            <p>
              <span className="text-slate-500">Runs in DB:</span>{" "}
              {data.no_runs_in_db ? (
                <span className="text-amber-700">none yet</span>
              ) : (
                <span className="text-emerald-700">yes</span>
              )}
            </p>
            {data.run ? (
              <p>
                <span className="text-slate-500">Last run:</span>{" "}
                <span className="font-mono text-xs">{data.run.run_uuid}</span> ·{" "}
                {data.run.status}
              </p>
            ) : null}
            {data.summary?.pipeline_text ? (
              <p className="text-slate-700">{data.summary.pipeline_text}</p>
            ) : (
              <p className="text-slate-500">
                Run “Generate leads” from the list page once the API is connected to
                persist debug JSON.
              </p>
            )}
          </div>
        ) : null}
      </div>
    </>
  );
}
