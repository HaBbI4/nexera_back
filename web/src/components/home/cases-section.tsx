"use client";

/** Case showcase inspired by 21st Project Showcase (9607). */
import Link from "next/link";
import { useRef, useState } from "react";
import { motion, useReducedMotion } from "motion/react";
import type { CaseContent } from "@/content/cases";
import { CaseVisual, ConceptBadge } from "@/components/cases/case-visual";
import { Reveal } from "@/components/ui/reveal";

export function CasesSection({ cases }: { cases: CaseContent[] }) {
  const reduced = useReducedMotion();
  const ref = useRef<HTMLDivElement>(null);
  const [hover, setHover] = useState<number | null>(null);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const featured = cases.slice(0, 4);

  return (
    <section
      className="section-pad border-t border-[var(--line)] bg-[var(--porcelain)] py-24 text-[var(--ink)] md:py-32"
      id="cases"
    >
      <div className="container-site" ref={ref}>
        <Reveal>
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent-deep)]">Кейсы</p>
          <div className="mt-4 flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
            <h2 className="font-display max-w-[14ch] text-[clamp(2rem,4.5vw,3.5rem)] font-semibold leading-[1.05]">
              Избранные концепты
            </h2>
            <Link
              href="/cases"
              className="text-sm uppercase tracking-[0.14em] text-[var(--ink-muted)] underline-offset-4 hover:text-[var(--ink)] hover:underline"
            >
              Все кейсы
            </Link>
          </div>
          <p className="mt-4 max-w-2xl text-sm text-[var(--ink-muted)]">
            Демо-портфолио с меткой «Концепт» — без чужих клиентов и фейковых метрик.
          </p>
        </Reveal>

        <div
          className="relative mt-14"
          onMouseMove={(e) => {
            if (!ref.current || reduced) return;
            const rect = ref.current.getBoundingClientRect();
            setPos({ x: e.clientX - rect.left, y: e.clientY - rect.top });
          }}
        >
          {!reduced && hover != null && featured[hover] ? (
            <motion.div
              className="pointer-events-none absolute z-20 hidden h-40 w-60 overflow-hidden border border-[var(--ink)] md:block"
              style={{ left: pos.x + 24, top: pos.y - 80 }}
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
            >
              <CaseVisual
                visual={featured[hover].visual}
                accent={featured[hover].accent}
                className="h-full w-full"
              />
            </motion.div>
          ) : null}

          <ul className="divide-y divide-[color-mix(in_oklab,var(--ink)_14%,transparent)] border-y border-[color-mix(in_oklab,var(--ink)_14%,transparent)]">
            {featured.map((item, index) => (
              <li key={item.slug}>
                <Link
                  href={`/cases/${item.slug}`}
                  className="grid gap-2 py-7 md:grid-cols-12 md:items-center"
                  onMouseEnter={() => setHover(index)}
                  onMouseLeave={() => setHover(null)}
                >
                  <span className="font-mono text-xs text-[var(--ink-muted)] md:col-span-1">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <span className="font-display flex flex-wrap items-center gap-2 text-2xl font-semibold md:col-span-4">
                    {item.title}
                    {item.demo ? (
                      <ConceptBadge className="text-[var(--accent-deep)]" />
                    ) : null}
                  </span>
                  <span className="text-sm text-[var(--ink-muted)] md:col-span-3">
                    {item.productType} · {item.industry}
                  </span>
                  <span className="text-sm text-[var(--ink-muted)] md:col-span-4">
                    {item.summary}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}
