import { Hero } from "@/components/home/hero";
import { PositioningSection } from "@/components/home/positioning-section";
import { ServicesSection } from "@/components/home/services-section";
import { CasesSection } from "@/components/home/cases-section";
import { ProcessSection } from "@/components/home/process-section";
import { WhySection } from "@/components/home/why-section";
import { FaqSection } from "@/components/home/faq-section";
import { FinalCta } from "@/components/home/final-cta";
import { QuickContactForm } from "@/components/forms/quick-contact";
import { Reveal } from "@/components/ui/reveal";
import { aboutCopy } from "@/content/positioning";
import { getFaqs, getReviews, getReviewSummary, getServices } from "@/lib/api";

export default async function HomePage() {
  const [services, faqs, reviews, summary] = await Promise.all([
    getServices(),
    getFaqs(),
    getReviews(),
    getReviewSummary(),
  ]);

  // Cases entity does not exist in backend yet — never invent.
  const cases: [] = [];

  return (
    <>
      <Hero />
      <PositioningSection />
      <ServicesSection services={services} />
      <CasesSection cases={cases} />
      <ProcessSection />
      <WhySection />

      <section className="section-pad border-t border-[var(--line)] py-24 md:py-32">
        <div className="container-site grid gap-10 lg:grid-cols-12">
          <Reveal className="lg:col-span-5">
            <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">О студии</p>
            <h2 className="font-display mt-4 text-[clamp(2rem,4vw,3.2rem)] font-semibold leading-[1.05]">
              {aboutCopy.title}
            </h2>
            <p className="prose-mute mt-5 text-base leading-relaxed">{aboutCopy.body}</p>
            <a
              href="/about"
              className="mt-6 inline-flex text-sm text-[var(--accent)] underline-offset-4 hover:underline"
            >
              Подробнее о подходе →
            </a>
          </Reveal>
          <ul className="lg:col-span-7 space-y-4">
            {aboutCopy.beliefs.map((b) => (
              <li
                key={b}
                className="border border-[var(--line)] bg-[var(--bg-elevated)] p-5 text-sm leading-relaxed md:text-base"
              >
                {b}
              </li>
            ))}
          </ul>
        </div>
      </section>

      {summary && summary.count > 0 && reviews.length > 0 ? (
        <section className="section-pad border-t border-[var(--line)] py-24">
          <div className="container-site">
            <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Отзывы</p>
            <h2 className="font-display mt-4 text-3xl font-semibold">
              {summary.average_rating.toFixed(1)} из 5 · {summary.count}{" "}
              {summary.count === 1 ? "отзыв" : "отзывов"}
            </h2>
            <ul className="mt-10 grid gap-4 md:grid-cols-2">
              {reviews.slice(0, 4).map((r) => (
                <li key={r.id} className="border border-[var(--line)] p-6">
                  <p className="text-sm text-[var(--accent)]">{"★".repeat(r.rating)}</p>
                  <p className="mt-3 text-sm leading-relaxed">{r.text}</p>
                  <p className="mt-4 text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                    {r.name}
                  </p>
                </li>
              ))}
            </ul>
          </div>
        </section>
      ) : null}

      <FaqSection faqs={faqs} />

      <section className="section-pad border-t border-[var(--line)] py-24 md:py-32">
        <div className="container-site grid gap-12 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Быстрый контакт</p>
            <h2 className="font-display mt-4 text-[clamp(2rem,4vw,3rem)] font-semibold leading-[1.05]">
              Коротко опишите задачу
            </h2>
            <p className="prose-mute mt-4 text-sm leading-relaxed">
              Заявка уходит в backend (`POST /api/v1/leads`) и доступна в админке. Без имитации успеха.
            </p>
          </div>
          <div className="lg:col-span-7">
            <QuickContactForm source="quick_contact" />
          </div>
        </div>
      </section>

      <FinalCta />
    </>
  );
}
