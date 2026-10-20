import { Accordion } from "@/components/ui/accordion";
import { Reveal } from "@/components/ui/reveal";
import { fallbackFaq } from "@/content/positioning";
import type { FAQ } from "@/lib/api";

export function FaqSection({ faqs }: { faqs: FAQ[] }) {
  const items =
    faqs.length > 0
      ? faqs.map((f) => ({ id: String(f.id), question: f.question, answer: f.answer }))
      : fallbackFaq.map((f) => ({ id: f.id, question: f.question, answer: f.answer }));

  return (
    <section className="section-pad border-t border-[var(--line)] py-24 md:py-32">
      <div className="container-site grid gap-12 lg:grid-cols-12">
        <Reveal className="lg:col-span-4">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">FAQ</p>
          <h2 className="font-display mt-4 text-[clamp(2rem,4vw,3.2rem)] font-semibold leading-[1.05]">
            Частые вопросы
          </h2>
          <p className="prose-mute mt-4 text-sm leading-relaxed">
            {faqs.length > 0
              ? "Ответы из опубликованного FAQ backend."
              : "Пока в API нет FAQ — показываем аккуратные общие ответы без выдуманных условий компании."}
          </p>
        </Reveal>
        <div className="lg:col-span-8">
          <Accordion items={items} />
        </div>
      </div>
    </section>
  );
}
