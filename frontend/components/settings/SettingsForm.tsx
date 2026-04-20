"use client";

import { useMemo, useState, type FormEvent } from "react";

import { ApiError, apiJson } from "@/lib/api-client";
import type {
  DiscoveryIntegrationRow,
  DiscoverySourcesToggles,
  ICPForm,
  PutSettingsRequest,
  SettingsResponse,
} from "@/lib/api-types";

const DISCOVERY_ROWS: {
  key: keyof DiscoverySourcesToggles;
  name: string;
  description: string;
}[] = [
  {
    key: "google",
    name: "Google",
    description: "Web search to surface candidate companies.",
  },
  {
    key: "seed",
    name: "Seed Discovery",
    description: "Built-in list of distinct companies.",
  },
  {
    key: "website_crawl",
    name: "Website Crawl",
    description:
      "Pulls extra context from company websites after a domain is known.",
  },
  {
    key: "job_signal",
    name: "Job Signal",
    description:
      "Surfaces hiring and training-related signals from job-style clues.",
  },
  {
    key: "apollo",
    name: "Apollo",
    description: "Company enrichment and firmographics by domain.",
  },
  {
    key: "linkedin",
    name: "LinkedIn",
    description: "Adds company LinkedIn URLs when enrichment returns them.",
  },
];

const MIN_EMPLOYEES = ["", "50", "100", "200", "500", "1000"] as const;
const MAX_EMPLOYEES = ["", "500", "1000", "5000"] as const;

function sub50On(icp: ICPForm): boolean {
  if (icp.apply_sub50_rule === null || icp.apply_sub50_rule === undefined) {
    return true;
  }
  return icp.apply_sub50_rule;
}

function signalSelectionFromICP(
  icp: ICPForm,
  allSignalIds: string[]
): Set<string> {
  const keys = icp.signal_keys;
  if (!keys?.length) {
    return new Set(allSignalIds);
  }
  return new Set(keys);
}

function buildPayload(
  discovery: DiscoverySourcesToggles,
  icpDraft: {
    industries: Set<string>;
    region: string;
    signals: Set<string>;
    allSignalIds: string[];
    applySub50: boolean;
    weightIndustry: string;
    weightSignal: string;
    weightSize: string;
    minEmployees: string;
    maxEmployees: string;
    version?: number;
  }
): PutSettingsRequest {
  const allS = icpDraft.allSignalIds;
  const sigEqualAll =
    icpDraft.signals.size === allS.length &&
    allS.every((id) => icpDraft.signals.has(id));

  const icp: ICPForm = {
    _v: icpDraft.version,
    target_industries: Array.from(icpDraft.industries),
    region_focus: icpDraft.region,
    signal_keys: sigEqualAll ? [] : Array.from(icpDraft.signals),
    apply_sub50_rule: icpDraft.applySub50,
    weight_industry: icpDraft.weightIndustry,
    weight_signal: icpDraft.weightSignal,
    weight_size: icpDraft.weightSize,
    min_employees: icpDraft.minEmployees,
    max_employees: icpDraft.maxEmployees,
  };

  return { discovery_sources: discovery, icp };
}

function formatWeightLabel(w: string): string {
  const t = w.trim().toLowerCase();
  if (!t) return w;
  return t.charAt(0).toUpperCase() + t.slice(1);
}

function DiscoveryIntegrationStatus({
  info,
  sourceOn,
}: {
  info: DiscoveryIntegrationRow | undefined;
  sourceOn: boolean;
}) {
  if (!info) {
    return (
      <p className="mt-1 text-xs text-slate-500">Integration status unavailable.</p>
    );
  }
  if (!info.requires_integration) {
    return (
      <p className="mt-1 text-xs text-slate-500">No integration required</p>
    );
  }
  const ok = info.configured === true;
  return (
    <>
      <p
        className={`mt-1 text-xs ${ok ? "text-emerald-800" : "text-amber-800"}`}
      >
        {ok
          ? "Integration required · Configured"
          : "Integration required · Not configured"}
      </p>
      {info.provider_name ? (
        <p className="mt-0.5 text-xs text-slate-600">
          Provider: {info.provider_name}
        </p>
      ) : null}
      {info.hint ? (
        <p className="mt-0.5 text-xs leading-snug text-slate-500">{info.hint}</p>
      ) : null}
      {sourceOn && !ok && !info.hint ? (
        <p className="mt-0.5 text-xs leading-snug text-amber-800/95">
          This source is on but will not run until integration is configured.
        </p>
      ) : null}
    </>
  );
}

