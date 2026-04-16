import Link from "next/link";

import { PageHeader } from "@/components/PageHeader";
import { ApiError, apiJson } from "@/lib/api-client";
import type { LeadDetailResponse } from "@/lib/api-types";

type Props = { params: Promise<{ id: string }> };

export default async function LeadDetailPage({ params }: Props) {
  const { id } = await params;
  let data: LeadDetailResponse | null = null;
  let errorMessage: string | null = null;

  try {
    data = await apiJson<LeadDetailResponse>(`/api/v1/leads/${encodeURIComponent(id)}`);
  } catch (e) {
    errorMessage =
      e instanceof ApiError
        ? e.message
        : e instanceof Error
          ? e.message
          : "Failed to load lead";
  }

  const lead = data?.lead;

  return (
    <>
      <PageHeader
        title={lead?.company ? String(lead.company) : `Lead #${id}`}
        description="Detail view (stub — full layout in Phase 4)."
      />
      <div className="p-6">
        <p className="mb-4 text-sm">
          <Link href="/leads" className="text-primary hover:underline">
            ← Back to list
          </Link>
        </p>
        {errorMessage ? (
          <div
            className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800"
            role="alert"
          >
            {errorMessage}
          </div>
        ) : lead ? (
          <dl className="max-w-xl space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm">
            <div>
              <dt className="text-slate-500">ICP match</dt>
              <dd className="font-medium">{String(lead.icp_match ?? "—")}</dd>
            </div>
            <div>
              <dt className="text-slate-500">Summary</dt>
              <dd className="text-slate-800">{String(lead.summary ?? "—")}</dd>
            </div>
            <div>
              <dt className="text-slate-500">Official domain</dt>
              <dd className="font-mono text-xs">
                {lead.official_domain ? String(lead.official_domain) : "—"}
              </dd>
            </div>
          </dl>
        ) : null}
      </div>
    </>
  );
}
