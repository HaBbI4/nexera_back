import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { casesCatalog, getCaseBySlug } from "@/content/cases";
import { CaseVisual, ConceptBadge } from "@/components/cases/case-visual";
import { MagneticLink } from "@/components/ui/magnetic-button";

type Props = { params: Promise<{ slug: string }> };

export function generateStaticParams() {
  return casesCatalog.map((c) => ({ slug: c.slug }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const item = getCaseBySlug(slug);
  if (!item) return { title: "Кейс" };
  return {
    title: item.demo ? `${item.title} (концепт)` : item.title,
    description: item.summary,
    robots: item.demo ? { index: false, follow: true } : { index: true, follow: true },
  };
}

export default async function CaseDetailPage({ params }: Props) {
  const { slug } = await params;
  const item = getCaseBySlug(slug);
  if (!item) notFound();

  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-16 md:py-24">
        <div className="container-site grid gap-10 lg:grid-cols-12 lg:items-end">
          <div className="lg:col-span-7">
            <Link
              href="/cases"
              className="text-xs uppercase tracking-[0.16em] text-[var(--porcelain-muted)] hover:text-[var(--accent)]"
            >
              ← Все кейсы
            </Link>
            <div className="mt-6 flex flex-wrap items-center gap-3">
              {item.demo ? <ConceptBadge className="text-[var(--accent)]" /> : null}
              <span className="text-xs uppercase tracking-[0.16em] text-[var(--porcelain-muted)]">
                {item.productType} · {item.industry}
              </span>
            </div>
            <h1 className="font-display mt-4 text-[clamp(2.2rem,5vw,4rem)] font-semibold leading-[0.95]">
              {item.title}
            </h1>
            <p className="prose-mute mt-5 max-w-2xl text-base leading-relaxed">{item.summary}</p>
            {item.demo ? (
              <p className="mt-4 text-sm text-[var(--accent)]">
                Демо-кейс: без имени клиента и без заявленных цифр роста. Показаны цели и подход.
              </p>
            ) : null}
          </div>
          <div className="lg:col-span-5">
            <CaseVisual visual={item.visual} accent={item.accent} className="aspect-[4/3]" />
          </div>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site grid gap-12 lg:grid-cols-12">
          <div className="lg:col-span-7 space-y-10">
            <Block title="Контекст">{item.context}</Block>
            <Block title="Исходная задача">{item.challenge}</Block>
            <Block title="Наше решение">{item.solution}</Block>
            <Block title="UX/UI">{item.uxNotes}</Block>
            <Block title="Техническая реализация">{item.techNotes}</Block>
          </div>
          <aside className="lg:col-span-5 space-y-6">
            <div className="border border-[var(--line)] bg-[var(--bg-elevated)] p-6">
              <p className="text-xs uppercase tracking-[0.16em] text-[var(--accent)]">
                Цели подхода
              </p>
              <ul className="mt-4 space-y-3 text-sm">
                {item.goals.map((g) => (
                  <li key={g} className="flex gap-2">
                    <span className="text-[var(--accent)]">→</span>
                    <span>{g}</span>
                  </li>
                ))}
              </ul>
              <p className="mt-4 text-xs text-[var(--porcelain-muted)]">
                Это ориентиры проектирования, не подтверждённые бизнес-результаты.
              </p>
            </div>
            <div className="border border-[var(--line)] p-6">
              <p className="text-xs uppercase tracking-[0.16em] text-[var(--porcelain-muted)]">
                Технологии
              </p>
              <ul className="mt-3 flex flex-wrap gap-2">
                {item.technologies.map((t) => (
                  <li
                    key={t}
                    className="border border-[var(--line)] px-2 py-1 text-xs text-[var(--porcelain-muted)]"
                  >
                    {t}
                  </li>
                ))}
              </ul>
            </div>
            <MagneticLink href="/brief" variant="primary" className="w-full !justify-center">
              Хотите похожий продукт?
            </MagneticLink>
          </aside>
        </div>
      </section>
    </div>
  );
}

function Block({ title, children }: { title: string; children: string }) {
  return (
    <div>
      <h2 className="font-display text-2xl font-semibold tracking-tight">{title}</h2>
      <p className="prose-mute mt-3 text-sm leading-relaxed md:text-base">{children}</p>
    </div>
  );
}
