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
