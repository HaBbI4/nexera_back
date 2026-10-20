"use client";

/** Service presentation inspired by 21st Service Card hover language (8223) — editorial rows, not card grid. */
import Link from "next/link";
import { motion, useReducedMotion } from "motion/react";
import { ArrowUpRight } from "lucide-react";
import type { Service } from "@/lib/api";
import { serviceSlug } from "@/lib/api";
import { Reveal } from "@/components/ui/reveal";

const accents = ["var(--accent)", "#7dd3fc", "#fbbf24", "#f472b6", "#a3e635"];

export function ServicesSection({ services }: { services: Service[] }) {
  const reduced = useReducedMotion();

  return (
    <section className="section-pad border-t border-[var(--line)] py-24 md:py-32" id="services">
      <div className="container-site">
        <Reveal>
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Услуги</p>
          <h2 className="font-display mt-4 max-w-[16ch] text-[clamp(2rem,4.5vw,3.5rem)] font-semibold leading-[1.05]">
            Цифровые продукты под реальные задачи бизнеса
          </h2>
          <p className="prose-mute mt-5 max-w-2xl text-base leading-relaxed">
            Каталог наполняется из админ-панели. Ниже — опубликованные услуги из API; если список пуст, откройте страницу услуг или заполните контент.
          </p>
        </Reveal>

        {services.length === 0 ? (
          <div className="mt-14 border border-dashed border-[var(--line)] p-10 text-[var(--porcelain-muted)]">
            <p className="font-display text-2xl text-[var(--porcelain)]">Услуги скоро появятся</p>
            <p className="mt-3 max-w-xl text-sm leading-relaxed">
              Опубликуйте услуги через админку backend (`POST /api/v1/admin/services`). Структура страницы уже готова.
            </p>
            <Link href="/services" className="mt-6 inline-flex text-[var(--accent)] hover:underline">
              Смотреть страницу услуг →
            </Link>
          </div>
        ) : (
          <ul className="mt-14 divide-y divide-[var(--line)] border-y border-[var(--line)]">
            {services.map((service, index) => (
              <li key={service.id}>
                <Link
                  href={`/services/${serviceSlug(service)}`}
                  className="group grid gap-4 py-8 md:grid-cols-12 md:items-end"
                >
                  <span className="font-mono text-xs text-[var(--porcelain-muted)] md:col-span-1">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <div className="md:col-span-7">
                    <h3 className="font-display text-2xl font-semibold tracking-tight md:text-3xl">
                      <span
                        className="mr-3 inline-block h-2.5 w-2.5 align-middle"
                        style={{ background: accents[index % accents.length] }}
                        aria-hidden
                      />
                      {service.name}
                    </h3>
                    {service.description ? (
                      <p className="prose-mute mt-3 max-w-2xl text-sm leading-relaxed md:text-base">
                        {service.description}
                      </p>
                    ) : null}
                  </div>
                  <div className="flex items-center justify-between gap-4 md:col-span-4 md:justify-end">
                    <span className="text-sm text-[var(--porcelain-muted)]">
                      {service.price != null
                        ? `от ${service.price.toLocaleString("ru-RU")} ₽`
                        : "Оценка по брифу"}
                    </span>
                    <motion.span
                      className="inline-flex h-10 w-10 items-center justify-center border border-[var(--line)] text-[var(--accent)] transition-colors group-hover:border-[var(--accent)] group-hover:bg-[var(--accent)] group-hover:text-[var(--ink)]"
                      whileHover={reduced ? undefined : { rotate: 45 }}
                    >
                      <ArrowUpRight className="h-4 w-4" />
                    </motion.span>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
