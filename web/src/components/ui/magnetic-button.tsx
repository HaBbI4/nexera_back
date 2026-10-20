"use client";

/**
 * Inspired by 21st.dev Magnetic Button (id 26725, mohammadshehadeh / hirael).
 * Adapted to Signal Forge tokens; no Radix Slot dependency.
 */
import * as React from "react";
import {
  motion,
  useReducedMotion,
  useSpring,
  type HTMLMotionProps,
} from "motion/react";
import { cn } from "@/lib/utils";

const SPRING = { stiffness: 200, damping: 15, mass: 0.1 };

type Props = HTMLMotionProps<"button"> & {
  strength?: number;
  variant?: "primary" | "ghost" | "porcelain";
};

export function MagneticButton({
  className,
  strength = 0.35,
  variant = "primary",
  children,
  ...props
}: Props) {
  const reduced = useReducedMotion();
  const x = useSpring(0, SPRING);
  const y = useSpring(0, SPRING);

  const onPointerMove = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (reduced) return;
    const rect = event.currentTarget.getBoundingClientRect();
    x.set((event.clientX - (rect.left + rect.width / 2)) * strength);
    y.set((event.clientY - (rect.top + rect.height / 2)) * strength);
  };

  const reset = () => {
    x.set(0);
    y.set(0);
  };

  return (
    <motion.button
      type="button"
      onPointerMove={onPointerMove}
      onPointerLeave={reset}
      style={{ x, y }}
      className={cn(
        "inline-flex items-center justify-center gap-2 px-6 py-3 text-sm font-medium tracking-wide transition-colors disabled:opacity-60",
        variant === "primary" &&
          "bg-[var(--accent)] text-[var(--ink)] hover:bg-[var(--accent-deep)]",
        variant === "ghost" &&
          "border border-[var(--line)] bg-transparent text-[var(--porcelain)] hover:border-[var(--accent)] hover:text-[var(--accent)]",
        variant === "porcelain" &&
          "bg-[var(--porcelain)] text-[var(--ink)] hover:bg-white",
        className,
      )}
      {...props}
    >
      {children}
    </motion.button>
  );
}

export function MagneticLink({
  href,
  className,
  strength = 0.35,
  variant = "primary",
  children,
}: {
  href: string;
  className?: string;
  strength?: number;
  variant?: "primary" | "ghost" | "porcelain";
  children: React.ReactNode;
}) {
  const reduced = useReducedMotion();
  const x = useSpring(0, SPRING);
  const y = useSpring(0, SPRING);

  const onPointerMove = (event: React.PointerEvent<HTMLAnchorElement>) => {
    if (reduced) return;
    const rect = event.currentTarget.getBoundingClientRect();
    x.set((event.clientX - (rect.left + rect.width / 2)) * strength);
    y.set((event.clientY - (rect.top + rect.height / 2)) * strength);
  };

  return (
    <motion.a
      href={href}
      onPointerMove={onPointerMove}
      onPointerLeave={() => {
        x.set(0);
        y.set(0);
      }}
      style={{ x, y }}
      className={cn(
        "inline-flex items-center justify-center gap-2 px-6 py-3 text-sm font-medium tracking-wide transition-colors",
        variant === "primary" &&
          "bg-[var(--accent)] text-[var(--ink)] hover:bg-[var(--accent-deep)]",
        variant === "ghost" &&
          "border border-[var(--line)] bg-transparent text-[var(--porcelain)] hover:border-[var(--accent)] hover:text-[var(--accent)]",
        variant === "porcelain" &&
          "bg-[var(--porcelain)] text-[var(--ink)] hover:bg-white",
        className,
      )}
    >
      {children}
    </motion.a>
  );
}
