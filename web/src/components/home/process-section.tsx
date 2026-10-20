"use client";

/** Inspired by 21st Vertical How It Works Timeline (26902). */
import { Reveal } from "@/components/ui/reveal";
import { processSteps } from "@/content/process";

export function ProcessSection() {
  return (
    <section className="section-pad border-t border-[var(--line)] py-24 md:py-32">
      <div className="container-site grid gap-14 lg:grid-cols-12">
        <Reveal className="lg:col-span-4 lg:sticky lg:top-28 lg:self-start">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Процесс</p>
          <h2 className="font-display mt-4 text-[clamp(2rem,4vw,3.2rem)] font-semibold leading-[1.05]">
            Понятный путь от идеи до развития
          </h2>
          <p className="prose-mute mt-5 text-sm leading-relaxed md:text-base">
            Шесть этапов без театральности: вы всегда понимаете, где проект и что будет дальше.
          </p>
        </Reveal>

        <ol className="relative lg:col-span-8 lg:border-l lg:border-dashed lg:border-[var(--line)] lg:pl-12">
          {processSteps.map((step, i) => (
            <li key={step.title} className="relative mb-8 last:mb-0">
              <span className="mb-3 inline-flex h-9 w-9 items-center justify-center border border-[var(--line)] bg-[var(--bg-elevated)] font-mono text-xs text-[var(--accent)] lg:absolute lg:-left-[3.85rem] lg:top-0">
                {String(i + 1).padStart(2, "0")}
              </span>
              <Reveal delay={i * 0.05}>
                <h3 className="font-display text-xl font-semibold tracking-tight md:text-2xl">
                  {step.title}
                </h3>
                <p className="prose-mute mt-2 max-w-2xl text-sm leading-relaxed md:text-base">
                  {step.body}
                </p>
              </Reveal>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
