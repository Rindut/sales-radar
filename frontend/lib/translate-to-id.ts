/**
 * Client-side EN → ID translation for the lead drawer (temporary).
 * Uses the public MyMemory API; text is sent to a third-party service.
 */

import type { Lead } from "@/lib/api-types";
import { sourceLabel } from "@/lib/lead-display";
import { summaryForDrawer, whyFitForDrawer } from "@/lib/sales-summary";

const MAX_CHUNK = 420;
const BETWEEN_MS = 90;

const cache = new Map<string, string>();

function delay(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

function chunkText(s: string, maxLen: number): string[] {
  const t = s.trim();
  if (!t) return [];
  if (t.length <= maxLen) return [t];
  const chunks: string[] = [];
  let rest = t;
  while (rest.length > maxLen) {
    let cut = rest.lastIndexOf(" ", maxLen);
    if (cut < maxLen * 0.4) cut = maxLen;
    chunks.push(rest.slice(0, cut).trim());
    rest = rest.slice(cut).trim();
  }
  if (rest) chunks.push(rest);
  return chunks;
}

async function fetchMyMemoryChunk(q: string): Promise<string> {
  const url = `https://api.mymemory.translated.net/get?q=${encodeURIComponent(q)}&langpair=en|id`;
  const res = await fetch(url);
  if (!res.ok) throw new Error("translate_http");
  const data = (await res.json()) as {
    responseData?: { translatedText?: string };
    responseStatus?: number;
  };
  const out = data.responseData?.translatedText;
  if (typeof out !== "string" || !out.trim()) throw new Error("translate_empty");
  return out;
}

/**
 * Translates English prose to Indonesian. Caches identical strings in-memory.
 */
export async function translateToID(text: string): Promise<string> {
  const raw = (text ?? "").trim();
  if (!raw) return "";
  const cached = cache.get(raw);
  if (cached !== undefined) return cached;

  const chunks = chunkText(raw, MAX_CHUNK);
  const parts: string[] = [];
  for (let i = 0; i < chunks.length; i++) {
    const c = chunks[i];
    const subCached = cache.get(c);
    if (subCached !== undefined) {
      parts.push(subCached);
    } else {
      const t = await fetchMyMemoryChunk(c);
      cache.set(c, t);
      parts.push(t);
    }
    if (i < chunks.length - 1) await delay(BETWEEN_MS);
  }
  const joined = parts.join("\n\n");
  cache.set(raw, joined);
  return joined;
}

export type LeadDrawerIdBundle = {
  summary: string;
  reason_for_fit: string;
  why_now: string;
  sales_angle: string;
  reasons: string[];
  accept_explanation: string;
  source_label: string;
  source_trace_line: string;
  source_ref_line: string;
  record_status_line: string;
  confidence: string;
  country_region: string;
  industry: string;
  employee_size: string;
};

/**
 * Translates all user-visible prose for the drawer when language is ID.
 * Runs sequentially to reduce rate limits on the free translation API.
 */
export async function translateLeadFieldsToId(lead: Lead): Promise<LeadDrawerIdBundle> {
  const reasonsIn = lead.reasons ?? [];
  const traceRaw = (lead.source_trace ?? []).join(" · ");
  const recordRaw = [lead.sales_status, lead.lead_status, lead.duplicate_status]
    .filter((x) => (x ?? "").trim())
    .join(" · ");

  const summary = await translateToID(summaryForDrawer(lead));
  await delay(BETWEEN_MS);
  const reason_for_fit = await translateToID(whyFitForDrawer(lead));
  await delay(BETWEEN_MS);
  const why_now = await translateToID(lead.why_now ?? "");
  await delay(BETWEEN_MS);
  const sales_angle = await translateToID(lead.sales_angle ?? "");
  await delay(BETWEEN_MS);
  const accept_explanation = await translateToID(lead.accept_explanation ?? "");
  await delay(BETWEEN_MS);
  const source_label = await translateToID(sourceLabel(lead));
  await delay(BETWEEN_MS);
  const source_trace_line = await translateToID(traceRaw);
  await delay(BETWEEN_MS);
  const source_ref_line = await translateToID(lead.source_ref ?? "");
  await delay(BETWEEN_MS);
  const record_status_line = await translateToID(recordRaw);
  await delay(BETWEEN_MS);
  const confidence = await translateToID(lead.confidence ?? "");
  await delay(BETWEEN_MS);
  const country_region = await translateToID(lead.country_region ?? "");
  await delay(BETWEEN_MS);
  const industry = await translateToID(lead.industry ?? "");
  await delay(BETWEEN_MS);
  const employee_size = await translateToID((lead.employee_size || lead.size) ?? "");

  const reasons: string[] = [];
  for (let i = 0; i < reasonsIn.length; i++) {
    await delay(BETWEEN_MS);
    reasons.push(await translateToID(reasonsIn[i]));
  }

  return {
    summary,
    reason_for_fit,
    why_now,
    sales_angle,
    reasons,
    accept_explanation,
    source_label,
    source_trace_line,
    source_ref_line,
    record_status_line,
    confidence,
    country_region,
    industry,
    employee_size,
  };
}
