import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getServiceBySlug, servicesCatalog } from "@/content/services";
import { getCaseBySlug } from "@/content/cases";
import { MagneticLink } from "@/components/ui/magnetic-button";
import { ConceptBadge } from "@/components/cases/case-visual";

type Props = { params: Promise<{ slug: string }> };

export function generateStaticParams() {
  return servicesCatalog.map((s) => ({ slug: s.slug }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const service = getServiceBySlug(slug);
  if (!service) return { title: "Услуга" };
  return {
    title: service.name,
    description: service.short,
  };
}

export default async function ServiceDetailPage({ params }: Props) {
  const { slug } = await params;
  const service = getServiceBySlug(slug);
  if (!service) notFound();

  const related = (service.relatedCaseSlugs ?? [])
    .map((s) => getCaseBySlug(s))
    .filter(Boolean);

  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <Link
            href="/services"
            className="text-xs uppercase tracking-[0.16em] text-[var(--porcelain-muted)] hover:text-[var(--accent)]"
          >
            ← Все услуги
          </Link>
          <h1 className="font-display mt-6 max-w-[16ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            {service.name}
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">{service.short}</p>
          <div className="mt-10 flex flex-wrap gap-3">
            <MagneticLink href="/brief" variant="primary">
              Обсудить эту услугу
            </MagneticLink>
            <MagneticLink href="/contact" variant="ghost">
              Быстрый контакт
            </MagneticLink>
          </div>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site grid gap-12 lg:grid-cols-12">
          <div className="lg:col-span-7 space-y-8">
            <Block title="Какую проблему решаем">{service.problem}</Block>
            <Block title="Для кого">{service.forWhom}</Block>
            <Block title="Что получает клиент">{service.outcome}</Block>
            {related.length > 0 ? (
              <div>
                <h2 className="font-display text-2xl font-semibold tracking-tight">
                  Связанные концепты
                </h2>
                <ul className="mt-4 space-y-3">
                  {related.map((c) =>
                    c ? (
                      <li key={c.slug}>
                        <Link
                          href={`/cases/${c.slug}`}
                          className="inline-flex items-center gap-3 hover:text-[var(--accent)]"
                        >
                          {c.title}
                          {c.demo ? <ConceptBadge className="text-[var(--accent)]" /> : null}
                        </Link>
                      </li>
                    ) : null,
                  )}
                </ul>
              </div>
            ) : null}
          </div>
          <aside className="lg:col-span-5">
            <div className="border border-[var(--line)] bg-[var(--bg-elevated)] p-6">
              <p className="text-xs uppercase tracking-[0.16em] text-[var(--accent)]">Этапы</p>
              <ol className="mt-4 space-y-3">
                {service.stages.map((s, i) => (
                  <li key={s} className="flex gap-3 text-sm">
                    <span className="font-mono text-[var(--porcelain-muted)]">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <span>{s}</span>
                  </li>
                ))}
              </ol>
              <p className="mt-6 text-sm text-[var(--porcelain-muted)]">
                Стоимость рассчитывается индивидуально после брифа — без фиксированных «от» на
                сайте.
              </p>
            </div>
          </aside>
        </div>
      </section>
    </div>
  );
}

function Block({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h2 className="font-display text-2xl font-semibold tracking-tight">{title}</h2>
      <p className="prose-mute mt-3 text-sm leading-relaxed md:text-base">{children}</p>
    </div>
  );
}
