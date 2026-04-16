"use client";

import { NavLink } from "./NavLink";

const nav = [
  { href: "/leads", label: "Lead list" },
  { href: "/settings", label: "Settings" },
  { href: "/debug", label: "Debug" },
] as const;

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen">
      <aside className="flex w-64 shrink-0 flex-col border-r border-slate-200 bg-white">
        <div className="border-b border-slate-200 px-4 py-4">
          <div className="text-sm font-semibold tracking-tight text-slate-900">
            Sales Radar
          </div>
          <div className="text-xs text-slate-500">BAWANA · Sales</div>
        </div>
        <nav className="flex flex-col gap-0.5 p-3" aria-label="Main">
          {nav.map((item) => (
            <NavLink key={item.href} href={item.href}>
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">{children}</div>
    </div>
  );
}
