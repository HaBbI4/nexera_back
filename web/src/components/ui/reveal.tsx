"use client";

import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function Reveal({
  children,
  className,
  delay = 0,
}: {
  children: ReactNode;
  className?: string;
  delay?: number;
}) {
  const reduced = useReducedMotion();
  if (reduced) {
    return <div className={className}>{children}</div>;
  }
  return (
    <motion.div
      className={className}
      initial={{ opacity: 0, y: 28 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-10% 0px" }}
      transition={{ duration: 0.65, delay, ease: [0.23, 1, 0.32, 1] }}
    >
      {children}
    </motion.div>
  );
}

export function KineticHeadline({
  lines,
  className,
}: {
  lines: string[];
  className?: string;
}) {
  const reduced = useReducedMotion();
  return (
    <h1 className={cn("font-display font-semibold leading-[0.95]", className)}>
      {lines.map((line, i) => (
        <span key={line} className="block overflow-hidden pb-1">
          <motion.span
            className="block"
            initial={reduced ? false : { y: "110%" }}
            animate={{ y: 0 }}
            transition={{
              duration: reduced ? 0 : 0.85,
              delay: reduced ? 0 : 0.15 + i * 0.12,
              ease: [0.23, 1, 0.32, 1],
            }}
          >
            {line}
          </motion.span>
        </span>
      ))}
    </h1>
  );
}
