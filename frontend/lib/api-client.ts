/**
 * Central HTTP client for the Sales Radar Go API.
 * Use NEXT_PUBLIC_API_BASE_URL (e.g. http://127.0.0.1:8080 for local cmd/api).
 *
 * Auth: attach headers here when middleware/JWT is added (see README).
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

function joinUrl(path: string): string {
  const base = getApiBaseUrl();
  if (!base) {
    throw new ApiError(
      500,
      "NEXT_PUBLIC_API_BASE_URL is not set. Copy .env.example to .env.local."
    );
  }
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${base}${p}`;
}

export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const url = joinUrl(path);
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
