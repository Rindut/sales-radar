"use client";

import { useEffect, useMemo, useState } from "react";

import type { Lead } from "@/lib/api-types";
import {
  type DrawerLang,
  drawerActionLabel,
  drawerIcpMatchDisplay,
  drawerPriorityLabel,
  drawerReadinessLabel,
  drawerStrings,
  getDrawerLang,
  setDrawerLang,
} from "@/lib/drawer-i18n";
import { displayDomain, priorityBand, sourceLabel } from "@/lib/lead-display";
import { IcpFitLevelBadge } from "@/components/leads/IcpFitLevelBadge";
import { summaryForDrawer, whyFitForDrawer } from "@/lib/sales-summary";
import {
  type LeadDrawerIdBundle,
  translateLeadFieldsToId,
} from "@/lib/translate-to-id";

function dash(s: string | undefined | null): string {
  const t = (s ?? "").trim();
  return t ? t : "—";
}

function leadSignature(lead: Lead): string {
  return [
    lead.id,
    lead.summary,
    lead.reason_for_fit,
    lead.why_now,
    lead.sales_angle,
    lead.accept_explanation,
    (lead.reasons ?? []).join("\u0001"),
    lead.sales_status,
    lead.lead_status,
    lead.duplicate_status,
    (lead.source_trace ?? []).join("|"),
    lead.source_ref,
    lead.country_region,
    lead.industry,
    lead.employee_size,
    lead.size,
    lead.confidence,
    lead.website_enrichment_status,
    lead.website_enrichment_summary,
    lead.website_enrichment_signals,
    lead.website_enriched_at,
  ].join("\u0002");
}

