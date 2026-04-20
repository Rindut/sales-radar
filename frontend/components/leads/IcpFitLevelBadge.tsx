"use client";

import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from "react";

const TOOLTIP_TEXT = `Confidence Levels:

High → Strong ICP match, ready for outreach
Medium → Partial match, needs validation
Low → Weak match, low priority`;

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
          className="pointer-events-none absolute left-1/2 top-full z-[60] mt-1.5 max-w-[250px] -translate-x-1/2 whitespace-pre-line rounded-md bg-slate-800 px-3 py-2 text-left text-[11px] leading-snug text-white shadow-lg"
        >
          {TOOLTIP_TEXT}
        </div>
      ) : null}
    </div>
  );
}
