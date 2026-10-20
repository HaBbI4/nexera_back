"use client";

/**
 * Inspired by 21st.dev Accordion (id 23530, ddoemonn) — spring height + chevron.
 * Simplified for FAQ use with Signal Forge styling.
 */
import { useId, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { cn } from "@/lib/utils";

export type AccordionItem = {
  id: string;
  question: string;
  answer: string;
};

export function Accordion({
  items,
  className,
}: {
  items: AccordionItem[];
  className?: string;
}) {
  const [openId, setOpenId] = useState<string | null>(items[0]?.id ?? null);
  const baseId = useId();
  const reduced = useReducedMotion();

  return (
    <div className={cn("divide-y divide-[var(--line)] border-y border-[var(--line)]", className)}>
      {items.map((item) => {
        const open = openId === item.id;
        const panelId = `${baseId}-${item.id}-panel`;
        const buttonId = `${baseId}-${item.id}-button`;
        return (
          <div key={item.id}>
            <h3>
              <button
                id={buttonId}
                type="button"
                aria-expanded={open}
                aria-controls={panelId}
                className="flex w-full items-start justify-between gap-6 py-5 text-left text-lg font-medium tracking-tight transition-colors hover:text-[var(--accent)] md:text-xl"
                onClick={() => setOpenId(open ? null : item.id)}
              >
                <span>{item.question}</span>
                <motion.span
                  aria-hidden
                  className="mt-1 inline-block text-[var(--accent)]"
                  animate={{ rotate: open ? 45 : 0 }}
                  transition={
                    reduced
                      ? { duration: 0 }
                      : { type: "spring", stiffness: 700, damping: 46 }
                  }
                >
                  +
                </motion.span>
              </button>
            </h3>
            <AnimatePresence initial={false}>
              {open ? (
                <motion.div
                  id={panelId}
                  role="region"
                  aria-labelledby={buttonId}
                  initial={reduced ? false : { height: 0, opacity: 0 }}
                  animate={{ height: "auto", opacity: 1 }}
                  exit={reduced ? undefined : { height: 0, opacity: 0 }}
                  transition={
                    reduced
                      ? { duration: 0 }
                      : { type: "spring", stiffness: 480, damping: 40, mass: 0.6 }
                  }
                  className="overflow-hidden"
                >
                  <p className="prose-mute max-w-3xl pb-6 text-base leading-relaxed">
                    {item.answer}
                  </p>
                </motion.div>
              ) : null}
            </AnimatePresence>
          </div>
        );
      })}
    </div>
  );
}