export function LeadDetailDrawer({
  lead,
  onClose,
}: {
  lead: Lead | null;
  onClose: () => void;
}) {
  const open = lead != null;
  const [entered, setEntered] = useState(false);
  const [lang, setLang] = useState<DrawerLang>("en");
  const [idBundle, setIdBundle] = useState<LeadDrawerIdBundle | null>(null);
  const [translating, setTranslating] = useState(false);
  const [translateErr, setTranslateErr] = useState<string | null>(null);

  const sig = useMemo(() => (lead ? leadSignature(lead) : ""), [lead]);

  useEffect(() => {
    setLang(getDrawerLang());
  }, []);

  useEffect(() => {
    if (!open) return;
    const id = requestAnimationFrame(() => setEntered(true));
    return () => {
      cancelAnimationFrame(id);
      setEntered(false);
    };
  }, [open, lead?.id]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prev;
    };
  }, [open]);

  useEffect(() => {
    if (!lead) {
      setIdBundle(null);
      setTranslating(false);
      setTranslateErr(null);
      return;
    }
    if (lang !== "id") {
      setIdBundle(null);
      setTranslating(false);
      setTranslateErr(null);
      return;
    }
    let cancelled = false;
    setTranslating(true);
    setIdBundle(null);
    setTranslateErr(null);
    translateLeadFieldsToId(lead)
      .then((b) => {
        if (!cancelled) {
          setIdBundle(b);
          setTranslating(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setTranslateErr(
            "Terjemahan gagal. Periksa koneksi atau coba lagi nanti."
          );
          setTranslating(false);
        }
      });
    return () => {
      cancelled = true;
    };
    // `sig` fingerprints lead content; `lead` is intentionally omitted to avoid
    // re-fetching when the parent passes a new object reference for the same row.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- see above
  }, [lang, sig]);

  if (!lead) return null;

  const ui = drawerStrings(lang);
  const showIdProse = lang === "id" && idBundle && !translating && !translateErr;

  const icpBand = priorityBand(lead);
  const icpLevelCls =
    icpBand === "high"
      ? "border-red-200 bg-red-50 text-red-900"
      : icpBand === "medium"
        ? "border-amber-200 bg-amber-50 text-amber-950"
        : "border-slate-200 bg-slate-100 text-slate-700";

  const integrations: string[] = [];
  if (lead.used_google) integrations.push("Google");
  if (lead.used_apollo) integrations.push("Apollo");
  if (lead.used_linkedin) integrations.push("LinkedIn");

  function selectLang(next: DrawerLang) {
    setDrawerLang(next);
    setLang(next);
  }

  const summaryText =
    lang === "en"
      ? dash(summaryForDrawer(lead))
      : showIdProse
        ? dash(idBundle!.summary)
        : "";
  const reasonFitText =
    lang === "en"
      ? dash(whyFitForDrawer(lead))
      : showIdProse
        ? dash(idBundle!.reason_for_fit)
        : "";
  const whyNowText =
    lang === "en" ? dash(lead.why_now) : showIdProse ? dash(idBundle!.why_now) : "";
  const salesAngleText =
    lang === "en"
      ? dash(lead.sales_angle)
      : showIdProse
        ? dash(idBundle!.sales_angle)
        : "";

  const hasDetailSections = !!(
    whyFitForDrawer(lead) ||
    lead.why_now ||
    lead.sales_angle
  );

  const hasWebsiteEnrichment = !!(
    (lead.website_enrichment_status || "").trim() ||
    (lead.website_enrichment_summary || "").trim() ||
    (lead.website_enrichment_signals || "").trim()
  );

  return (
    <>
      <button
        type="button"
        aria-hidden
        tabIndex={-1}
        className={`fixed inset-0 z-40 bg-slate-900/40 transition-opacity duration-300 ${
          entered ? "opacity-100" : "opacity-0"
        }`}
        onClick={onClose}
      />

      <aside
        className={`fixed inset-y-0 right-0 z-50 flex w-full max-w-lg flex-col border-l border-slate-200 bg-white shadow-xl transition-transform duration-300 ease-out sm:max-w-xl ${
          entered ? "translate-x-0" : "translate-x-full"
        }`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="lead-drawer-title"
      >
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-slate-200 px-4 py-4">
          <div className="min-w-0 flex-1">
            <h2
              id="lead-drawer-title"
              className="break-words text-lg font-semibold text-slate-900"
            >
              {dash(lead.company)}
            </h2>
            <p className="mt-0.5 break-words font-mono text-sm text-slate-500">
              {displayDomain(lead)}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <div
              className="flex items-center gap-0.5 rounded-lg border border-slate-200 bg-slate-50 p-0.5"
              role="group"
              aria-label={ui.langGroupAria}
            >
              <button
                type="button"
                onClick={() => selectLang("id")}
                className={`rounded px-2 py-1 text-xs font-semibold ${
                  lang === "id"
                    ? "bg-white text-slate-900 shadow-sm"
                    : "text-slate-600 hover:text-slate-900"
                }`}
                aria-pressed={lang === "id"}
              >
                ID
              </button>
              <span className="text-slate-300" aria-hidden>
                |
              </span>
              <button
                type="button"
                onClick={() => selectLang("en")}
                className={`rounded px-2 py-1 text-xs font-semibold ${
                  lang === "en"
                    ? "bg-white text-slate-900 shadow-sm"
                    : "text-slate-600 hover:text-slate-900"
                }`}
                aria-pressed={lang === "en"}
              >
                EN
              </button>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-slate-200 px-2.5 py-1 text-lg leading-none text-slate-600 hover:bg-slate-100"
              aria-label={ui.closeAria}
            >
              ×
            </button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
          {lang === "id" && translating && (
            <p className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950">
              Menerjemahkan konten ke Bahasa Indonesia…
            </p>
          )}
          {lang === "id" && translateErr && (
            <p className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-900">
              {translateErr}
            </p>
          )}

          <section className="relative z-10 mb-4 overflow-visible rounded-lg border border-primary/20 bg-primary-soft/80 p-4">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-600">
              {ui.icpTitle}
            </h3>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <IcpFitLevelBadge className={icpLevelCls}>
                {drawerIcpMatchDisplay(lead, lang)}
              </IcpFitLevelBadge>
              <span className="text-sm text-slate-600">
                {ui.priorityPrefix}{" "}
                <strong>{drawerPriorityLabel(lead, lang)}</strong> ({lead.priority_score})
              </span>
            </div>
            {lead.reasons &&
            lead.reasons.length > 0 &&
            (lang === "en" || showIdProse) ? (
              <ul className="mt-3 list-disc space-y-1 pl-5 text-sm leading-relaxed text-slate-700">
                {lead.reasons.map((r, i) => (
                  <li key={`${lead.id}-r-${i}`} className="break-words">
                    {lang === "en" ? r : idBundle!.reasons[i] ?? ""}
                  </li>
                ))}
              </ul>
            ) : null}
          </section>

          <section className="mb-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-600">
              {ui.suggestedTitle}
            </h3>
            <p className="mt-2 text-sm font-medium text-slate-900">
              {drawerActionLabel(lead, lang)}
            </p>
            <p className="mt-1 text-xs text-slate-600">
              {ui.readinessPrefix} {drawerReadinessLabel(lead, lang)}
            </p>
          </section>

          <section className="mb-4">
            <h3 className="text-sm font-semibold text-slate-900">{ui.summaryTitle}</h3>
            <p className="mt-1 break-words whitespace-pre-wrap text-sm leading-relaxed text-slate-700">
              {lang === "id" && !showIdProse ? null : summaryText}
            </p>
          </section>

          {hasWebsiteEnrichment && (
            <section className="mb-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
              <h3 className="text-sm font-semibold text-slate-900">
                Website enrichment
              </h3>
              <p className="mt-1 text-xs text-slate-500">
                {(lead.website_enrichment_status || "").trim() || "—"}
                {lead.website_enriched_at
                  ? ` · ${lead.website_enriched_at}`
                  : ""}
              </p>
              {(lead.website_enrichment_signals || "").trim() ? (
                <p className="mt-2 break-words text-sm font-medium text-slate-900">
                  {lead.website_enrichment_signals}
                </p>
              ) : null}
              {(lead.website_enrichment_summary || "").trim() ? (
                <p className="mt-2 break-words whitespace-pre-wrap text-sm leading-relaxed text-slate-700">
                  {lead.website_enrichment_summary}
                </p>
              ) : null}
            </section>
          )}

          {hasDetailSections && (
            <div className="mb-4 space-y-4">
              {whyFitForDrawer(lead) ? (
                <section>
                  <h3 className="text-sm font-semibold text-slate-900">{ui.whyFitTitle}</h3>
                  <p className="mt-1 break-words whitespace-pre-wrap text-sm leading-relaxed text-slate-700">
                    {lang === "id" && !showIdProse ? null : reasonFitText}
                  </p>
                </section>
              ) : null}
              {lead.why_now ? (
                <section>
                  <h3 className="text-sm font-semibold text-slate-900">{ui.whyNowTitle}</h3>
                  <p className="mt-1 break-words whitespace-pre-wrap text-sm leading-relaxed text-slate-700">
                    {lang === "id" && !showIdProse ? null : whyNowText}
                  </p>
                </section>
              ) : null}
              {lead.sales_angle ? (
                <section>
                  <h3 className="text-sm font-semibold text-slate-900">{ui.howApproachTitle}</h3>
                  <p className="mt-1 break-words whitespace-pre-wrap text-sm leading-relaxed text-slate-700">
                    {lang === "id" && !showIdProse ? null : salesAngleText}
                  </p>
                </section>
              ) : null}
            </div>
          )}

          <section className="mb-4">
            <h3 className="text-sm font-semibold text-slate-900">{ui.companyTitle}</h3>
            <dl className="mt-2 space-y-2 text-sm">
              <div className="grid grid-cols-[8rem_1fr] gap-1">
                <dt className="text-slate-500">{ui.dtWebsite}</dt>
                <dd className="break-words font-mono text-xs text-slate-800">
                  {dash(lead.official_domain || lead.website_domain)}
                </dd>
              </div>
              <div className="grid grid-cols-[8rem_1fr] gap-1">
                <dt className="text-slate-500">{ui.dtLinkedIn}</dt>
                <dd className="break-words text-slate-800">
                  {lead.linkedin_url ? (
                    <a
                      href={lead.linkedin_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline"
                    >
                      {ui.openProfile}
                    </a>
                  ) : (
                    "—"
                  )}
                </dd>
              </div>
              <div className="grid grid-cols-[8rem_1fr] gap-1">
                <dt className="text-slate-500">{ui.dtRegion}</dt>
                <dd className="break-words">
                  {lang === "en"
                    ? dash(lead.country_region)
                    : showIdProse
                      ? dash(idBundle!.country_region)
                      : null}
                </dd>
              </div>
              <div className="grid grid-cols-[8rem_1fr] gap-1">
                <dt className="text-slate-500">{ui.dtIndustry}</dt>
                <dd className="break-words">
                  {lang === "en"
                    ? dash(lead.industry)
                    : showIdProse
                      ? dash(idBundle!.industry)
                      : null}
                </dd>
              </div>
              <div className="grid grid-cols-[8rem_1fr] gap-1">
                <dt className="text-slate-500">{ui.dtSize}</dt>
                <dd className="break-words">
                  {lang === "en"
                    ? dash(lead.employee_size || lead.size)
                    : showIdProse
                      ? dash(idBundle!.employee_size)
                      : null}
                </dd>
              </div>
            </dl>
          </section>

          <details className="mb-4 rounded-lg border border-slate-200 bg-white">
            <summary className="cursor-pointer px-3 py-2 text-sm font-medium text-slate-800">
              {ui.moreDetails}
            </summary>
            <div className="border-t border-slate-100 px-3 py-3 text-sm">
              <dl className="space-y-2">
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtSource}</dt>
                  <dd>
                    {lang === "en"
                      ? sourceLabel(lead)
                      : showIdProse
                        ? idBundle!.source_label
                        : null}
                  </dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtTrace}</dt>
                  <dd className="break-all font-mono text-xs">
                    {lang === "en"
                      ? (lead.source_trace || []).length > 0
                        ? lead.source_trace!.join(" · ")
                        : "—"
                      : showIdProse
                        ? dash(idBundle!.source_trace_line)
                        : null}
                  </dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtSref}</dt>
                  <dd className="break-all font-mono text-xs">
                    {lang === "en"
                      ? dash(lead.source_ref)
                      : showIdProse
                        ? dash(idBundle!.source_ref_line)
                        : null}
                  </dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtIntegrations}</dt>
                  <dd>{integrations.length ? integrations.join(", ") : "—"}</dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtAdded}</dt>
                  <dd className="text-xs">{dash(lead.created_at)}</dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtCompleteness}</dt>
                  <dd>
                    {lead.data_completeness != null ? `${lead.data_completeness} / 100` : "—"}
                  </dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtConfidence}</dt>
                  <dd>
                    {lang === "en"
                      ? dash(lead.confidence)
                      : showIdProse
                        ? dash(idBundle!.confidence)
                        : null}
                  </dd>
                </div>
                <div className="grid grid-cols-[8rem_1fr] gap-1">
                  <dt className="text-slate-500">{ui.dtRecord}</dt>
                  <dd className="break-words text-xs">
                    {lang === "en"
                      ? `${dash(lead.sales_status)} · ${dash(lead.lead_status)}${
                          lead.duplicate_status ? ` · ${lead.duplicate_status}` : ""
                        }`
                      : showIdProse
                        ? dash(idBundle!.record_status_line)
                        : null}
                  </dd>
                </div>
                {lead.accept_explanation ? (
                  <div className="grid grid-cols-[8rem_1fr] gap-1">
                    <dt className="text-slate-500">{ui.dtInternal}</dt>
                    <dd className="break-words text-slate-700">
                      {lang === "en"
                        ? lead.accept_explanation
                        : showIdProse
                          ? idBundle!.accept_explanation
                          : null}
                    </dd>
                  </div>
                ) : null}
              </dl>
            </div>
          </details>
        </div>
      </aside>
    </>
  );
}
