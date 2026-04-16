import { PageHeader } from "@/components/PageHeader";
import { ApiError, apiJson } from "@/lib/api-client";
import type { SettingsResponse } from "@/lib/api-types";

export default async function SettingsPage() {
  let data: SettingsResponse | null = null;
  let errorMessage: string | null = null;

  try {
    data = await apiJson<SettingsResponse>("/api/v1/settings");
  } catch (e) {
    errorMessage =
      e instanceof ApiError
        ? e.message
        : e instanceof Error
          ? e.message
          : "Failed to load settings";
  }

  return (
    <>
      <PageHeader
        title="Settings"
        description="Discovery sources and ICP configuration (read-only stub — editing in Phase 4)."
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
          <div className="max-w-2xl space-y-4 text-sm">
            <section className="rounded-lg border border-slate-200 bg-white p-4">
              <h2 className="font-semibold text-slate-900">Discovery sources</h2>
              <ul className="mt-2 grid grid-cols-2 gap-2 text-slate-700">
                {Object.entries(data.discovery_sources).map(([k, v]) => (
                  <li key={k}>
                    <span className="text-slate-500">{k}:</span>{" "}
                    {v ? (
                      <span className="text-emerald-700">on</span>
                    ) : (
                      <span className="text-slate-400">off</span>
                    )}
                  </li>
                ))}
              </ul>
            </section>
            <section className="rounded-lg border border-slate-200 bg-white p-4">
              <h2 className="font-semibold text-slate-900">Catalogs</h2>
              <p className="mt-1 text-slate-600">
                {data.catalogs.industries.length} industries ·{" "}
                {data.catalogs.weights.length} weight options
              </p>
            </section>
          </div>
        ) : null}
      </div>
    </>
  );
}
