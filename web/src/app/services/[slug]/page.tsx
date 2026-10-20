import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getService, getServices, parseServiceIdFromSlug, serviceSlug } from "@/lib/api";
import { MagneticLink } from "@/components/ui/magnetic-button";
import { processSteps } from "@/content/process";

type Props = { params: Promise<{ slug: string }> };

export async function generateStaticParams() {
  const services = await getServices();
  return services.map((s) => ({ slug: serviceSlug(s) }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const id = parseServiceIdFromSlug(slug);
  if (!id) return { title: "Услуга" };
  const service = await getService(id);
  if (!service) return { title: "Услуга" };
  return {
    title: service.name,
    description: service.description?.slice(0, 160) || `Услуга Nexera: ${service.name}`,
  };
}

export default async function ServiceDetailPage({ params }: Props) {
  const { slug } = await params;
  const id = parseServiceIdFromSlug(slug);
  if (!id) notFound();
  const service = await getService(id);
  if (!service) notFound();

  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <Link href="/services" className="text-xs uppercase tracking-[0.16em] text-[var(--porcelain-muted)] hover:text-[var(--accent)]">
            ← Все услуги
          </Link>
          <h1 className="font-display mt-6 max-w-[16ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            {service.name}
          </h1>
          {service.description ? (
            <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">{service.description}</p>
          ) : null}
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
            <Block title="Какую проблему решаем">
              Помогаем превратить бизнес-задачу в рабочий цифровой продукт — с понятными
              сценариями для пользователей и устойчивой технической основой.
            </Block>
            <Block title="Для кого">
              Для команд и компаний, которым нужно веб-приложение, мобильный продукт, SaaS или
              развитие существующей системы — когда важны и UX, и инженерия.
            </Block>
            <Block title="Что получает клиент">
              Согласованную концепцию, спроектированный интерфейс, реализованный продукт и план
              развития после запуска. Точный состав фиксируем после брифа.
            </Block>
          </div>
          <aside className="lg:col-span-5">
            <div className="border border-[var(--line)] bg-[var(--bg-elevated)] p-6">
              <p className="text-xs uppercase tracking-[0.16em] text-[var(--accent)]">Этапы</p>
              <ol className="mt-4 space-y-3">
                {processSteps.map((s, i) => (
                  <li key={s.title} className="flex gap-3 text-sm">
                    <span className="font-mono text-[var(--porcelain-muted)]">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <span>{s.title}</span>
                  </li>
                ))}
              </ol>
              <p className="mt-6 text-sm text-[var(--porcelain-muted)]">
                {service.price != null
                  ? `Ориентир в каталоге: от ${service.price.toLocaleString("ru-RU")} ₽. Финальная оценка — после брифа.`
                  : "Стоимость рассчитывается индивидуально после брифа."}
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
