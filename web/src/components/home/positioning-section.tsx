import { Reveal } from "@/components/ui/reveal";
import { positioning } from "@/content/positioning";

export function PositioningSection() {
  return (
    <section className="section-pad border-t border-[var(--line)] py-24 md:py-32">
      <div className="container-site">
        <Reveal>
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Подход</p>
          <h2 className="font-display mt-4 max-w-[18ch] text-[clamp(2rem,4.5vw,3.4rem)] font-semibold leading-[1.05]">
            {positioning.headline}
          </h2>
          <p className="prose-mute mt-5 max-w-2xl text-base leading-relaxed">{positioning.lead}</p>
        </Reveal>
        <div className="mt-14 grid gap-px bg-[var(--line)] sm:grid-cols-2 lg:grid-cols-4">
          {positioning.pillars.map((p) => (
            <div key={p.title} className="bg-[var(--bg)] p-6 md:p-8">
              <h3 className="font-display text-xl font-semibold">{p.title}</h3>
              <p className="prose-mute mt-3 text-sm leading-relaxed">{p.body}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
