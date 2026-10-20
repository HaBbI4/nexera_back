import { Reveal } from "@/components/ui/reveal";
import { positioning } from "@/content/positioning";

export function WhySection() {
  return (
    <section className="section-pad border-t border-[var(--line)] py-24 md:py-32">
      <div className="container-site grid gap-12 lg:grid-cols-12">
        <Reveal className="lg:col-span-5">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Почему Nexera</p>
          <h2 className="font-display mt-4 text-[clamp(2rem,4vw,3.2rem)] font-semibold leading-[1.05]">
            Инженерия с вниманием к продукту
          </h2>
        </Reveal>
        <ul className="lg:col-span-7">
          {positioning.principles.map((item, i) => (
            <li
              key={item}
              className="flex gap-6 border-t border-[var(--line)] py-5 first:border-t-0"
            >
              <span className="font-mono text-xs text-[var(--accent)]">
                {String(i + 1).padStart(2, "0")}
              </span>
              <span className="text-lg tracking-tight md:text-xl">{item}</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
