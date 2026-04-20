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
    /** Backstop so the button never spins forever if the proxy/API hangs. Slightly above server handler max (45m). */
    const clientTTLms = 50 * 60 * 1000;
    const controller = new AbortController();
    const ttl = window.setTimeout(() => controller.abort(), clientTTLms);
    try {
      const res = await fetch(clientApiUrl("/api/v1/pipeline/run"), {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: "{}",
        signal: controller.signal,
      });
      const text = await res.text();
      if (!res.ok) {
        let msg = text.slice(0, 400) || res.statusText;
        try {
          const j = JSON.parse(text) as {
            error?: { message?: string; code?: string };
          };
          if (j?.error?.message) msg = j.error.message;
        } catch {
          /* keep msg */
        }
        const generic =
          msg === "Internal Server Error" ||
          /^[\s]*Internal Server Error[\s]*$/i.test(msg);
        throw new Error(
          generic
            ? `HTTP ${res.status}: request to the Go API failed or timed out (Next dev proxy defaults to 30s; \`experimental.proxyTimeout\` in next.config is set higher—restart dev after pull).`
            : res.status === 504
              ? `${msg} (gateway timeout — pipeline hit SALESRADAR_PIPELINE_HANDLER_TIMEOUT_SEC or upstream limit)`
              : `${msg} (HTTP ${res.status})`
        );
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
      const aborted =
        e instanceof Error &&
        (e.name === "AbortError" || /aborted/i.test(e.message));
      window.alert(
        aborted
          ? `Timed out after ${Math.floor(clientTTLms / 60000)} minutes — aborting the request. Check that the Go API is running and see SALESRADAR_* timeout env vars.`
          : e instanceof Error
            ? e.message
            : "Pipeline failed"
      );
    } finally {
      window.clearTimeout(ttl);
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
