/**
 * Drawer-only EN/ID strings (parity with cmd/web list.html DRAWER_UI).
 * Preference key matches legacy: salesradar.drawerLang
 */

import type { Lead } from "@/lib/api-types";
import { actionLabel, priorityBand, readinessKind } from "@/lib/lead-display";

export type DrawerLang = "en" | "id";

export const DRAWER_LANG_STORAGE_KEY = "salesradar.drawerLang";

export function getDrawerLang(): DrawerLang {
  if (typeof window === "undefined") return "en";
  try {
    const v = window.localStorage.getItem(DRAWER_LANG_STORAGE_KEY);
    if (v === "id" || v === "en") return v;
  } catch {
    /* ignore */
  }
  return "en";
}

export function setDrawerLang(lang: DrawerLang): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(DRAWER_LANG_STORAGE_KEY, lang);
  } catch {
    /* ignore */
  }
}

const UI = {
  en: {
    langGroupAria: "Drawer content language",
    closeAria: "Close panel",
    icpTitle: "ICP fit",
    suggestedTitle: "Suggested action",
    summaryTitle: "Summary",
    whyFitTitle: "Why this lead fits",
    whyNowTitle: "Why now",
    howApproachTitle: "How to approach",
    companyTitle: "Company",
    priorityPrefix: "ICP Priority:",
    readinessPrefix: "Readiness:",
    tierHigh: "High",
    tierMed: "Medium",
    tierLow: "Low",
    openProfile: "Open profile",
    dtWebsite: "Website",
    dtLinkedIn: "LinkedIn",
    dtRegion: "Region",
    dtIndustry: "Industry",
    dtSize: "Size",
    moreDetails: "More details",
    dtSource: "Source",
    dtTrace: "Source path",
    dtSref: "Source reference",
    dtIntegrations: "Integrations",
    dtAdded: "Added",
    dtCompleteness: "Data completeness",
    dtConfidence: "Confidence",
    dtRecord: "Sales / lead status",
    dtInternal: "Internal note",
  },
  id: {
    langGroupAria: "Bahasa konten panel",
    closeAria: "Tutup panel",
    icpTitle: "Kecocokan ICP",
    suggestedTitle: "Saran tindakan",
    summaryTitle: "Ringkasan",
    whyFitTitle: "Mengapa lead ini cocok",
    whyNowTitle: "Mengapa sekarang",
    howApproachTitle: "Cara pendekatan",
    companyTitle: "Perusahaan",
    priorityPrefix: "Prioritas ICP:",
    readinessPrefix: "Kesiapan:",
    tierHigh: "Tinggi",
    tierMed: "Sedang",
    tierLow: "Rendah",
    openProfile: "Buka profil",
    dtWebsite: "Situs web",
    dtLinkedIn: "LinkedIn",
    dtRegion: "Wilayah",
    dtIndustry: "Industri",
    dtSize: "Ukuran",
    moreDetails: "Selengkapnya",
    dtSource: "Sumber",
    dtTrace: "Jalur sumber",
    dtSref: "Referensi sumber",
    dtIntegrations: "Integrasi",
    dtAdded: "Ditambahkan",
    dtCompleteness: "Kelengkapan data",
    dtConfidence: "Keyakinan",
    dtRecord: "Status sales / lead",
    dtInternal: "Catatan internal",
  },
} as const;

export function drawerStrings(lang: DrawerLang) {
  return UI[lang];
}

function actionLabelId(lead: Lead): string {
  const a = (lead.action || "").trim();
  if (a === "Contact") return "Siap dihubungi";
  if (a === "Ignore") return "Lewati untuk saat ini";
  return "Riset terlebih dahulu";
}

export function drawerActionLabel(lead: Lead, lang: DrawerLang): string {
  if (lang === "en") return actionLabel(lead);
  return actionLabelId(lead);
}

function readinessLabelEn(lead: Lead): string {
  const k = readinessKind(lead);
  if (k === "ready") return "Ready";
  if (k === "almost") return "Almost ready";
  return "Not ready";
}

function readinessLabelId(lead: Lead): string {
  const k = readinessKind(lead);
  if (k === "ready") return "Siap";
  if (k === "almost") return "Hampir siap";
  return "Belum siap";
}

export function drawerReadinessLabel(lead: Lead, lang: DrawerLang): string {
  return lang === "en" ? readinessLabelEn(lead) : readinessLabelId(lead);
}

function priorityLabelFromBand(lang: DrawerLang, band: "high" | "medium" | "low"): string {
  const u = UI[lang];
  if (band === "high") return u.tierHigh;
  if (band === "medium") return u.tierMed;
  return u.tierLow;
}

export function drawerPriorityLabel(lead: Lead, lang: DrawerLang): string {
  return priorityLabelFromBand(lang, priorityBand(lead));
}

/** Display ICP tier chip; translates common English tier tokens when lang is id. */
export function drawerIcpMatchDisplay(lead: Lead, lang: DrawerLang): string {
  const raw = (lead.icp_match ?? "").trim();
  if (!raw) return "—";
  if (lang === "en") return raw;
  const lower = raw.toLowerCase();
  if (lower === "high" || lower === "yes") return UI.id.tierHigh;
  if (lower === "medium") return UI.id.tierMed;
  if (lower === "low" || lower === "no") return UI.id.tierLow;
  return raw;
}
