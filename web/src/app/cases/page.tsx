import type { Metadata } from "next";
import Link from "next/link";
import { getPublishedCases } from "@/content/cases";
import { ConceptBadge } from "@/components/cases/case-visual";
import { MagneticLink } from "@/components/ui/magnetic-button";

export const metadata: Metadata = {
  title: "Кейсы",
  description:
    "Концептуальные кейсы Nexera: e-commerce, корпоративные сайты, мобильные приложения и SaaS. Реальные проекты появятся позже.",
};

export default function CasesPage() {
  const cases = getPublishedCases();

  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Кейсы</p>
          <h1 className="font-display mt-4 max-w-[14ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            Портфолио и концепты
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">
            Сейчас показаны демо-кейсы с меткой «Концепт» — без чужих клиентов и выдуманных
            метрик. Замените на реальные проекты, убрав записи с{" "}
            <code className="text-[var(--porcelain)]">demo: true</code> в{" "}
            <code className="text-[var(--porcelain)]">content/cases.ts</code>.
          </p>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site">
          <ul className="divide-y divide-[var(--line)] border-y border-[var(--line)]">
            {cases.map((item, index) => (
              <li key={item.slug}>
                <Link
                  href={`/cases/${item.slug}`}
                  className="grid gap-3 py-8 md:grid-cols-12 md:items-center"
                >
                  <span className="font-mono text-xs text-[var(--porcelain-muted)] md:col-span-1">
                    {String(index + 1).padStart(2, "0")}
                  </span>
                  <span className="font-display flex flex-wrap items-center gap-3 text-2xl font-semibold md:col-span-5">
                    {item.title}
                    {item.demo ? <ConceptBadge className="text-[var(--accent)]" /> : null}
                  </span>
                  <span className="text-sm text-[var(--porcelain-muted)] md:col-span-3">
                    {item.productType} · {item.industry}
                  </span>
                  <span className="text-sm text-[var(--porcelain-muted)] md:col-span-3">
                    {item.summary}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
          <MagneticLink href="/brief" className="mt-12" variant="primary">
            Хотите похожий продукт?
          </MagneticLink>
        </div>
      </section>
    </div>
  );
}
