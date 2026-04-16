"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const active =
  "rounded-lg bg-primary-soft px-3 py-2 text-sm font-medium text-primary";
const idle =
  "rounded-lg px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 hover:text-slate-900";

export function NavLink({
  href,
  children,
}: {
  href: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const isActive =
    pathname === href || (href !== "/leads" && pathname.startsWith(href + "/"));
  return (
    <Link href={href} className={isActive ? active : idle}>
      {children}
    </Link>
  );
}
