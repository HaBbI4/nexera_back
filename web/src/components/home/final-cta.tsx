"use client";

/** Final CTA inspired by 21st CTA sections 19355 / 28155 — large type, no nested card clutter. */
import { motion, useReducedMotion } from "motion/react";
import { MagneticLink } from "@/components/ui/magnetic-button";

export function FinalCta() {
  const reduced = useReducedMotion();
  return (
    <section className="relative overflow-hidden border-t border-[var(--line)]">
      <div
        className="absolute inset-0"
        style={{
          background:
            "radial-gradient(ellipse 80% 80% at 50% 120%, color-mix(in oklab, var(--accent) 35%, transparent), transparent 55%), var(--bg)",
        }}
      />
      <div className="section-pad container-site relative py-28 md:py-36">
        <motion.p
          className="text-xs uppercase tracking-[0.22em] text-[var(--accent)]"
          initial={reduced ? false : { opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true }}
        >
          Следующий шаг
        </motion.p>
        <motion.h2
          className="font-display mt-6 max-w-[12ch] text-[clamp(2.6rem,8vw,6rem)] font-semibold leading-[0.95]"
          initial={reduced ? false : { opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.7, ease: [0.23, 1, 0.32, 1] }}
        >
          У вас есть идея. Давайте сделаем продукт.
        </motion.h2>
        <p className="prose-mute mt-6 max-w-xl text-base leading-relaxed">
          Короткий бриф — и мы вернёмся с вопросами и первой оценкой объёма. Без навязчивых скриптов.
        </p>
        <div className="mt-10 flex flex-wrap gap-3">
          <MagneticLink href="/brief" variant="primary" className="!px-8 !py-4 text-base">
            Заполнить бриф
          </MagneticLink>
          <MagneticLink href="/contact" variant="ghost" className="!px-8 !py-4 text-base">
            Написать коротко
          </MagneticLink>
        </div>
      </div>
    </section>
  );
}
