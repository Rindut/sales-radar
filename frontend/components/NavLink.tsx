"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const active =
  "rounded-lg bg-primary-soft text-sm font-medium text-primary shadow-sm";
const idle =
  "rounded-lg text-sm font-medium text-slate-700 hover:bg-slate-100 hover:text-slate-900";

export function NavLink({
  href,
  label,
  icon,
  collapsed,
}: {
  href: string;
  label: string;
  icon?: React.ReactNode;
  collapsed?: boolean;
}) {
  const pathname = usePathname();
  const isActive =
    pathname === href || (href !== "/leads" && pathname.startsWith(href + "/"));
  const cls = `${isActive ? active : idle} flex items-center ${
    collapsed ? "justify-center px-2 py-2" : "gap-2.5 px-3 py-2"
  }`;

  return (
    <Link
      href={href}
      className={cls}
      title={collapsed ? label : undefined}
      aria-label={label}
    >
      {icon ? (
        <span className="flex shrink-0 items-center justify-center [&_svg]:h-5 [&_svg]:w-5">
          {icon}
        </span>
      ) : null}
      {collapsed ? (
        <span className="sr-only">{label}</span>
      ) : (
        <span>{label}</span>
      )}
    </Link>
  );
}
