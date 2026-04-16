"use client";

import { useSearchParams } from "next/navigation";
import { useState } from "react";

import { getApiBaseUrl } from "@/lib/api-client";

export function ExportCsvButton({ enabled }: { enabled: boolean }) {
  const searchParams = useSearchParams();
  const [loading, setLoading] = useState(false);

  async function onClick() {
    if (!enabled || loading) return;
    setLoading(true);
    try {
      const base = getApiBaseUrl();
      const q = searchParams.toString();
      const url = `${base}/api/v1/export.csv${q ? `?${q}` : ""}`;
      const res = await fetch(url, { method: "GET", credentials: "omit" });
      if (!res.ok) {
        const t = await res.text();
        throw new Error(t.slice(0, 200) || res.statusText);
      }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition");
      let name = "leads_export.csv";
      const m = cd?.match(/filename="([^"]+)"/);
      if (m) name = m[1];
      const href = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = href;
      a.download = name;
      a.rel = "noopener";
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(href);
    } catch (e) {
      window.alert(e instanceof Error ? e.message : "Export failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!enabled || loading}
      aria-busy={loading}
      title={!enabled ? "No data to export" : undefined}
      className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
    >
      {loading ? (
        <svg
          className="h-4 w-4 animate-spin text-slate-600"
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
        <span aria-hidden>⬇</span>
      )}
      Export CSV
    </button>
  );
}