export function SettingsForm({ initial }: { initial: SettingsResponse }) {
  const allSignalIds = useMemo(
    () => initial.catalogs.signals.map((s) => s.id),
    [initial.catalogs.signals]
  );

  const [discoveryIntegrations, setDiscoveryIntegrations] = useState<
    DiscoveryIntegrationRow[]
  >(() => initial.discovery_integrations ?? []);

  const discoveryIntegrationByKey = useMemo(() => {
    const m = new Map<string, DiscoveryIntegrationRow>();
    for (const row of discoveryIntegrations) {
      m.set(row.key, row);
    }
    return m;
  }, [discoveryIntegrations]);

  const [discovery, setDiscovery] = useState<DiscoverySourcesToggles>(() => ({
    ...initial.discovery_sources,
  }));
  const [industries, setIndustries] = useState<Set<string>>(() => {
    const ids = new Set(initial.catalogs.industries.map((i) => i.id));
    const from = initial.icp.target_industries ?? [];
    const next = new Set<string>();
    for (const id of from) {
      if (ids.has(id)) next.add(id);
    }
    if (next.size === 0) {
      for (const id of ids) next.add(id);
    }
    return next;
  });
  const [region, setRegion] = useState(
    () => initial.icp.region_focus ?? ""
  );
  const [signals, setSignals] = useState<Set<string>>(() =>
    signalSelectionFromICP(initial.icp, allSignalIds)
  );
  const [applySub50, setApplySub50] = useState(() => sub50On(initial.icp));
  const [weightIndustry, setWeightIndustry] = useState(
    () => initial.icp.weight_industry || "medium"
  );
  const [weightSignal, setWeightSignal] = useState(
    () => initial.icp.weight_signal || "medium"
  );
  const [weightSize, setWeightSize] = useState(
    () => initial.icp.weight_size || "medium"
  );
  const [minEmployees, setMinEmployees] = useState(
    () => initial.icp.min_employees ?? ""
  );
  const [maxEmployees, setMaxEmployees] = useState(
    () => initial.icp.max_employees ?? ""
  );

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  function toggleIndustry(id: string) {
    setIndustries((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleSignal(id: string) {
    setSignals((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleDiscovery(key: keyof DiscoverySourcesToggles) {
    setDiscovery((d) => ({ ...d, [key]: !d[key] }));
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(false);
    setSaving(true);
    const body = buildPayload(discovery, {
      industries,
      region,
      signals,
      allSignalIds,
      applySub50,
      weightIndustry,
      weightSignal,
      weightSize,
      minEmployees,
      maxEmployees,
      version: initial.icp._v,
    });
    try {
      const updated = await apiJson<SettingsResponse>("/api/v1/settings", {
        method: "PUT",
        body: JSON.stringify(body),
      });
      setDiscovery({ ...updated.discovery_sources });
      const ids = new Set(updated.catalogs.industries.map((i) => i.id));
      const ti = updated.icp.target_industries ?? [];
      const nextInd = new Set<string>();
      for (const id of ti) {
        if (ids.has(id)) nextInd.add(id);
      }
      if (nextInd.size === 0) {
        for (const id of ids) nextInd.add(id);
      }
      setIndustries(nextInd);
      setRegion(updated.icp.region_focus ?? "");
      setSignals(signalSelectionFromICP(updated.icp, allSignalIds));
      setApplySub50(sub50On(updated.icp));
      setWeightIndustry(updated.icp.weight_industry || "medium");
      setWeightSignal(updated.icp.weight_signal || "medium");
      setWeightSize(updated.icp.weight_size || "medium");
      setMinEmployees(updated.icp.min_employees ?? "");
      setMaxEmployees(updated.icp.max_employees ?? "");
      setDiscoveryIntegrations(updated.discovery_integrations ?? []);
      setSuccess(true);
    } catch (err) {
      setSuccess(false);
      setError(
        err instanceof ApiError
          ? err.message
          : err instanceof Error
            ? err.message
            : "Save failed"
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="max-w-3xl space-y-6 text-sm">
      {error ? (
        <div
          className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-red-800"
          role="alert"
        >
          {error}
        </div>
      ) : null}
      {success ? (
        <div
          className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-emerald-900"
          role="status"
        >
          Settings saved.
        </div>
      ) : null}

      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
        <h2 className="text-base font-semibold text-slate-900">
          Discovery sources
        </h2>
        <p className="mt-1 text-slate-600">
          Turn sources on or off for the next pipeline run.
        </p>
        <ul className="mt-4 space-y-3">
          {DISCOVERY_ROWS.map((row) => (
            <li
              key={row.key}
              className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 pb-3 last:border-0 last:pb-0"
            >
              <div className="min-w-0 flex-1">
                <div className="font-medium text-slate-800">{row.name}</div>
                <p className="mt-0.5 text-slate-600">{row.description}</p>
                <DiscoveryIntegrationStatus
                  info={discoveryIntegrationByKey.get(row.key)}
                  sourceOn={discovery[row.key]}
                />
              </div>
              <label className="flex shrink-0 cursor-pointer items-center gap-2">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-primary focus:ring-primary"
                  checked={discovery[row.key]}
                  onChange={() => toggleDiscovery(row.key)}
                />
                <span className="text-slate-700">
                  {discovery[row.key] ? "On" : "Off"}
                </span>
              </label>
            </li>
          ))}
        </ul>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
        <h2 className="text-base font-semibold text-slate-900">ICP settings</h2>
        <p className="mt-1 text-slate-600">
          Who you want to pursue. Applied the next time you run{" "}
          <strong>Generate Leads</strong>.
        </p>

        <div className="mt-6 space-y-8">
          <div>
            <h3 className="font-medium text-slate-800">Industry target</h3>
            <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {initial.catalogs.industries.map((ind) => (
                <label
                  key={ind.id}
                  className="flex cursor-pointer items-start gap-2 rounded-md border border-slate-100 px-2 py-1.5 hover:bg-slate-50"
                >
                  <input
                    type="checkbox"
                    className="mt-0.5 h-4 w-4 rounded border-slate-300 text-primary focus:ring-primary"
                    checked={industries.has(ind.id)}
                    onChange={() => toggleIndustry(ind.id)}
                  />
                  <span className="text-slate-700">{ind.label}</span>
                </label>
              ))}
            </div>
          </div>

          <div>
            <h3 className="font-medium text-slate-800">Company size</h3>
            <div className="mt-3 grid gap-4 sm:grid-cols-2">
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-slate-600">
                  Minimum employees
                </span>
                <select
                  value={minEmployees}
                  onChange={(e) => setMinEmployees(e.target.value)}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-primary focus:border-primary focus:ring-1"
                >
                  <option value="">—</option>
                  {MIN_EMPLOYEES.filter((x) => x !== "").map((v) => (
                    <option key={v} value={v}>
                      {v}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-slate-600">
                  Maximum employees
                </span>
                <select
                  value={maxEmployees}
                  onChange={(e) => setMaxEmployees(e.target.value)}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-primary focus:border-primary focus:ring-1"
                >
                  <option value="">No limit</option>
                  {MAX_EMPLOYEES.filter((x) => x !== "").map((v) => (
                    <option key={v} value={v}>
                      {v}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <label className="mt-4 flex cursor-pointer items-start gap-2">
              <input
                type="checkbox"
                className="mt-0.5 h-4 w-4 rounded border-slate-300 text-primary focus:ring-primary"
                checked={applySub50}
                onChange={(e) => setApplySub50(e.target.checked)}
              />
              <span className="text-slate-700">
                Exclude companies with under 50 employees when size is clear in
                the data
              </span>
            </label>
          </div>

          <div>
            <h3 className="font-medium text-slate-800">Region</h3>
            <label className="mt-3 block max-w-xs">
              <span className="mb-1 block text-xs font-medium text-slate-600">
                Market focus
              </span>
              <select
                value={region}
                onChange={(e) => setRegion(e.target.value)}
                className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-primary focus:border-primary focus:ring-1"
              >
                {initial.catalogs.regions.map((r) => (
                  <option key={r.id || "global"} value={r.id}>
                    {r.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <div>
            <h3 className="font-medium text-slate-800">Buying signals</h3>
            <p className="mt-1 text-slate-600">
              What we look for in company text. Uncheck to ignore a theme.
            </p>
            <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {initial.catalogs.signals.map((sig) => (
                <label
                  key={sig.id}
                  className="flex cursor-pointer items-start gap-2 rounded-md border border-slate-100 px-2 py-1.5 hover:bg-slate-50"
                >
                  <input
                    type="checkbox"
                    className="mt-0.5 h-4 w-4 rounded border-slate-300 text-primary focus:ring-primary"
                    checked={signals.has(sig.id)}
                    onChange={() => toggleSignal(sig.id)}
                  />
                  <span className="text-slate-700">{sig.label}</span>
                </label>
              ))}
            </div>
          </div>

          <div>
            <h3 className="font-medium text-slate-800">Priority</h3>
            <p className="mt-1 text-slate-600">
              How much each factor raises or lowers the priority score (0–100).
            </p>
            <div className="mt-3 grid gap-4 sm:grid-cols-3">
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-slate-600">
                  Industry fit
                </span>
                <select
                  value={weightIndustry}
                  onChange={(e) => setWeightIndustry(e.target.value)}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm capitalize outline-none ring-primary focus:border-primary focus:ring-1"
                >
                  {initial.catalogs.weights.map((w) => (
                    <option key={w} value={w}>
                      {formatWeightLabel(w)}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-slate-600">
                  Signals
                </span>
                <select
                  value={weightSignal}
                  onChange={(e) => setWeightSignal(e.target.value)}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-primary focus:border-primary focus:ring-1"
                >
                  {initial.catalogs.weights.map((w) => (
                    <option key={w} value={w}>
                      {formatWeightLabel(w)}
                    </option>
                  ))}
                </select>
              </label>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-slate-600">
                  Company size
                </span>
                <select
                  value={weightSize}
                  onChange={(e) => setWeightSize(e.target.value)}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm outline-none ring-primary focus:border-primary focus:ring-1"
                >
                  {initial.catalogs.weights.map((w) => (
                    <option key={w} value={w}>
                      {formatWeightLabel(w)}
                    </option>
                  ))}
                </select>
              </label>
            </div>
          </div>
        </div>
      </section>

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="submit"
          disabled={saving}
          className="inline-flex items-center justify-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-primary/90 disabled:opacity-60"
        >
          {saving ? "Saving…" : "Save changes"}
        </button>
      </div>
    </form>
  );
}
