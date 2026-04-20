"use client";

import { useRouter } from "next/navigation";
import { usePathname, useSearchParams } from "next/navigation";
import { useState } from "react";

import { clientApiUrl } from "@/lib/api-client";
import type { PipelineRunAPIResponse } from "@/lib/api-types";
import { mergePipelineRunIntoPath } from "@/lib/pipeline-url";

export function GenerateLeadsButton() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [loading, setLoading] = useState(false);

  async function onClick() {
    setLoading(true);
    try {
      const res = await fetch(clientApiUrl("/api/v1/pipeline/run"), {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: "{}",
      });
      const text = await res.text();
      if (!res.ok) {
        let msg = text.slice(0, 400) || res.statusText;
        try {
          const j = JSON.parse(text) as { error?: { message?: string } };
          if (j?.error?.message) msg = j.error.message;
        } catch {
          /* keep msg */
        }
        throw new Error(msg);
      }
      const data = JSON.parse(text) as PipelineRunAPIResponse;
      const next = mergePipelineRunIntoPath(
        pathname || "/leads",
        searchParams.toString(),
        data.stats,
        data.rows_persisted
      );
      router.push(next);
      router.refresh();
    } catch (e) {
      window.alert(e instanceof Error ? e.message : "Pipeline failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        onClick={onClick}
        disabled={loading}
        aria-busy={loading}
        className="inline-flex items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-primary-hover disabled:opacity-60"
      >
        {loading ? (
          <svg
            className="h-4 w-4 animate-spin"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            aria-hidden
          >
            <circle
              cx="12"
              cy="12"
              r="10"
              strokeWidth="3"
              strokeDasharray="48"
              strokeDashoffset="12"
              strokeLinecap="round"
            />
          </svg>
        ) : (
          <svg
            className="h-4 w-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden
          >
            <path
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M13 10V3L4 14h7v7l9-11h-7z"
            />
          </svg>
        )}
        {loading ? "Generating…" : "Generate leads"}
      </button>
      <p className="text-xs text-slate-500">Discover new leads from multiple sources</p>
    </div>
  );
}
