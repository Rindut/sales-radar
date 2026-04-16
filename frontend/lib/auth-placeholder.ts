/**
 * Future auth hook for Sales Radar (SSO, API keys, or session cookies).
 *
 * - Add middleware in middleware.ts to protect routes or attach tokens.
 * - Extend apiFetch in api-client.ts to add Authorization headers from here.
 * - Do not commit secrets; use env vars on Vercel + the Go API host.
 */
export function getAuthHeaders(): Record<string, string> {
  return {};
}
