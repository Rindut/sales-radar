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
};

export default nextConfig;
