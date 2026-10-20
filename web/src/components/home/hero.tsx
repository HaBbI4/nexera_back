"use client";

/**
 * Hero composition inspired by 21st Animated Hero (5590) particle/cursor energy
 * and Agency Hero (28280) CTA structure — rebuilt for Signal Forge / brand-first rules.
 */
import { motion, useReducedMotion } from "motion/react";
import { KineticHeadline } from "@/components/ui/reveal";
import { MagneticLink } from "@/components/ui/magnetic-button";
import { site } from "@/lib/site";

export function Hero() {
  const reduced = useReducedMotion();

  return (
    <section className="atmosphere relative min-h-[100svh] overflow-hidden pt-[var(--header-h)]">
      <div className="site-grid pointer-events-none absolute inset-0 opacity-60" aria-hidden />

      <div className="section-pad container-site relative grid min-h-[calc(100svh-var(--header-h))] items-center gap-12 py-16 lg:grid-cols-12 lg:gap-8">
        <div className="lg:col-span-7">
          <p className="mb-6 text-xs uppercase tracking-[0.22em] text-[var(--accent)]">
            {site.name} — digital product studio
          </p>
          <KineticHeadline
            className="max-w-[14ch] text-[clamp(2.6rem,7vw,5.4rem)]"
            lines={["Сложные идеи", "— в продукты,", "которыми", "пользуются"]}
          />
          <motion.p
            className="prose-mute mt-8 max-w-xl text-base leading-relaxed md:text-lg"
            initial={reduced ? false : { opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: reduced ? 0 : 0.55, duration: 0.6 }}
          >
            Корпоративные сайты, магазины, лендинги, веб- и мобильные приложения, SaaS и
            интеграции — от стратегии и UX до запуска и поддержки.
          </motion.p>
          <motion.div
            className="mt-10 flex flex-wrap gap-3"
            initial={reduced ? false : { opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: reduced ? 0 : 0.7, duration: 0.55 }}
          >
            <MagneticLink href="/brief" variant="primary">
              Обсудить проект
            </MagneticLink>
            <MagneticLink href="/cases" variant="ghost">
              Смотреть кейсы
            </MagneticLink>
          </motion.div>
        </div>

        <div className="relative lg:col-span-5">
          <HeroStage reduced={!!reduced} />
        </div>
      </div>

      <div className="section-pad container-site absolute inset-x-0 bottom-6 flex items-center justify-between text-[10px] uppercase tracking-[0.2em] text-[var(--porcelain-muted)]">
        <span>Scroll</span>
        <motion.span
          aria-hidden
          animate={reduced ? undefined : { y: [0, 6, 0] }}
          transition={{ repeat: Infinity, duration: 1.8, ease: "easeInOut" }}
          className="inline-block h-8 w-px bg-[var(--accent)]"
        />
      </div>
    </section>
  );
}

function HeroStage({ reduced }: { reduced: boolean }) {
  return (
    <div className="relative mx-auto aspect-[4/5] w-full max-w-md lg:max-w-none">
      <div
        className="absolute inset-0 rounded-sm border border-[var(--line)] bg-[var(--bg-elevated)]"
        style={{
          backgroundImage:
            "linear-gradient(145deg, color-mix(in oklab, var(--accent) 12%, transparent), transparent 45%), linear-gradient(var(--bg-elevated), var(--bg-soft))",
        }}
      />
      {[0, 1, 2].map((i) => (
        <motion.div
          key={i}
          className="absolute left-[8%] right-[8%] overflow-hidden border border-[var(--line)] bg-[color-mix(in_oklab,var(--bg)_92%,transparent)] p-4 shadow-[0_20px_60px_rgba(0,0,0,0.35)] backdrop-blur-sm"
          style={{ top: `${12 + i * 18}%` }}
          initial={reduced ? false : { opacity: 0, y: 24, rotate: -2 + i }}
          animate={{ opacity: 1, y: 0, rotate: -2 + i }}
          transition={{ delay: reduced ? 0 : 0.4 + i * 0.12, duration: 0.7, ease: [0.23, 1, 0.32, 1] }}
        >
          <div className="mb-3 flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-[var(--accent)]" />
            <span className="text-[10px] uppercase tracking-[0.16em] text-[var(--porcelain-muted)]">
              {i === 0 ? "Product surface" : i === 1 ? "API layer" : "Ops console"}
            </span>
          </div>
          <div className="space-y-2">
            <div className="h-2 w-[72%] bg-[var(--line)]" />
            <div className="h-2 w-[48%] bg-[var(--line)]" />
            <div className="mt-4 grid grid-cols-3 gap-2">
              <div className="col-span-2 h-16 border border-[var(--line)] bg-[var(--accent-dim)]" />
              <div className="h-16 border border-[var(--line)]" />
            </div>
          </div>
        </motion.div>
      ))}
    </div>
  );
}
