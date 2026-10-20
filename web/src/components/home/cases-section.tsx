"use client";

/**
 * Case showcase inspired by 21st Project Showcase (9607) — list + floating preview.
 * No invented cases: empty state when portfolio data is absent.
 */
import Link from "next/link";
import { useRef, useState } from "react";
import { motion, useReducedMotion } from "motion/react";
import { Reveal } from "@/components/ui/reveal";

export type CasePreview = {
  slug: string;
  title: string;
  type: string;
  industry: string;
  summary: string;
  image?: string;
};

export function CasesSection({ cases }: { cases: CasePreview[] }) {
  const reduced = useReducedMotion();
  const ref = useRef<HTMLDivElement>(null);
  const [hover, setHover] = useState<number | null>(null);
  const [pos, setPos] = useState({ x: 0, y: 0 });

  return (
    <section className="section-pad border-t border-[var(--line)] bg-[var(--porcelain)] py-24 text-[var(--ink)] md:py-32" id="cases">
      <div className="container-site" ref={ref}>
        <Reveal>
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent-deep)]">Кейсы</p>
          <div className="mt-4 flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
            <h2 className="font-display max-w-[14ch] text-[clamp(2rem,4.5vw,3.5rem)] font-semibold leading-[1.05]">
              Работы, которые можно показать
            </h2>
            <Link
              href="/cases"
              className="text-sm uppercase tracking-[0.14em] text-[var(--ink-muted)] underline-offset-4 hover:text-[var(--ink)] hover:underline"
            >
              Все кейсы
            </Link>
          </div>
        </Reveal>

        {cases.length === 0 ? (
          <div className="mt-14 border border-dashed border-[color-mix(in_oklab,var(--ink)_18%,transparent)] p-10">
            <p className="font-display text-2xl">Портфолио пока пустое</p>
            <p className="mt-3 max-w-2xl text-sm leading-relaxed text-[var(--ink-muted)]">
              В backend ещё нет сущности кейсов — мы не публикуем вымышленные проекты.
              Когда появятся материалы, добавьте их через админку или см.{" "}
              <code className="text-[var(--ink)]">docs/CONTENT_REQUIRED.md</code>.
            </p>
          </div>
        ) : (
          <div
            className="relative mt-14"
            onMouseMove={(e) => {
              if (!ref.current || reduced) return;
              const rect = ref.current.getBoundingClientRect();
              setPos({ x: e.clientX - rect.left, y: e.clientY - rect.top });
            }}
          >
            {!reduced && hover != null && cases[hover]?.image ? (
              <motion.div
                className="pointer-events-none absolute z-20 hidden h-40 w-60 overflow-hidden border border-[var(--ink)] bg-[var(--bg)] md:block"
                style={{ left: pos.x + 24, top: pos.y - 80 }}
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={cases[hover].image}
                  alt=""
                  className="h-full w-full object-cover"
                />
              </motion.div>
            ) : null}

            <ul className="divide-y divide-[color-mix(in_oklab,var(--ink)_14%,transparent)] border-y border-[color-mix(in_oklab,var(--ink)_14%,transparent)]">
              {cases.map((item, index) => (
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
                    <span className="font-display text-2xl font-semibold md:col-span-4">
                      {item.title}
                    </span>
                    <span className="text-sm text-[var(--ink-muted)] md:col-span-3">
                      {item.type} · {item.industry}
                    </span>
                    <span className="text-sm text-[var(--ink-muted)] md:col-span-4">
                      {item.summary}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </section>
  );
}
