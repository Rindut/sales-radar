/**
 * Central HTTP client for the Sales Radar Go API.
 *
 * - If `NEXT_PUBLIC_API_BASE_URL` is set, requests go there directly.
 * - If unset, use same-origin paths `/api/v1/...` (see `next.config.ts` rewrites → Go).
 */

import type { ApiErrorBody } from "./api-types";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly code?: string,
    public readonly body?: ApiErrorBody
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function getApiBaseUrl(): string {
  const raw = process.env.NEXT_PUBLIC_API_BASE_URL?.trim() ?? "";
  return raw.replace(/\/$/, "");
}

/**
 * Browser-only: when `NEXT_PUBLIC_API_BASE_URL` is unset, use same-origin paths
 * (rewritten by Next.js to the Go API).
 */
export function clientApiUrl(path: string): string {
  const p = path.startsWith("/") ? path : `/${path}`;
  const base = getApiBaseUrl();
  if (base) return `${base}${p}`;
  return p;
}

async function resolveUrl(path: string): Promise<string> {
  const p = path.startsWith("/") ? path : `/${path}`;
  const base = getApiBaseUrl();
  if (base) {
    return `${base}${p}`;
  }
  if (typeof window !== "undefined") {
    return p;
  }
  try {
    const { headers } = await import("next/headers");
    const h = await headers();
    const host = h.get("x-forwarded-host") ?? h.get("host");
    const proto = h.get("x-forwarded-proto") ?? "http";
    if (host) {
      return `${proto}://${host}${p}`;
    }
  } catch {
    /* headers() unavailable outside a request */
  }
  const fallback =
    process.env.API_UPSTREAM?.trim() || "http://127.0.0.1:8080";
  return `${fallback.replace(/\/$/, "")}${p}`;
}

export async function apiFetch(
  path: string,
  init?: RequestInit
): Promise<Response> {
  const url = await resolveUrl(path);
  const headers = new Headers(init?.headers);
  if (!headers.has("Accept") && !path.endsWith(".csv")) {
    headers.set("Accept", "application/json");
  }
  if (
    init?.body != null &&
    typeof init.body === "string" &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(url, {
    ...init,
    headers,
    cache: "no-store",
  });
}

export async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await apiFetch(path, init);
  const text = await res.text();
  if (!res.ok) {
    let body: ApiErrorBody | undefined;
    try {
      body = text ? (JSON.parse(text) as ApiErrorBody) : undefined;
    } catch {
      body = undefined;
    }
    const msg =
      body?.error?.message ??
      text?.slice(0, 200) ??
      res.statusText;
    const code = body?.error?.code;
    throw new ApiError(res.status, msg, code, body);
  }
  if (!text) {
    return {} as T;
  }
  return JSON.parse(text) as T;
}
