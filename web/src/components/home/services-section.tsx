"use client";

/** Service presentation inspired by 21st Service Card hover language (8223). */
import Link from "next/link";
import { motion, useReducedMotion } from "motion/react";
import { ArrowUpRight } from "lucide-react";
import type { ServiceContent } from "@/content/services";
import { Reveal } from "@/components/ui/reveal";

const accents = ["var(--accent)", "#7dd3fc", "#fbbf24", "#f472b6", "#a3e635"];

export function ServicesSection({ services }: { services: ServiceContent[] }) {
  const reduced = useReducedMotion();
  const list = services.slice(0, 6);

  return (
    <section className="section-pad border-t border-[var(--line)] py-24 md:py-32" id="services">
      <div className="container-site">
        <Reveal>
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Услуги</p>
          <h2 className="font-display mt-4 max-w-[16ch] text-[clamp(2rem,4.5vw,3.5rem)] font-semibold leading-[1.05]">
            Сайты, приложения и цифровые сервисы
          </h2>
          <p className="prose-mute mt-5 max-w-2xl text-base leading-relaxed">
            Корпоративные сайты, магазины, лендинги, веб- и мобильные продукты, SaaS/MVP,
            интеграции и поддержка.
          </p>
        </Reveal>

        <ul className="mt-14 divide-y divide-[var(--line)] border-y border-[var(--line)]">
          {list.map((service, index) => (
            <li key={service.slug}>
              <Link
                href={`/services/${service.slug}`}
                className="group grid gap-4 py-8 md:grid-cols-12 md:items-end"
              >
                <span className="font-mono text-xs text-[var(--porcelain-muted)] md:col-span-1">
                  {String(index + 1).padStart(2, "0")}
                </span>
                <div className="md:col-span-8">
                  <h3 className="font-display text-2xl font-semibold tracking-tight md:text-3xl">
                    <span
                      className="mr-3 inline-block h-2.5 w-2.5 align-middle"
                      style={{ background: accents[index % accents.length] }}
                      aria-hidden
                    />
                    {service.name}
                  </h3>
                  <p className="prose-mute mt-3 max-w-2xl text-sm leading-relaxed md:text-base">
                    {service.short}
                  </p>
                </div>
                <div className="flex justify-end md:col-span-3">
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
        <Link
          href="/services"
          className="mt-8 inline-flex text-sm text-[var(--accent)] underline-offset-4 hover:underline"
        >
          Все услуги →
        </Link>
      </div>
    </section>
  );
}
