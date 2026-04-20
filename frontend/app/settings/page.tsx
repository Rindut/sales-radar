import { PageHeader } from "@/components/PageHeader";
import { SettingsForm } from "@/components/settings/SettingsForm";
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
        description="Discovery sources and ICP configuration. Changes apply the next time you run Generate Leads."
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
          <SettingsForm initial={data} />
        ) : null}
      </div>
    </>
  );
}
