/** Helpers for Next.js `searchParams` (App Router). */

export function spGet(
  sp: Record<string, string | string[] | undefined> | null | undefined,
  key: string
): string {
  if (sp == null || typeof sp !== "object") return "";
  const v = sp[key];
  if (Array.isArray(v)) return v[0] ?? "";
  return typeof v === "string" ? v : "";
}

export function toQueryString(
  sp: Record<string, string | string[] | undefined> | null | undefined
): string {
  if (sp == null || typeof sp !== "object" || Array.isArray(sp)) {
    return "";
  }
  const u = new URLSearchParams();
  for (const [k, v] of Object.entries(sp)) {
    if (v === undefined) continue;
    if (Array.isArray(v)) {
      v.forEach((x) => u.append(k, x));
    } else {
      u.set(k, v);
    }
  }
  return u.toString();
}

/**
 * Next.js 15 passes `searchParams` as a Promise; it may resolve to `undefined`
 * in edge cases. Never pass that through to `Object.entries` (would throw → 500).
 */
export async function resolveSearchParams(
  searchParams:
    | Promise<Record<string, string | string[] | undefined>>
    | Record<string, string | string[] | undefined>
    | undefined
): Promise<Record<string, string | string[] | undefined>> {
  if (searchParams == null) return {};
  const raw = await Promise.resolve(searchParams);
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) {
    return {};
  }
  return raw;
}
