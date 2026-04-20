import path from "node:path";
import { fileURLToPath } from "node:url";

import type { NextConfig } from "next";

/** Lock the app root to this directory so Next does not infer the monorepo parent when multiple lockfiles exist. */
const frontendRoot = path.dirname(fileURLToPath(import.meta.url));

const nextConfig: NextConfig = {
  reactStrictMode: true,
  /** Standalone app in `frontend/` — avoid inferring parent repo when another lockfile exists. */
  outputFileTracingRoot: frontendRoot,
  turbopack: {
    root: frontendRoot,
  },
  /**
   * `rewrites` are proxied with a default **30s** timeout in dev; longer requests get
   * `500` and body `Internal Server Error` (see `next/dist/.../proxy-request.js`).
   * Generate Leads runs discovery + website crawl and often exceeds 30s.
   */
  experimental: {
    // Must exceed Go `SALESRADAR_PIPELINE_HANDLER_TIMEOUT_SEC` (default 45m) or the proxy returns 500 before the API finishes.
    proxyTimeout: 3_000_000, // 50 minutes (ms); client fetch abort is 50m as well
  },
  /**
   * Proxy `/api/v1/*` to the Go API so the browser and SSR can use same-origin URLs
   * when `NEXT_PUBLIC_API_BASE_URL` is unset (avoids CORS and connection issues).
   */
  async rewrites() {
    const upstream =
      process.env.API_UPSTREAM?.trim() || "http://127.0.0.1:8080";
    const base = upstream.replace(/\/$/, "");
    return [
      {
        source: "/api/v1/:path*",
        destination: `${base}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
