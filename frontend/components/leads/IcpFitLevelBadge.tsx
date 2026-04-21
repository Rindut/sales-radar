"use client";

import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from "react";

const TOOLTIP_ROWS = [
  {
    label: "High",
    description: "Strong ICP match, worth immediate attention",
    labelClassName: "text-emerald-700",
  },
  {
    label: "Medium",
    description: "Partial fit, needs review",
    labelClassName: "text-amber-600",
  },
  {
    label: "Low",
    description: "Weak fit, low priority",
    labelClassName: "text-slate-500",
  },
];

/**
 * ICP tier badge with a tooltip explaining all confidence levels (hover on desktop, tap on touch).
 */
export function IcpFitLevelBadge({
  className,
  children,
}: {
  className: string;
  children: ReactNode;
}) {
  const tipId = useId();
  const wrapRef = useRef<HTMLDivElement>(null);
  const [hoverOpen, setHoverOpen] = useState(false);
  const [tapOpen, setTapOpen] = useState(false);
  const [prefersHover, setPrefersHover] = useState(true);

  useEffect(() => {
    const mq = window.matchMedia("(hover: hover)");
    setPrefersHover(mq.matches);
    const fn = () => setPrefersHover(mq.matches);
    mq.addEventListener("change", fn);
    return () => mq.removeEventListener("change", fn);
  }, []);

  const open = prefersHover ? hoverOpen : tapOpen;

  useEffect(() => {
    if (prefersHover || !tapOpen) return;
    const close = (e: PointerEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setTapOpen(false);
      }
    };
    document.addEventListener("pointerdown", close, true);
    return () => document.removeEventListener("pointerdown", close, true);
  }, [prefersHover, tapOpen]);

  const onBadgeClick = useCallback(
    (e: React.MouseEvent) => {
      if (!prefersHover) {
        e.preventDefault();
        setTapOpen((v) => !v);
      }
    },
    [prefersHover]
  );

  return (
    <div
      ref={wrapRef}
      className="relative inline-flex max-w-full"
      onMouseEnter={() => prefersHover && setHoverOpen(true)}
      onMouseLeave={() => prefersHover && setHoverOpen(false)}
    >
      <button
        type="button"
        className={`inline-flex max-w-full items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold capitalize ${className} ${
          prefersHover ? "cursor-default" : "cursor-pointer"
        } focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1`}
        aria-describedby={open ? tipId : undefined}
        aria-label="ICP confidence level. Shows definitions for all levels."
        onClick={onBadgeClick}
      >
        {children}
      </button>
      {open ? (
        <div
          id={tipId}
          role="tooltip"
          className="pointer-events-none absolute left-1/2 top-full z-[60] mt-2 min-w-[260px] max-w-[320px] -translate-x-1/2 rounded-[8px] border border-[#e5e7eb] bg-white px-[14px] py-[12px] text-left text-[13px] leading-[1.6] text-[#374151] shadow-[0_4px_12px_rgba(0,0,0,0.08)]"
        >
          <div className="space-y-[6px]">
            <div className="text-[13px] font-medium text-[#374151]">
              How to read this:
            </div>
            {TOOLTIP_ROWS.map((row) => (
              <div key={row.label} className="grid grid-cols-[70px_minmax(0,1fr)] gap-3">
                <div className={`font-semibold ${row.labelClassName}`}>
                  {row.label}
                </div>
                <div className="text-[#374151]">{row.description}</div>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}
