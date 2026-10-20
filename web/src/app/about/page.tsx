import type { Metadata } from "next";
import { aboutCopy, positioning } from "@/content/positioning";
import { MagneticLink } from "@/components/ui/magnetic-button";
import { legalEntityPlaceholder } from "@/content/contacts";

export const metadata: Metadata = {
  title: "О студии",
  description: aboutCopy.body,
};

export default function AboutPage() {
  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">О студии</p>
          <h1 className="font-display mt-4 max-w-[14ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            {aboutCopy.title}
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">{aboutCopy.body}</p>
          <p className="mt-4 text-sm text-[var(--porcelain-muted)]">
            Юрлицо: {legalEntityPlaceholder}{" "}
            <span className="text-[var(--accent)]">(заглушка)</span>
          </p>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site grid gap-12 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <h2 className="font-display text-3xl font-semibold">Во что верим</h2>
          </div>
          <ul className="space-y-4 lg:col-span-7">
            {aboutCopy.beliefs.map((b) => (
              <li key={b} className="border-l-2 border-[var(--accent)] pl-5 text-base leading-relaxed">
                {b}
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section className="section-pad border-t border-[var(--line)] py-16 md:py-24">
        <div className="container-site">
          <h2 className="font-display text-3xl font-semibold">Специализации</h2>
          <ul className="mt-8 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {aboutCopy.specializations.map((s) => (
              <li key={s} className="border border-[var(--line)] bg-[var(--bg-elevated)] p-4 text-sm">
                {s}
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section className="section-pad border-t border-[var(--line)] py-16 md:py-24">
        <div className="container-site">
          <h2 className="font-display text-3xl font-semibold">Как подходим к разработке</h2>
          <p className="prose-mute mt-4 max-w-2xl">{positioning.lead}</p>
          <div className="mt-10 grid gap-px bg-[var(--line)] sm:grid-cols-2">
            {positioning.pillars.map((p) => (
              <div key={p.title} className="bg-[var(--bg)] p-6 md:p-8">
                <h3 className="font-display text-xl font-semibold">{p.title}</h3>
                <p className="prose-mute mt-3 text-sm leading-relaxed">{p.body}</p>
              </div>
            ))}
          </div>
          <MagneticLink href="/brief" className="mt-12" variant="primary">
            Обсудить проект
          </MagneticLink>
        </div>
      </section>
    </div>
  );
}
