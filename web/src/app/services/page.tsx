import type { Metadata } from "next";
import Link from "next/link";
import { getServices, serviceSlug } from "@/lib/api";
import { MagneticLink } from "@/components/ui/magnetic-button";

export const metadata: Metadata = {
  title: "Услуги",
  description:
    "Разработка веб-приложений, мобильных продуктов, SaaS и корпоративных систем — услуги студии Nexera.",
};

export default async function ServicesPage() {
  const services = await getServices();

  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Услуги</p>
          <h1 className="font-display mt-4 max-w-[16ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            Что мы проектируем и собираем
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">
            Каждая услуга объясняет задачу, аудиторию и результат. Контент подтягивается из
            публичного API backend.
          </p>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site">
          {services.length === 0 ? (
            <div className="border border-dashed border-[var(--line)] p-10">
              <p className="font-display text-2xl">Каталог пока пуст</p>
              <p className="prose-mute mt-3 max-w-xl text-sm leading-relaxed">
                Опубликуйте услуги в админ-панели. См. docs/CONTENT_REQUIRED.md.
              </p>
              <MagneticLink href="/brief" className="mt-8" variant="primary">
                Обсудить задачу без каталога
              </MagneticLink>
            </div>
          ) : (
            <ul className="grid gap-6 lg:grid-cols-2">
              {services.map((s) => (
                <li key={s.id} className="border border-[var(--line)] bg-[var(--bg-elevated)] p-8">
                  <h2 className="font-display text-2xl font-semibold tracking-tight">
                    <Link href={`/services/${serviceSlug(s)}`} className="hover:text-[var(--accent)]">
                      {s.name}
                    </Link>
                  </h2>
                  {s.description ? (
                    <p className="prose-mute mt-4 text-sm leading-relaxed">{s.description}</p>
                  ) : (
                    <p className="prose-mute mt-4 text-sm">Описание появится после заполнения в админке.</p>
                  )}
                  <p className="mt-6 text-sm text-[var(--porcelain-muted)]">
                    {s.price != null
                      ? `Ориентир: от ${s.price.toLocaleString("ru-RU")} ₽`
                      : "Стоимость — по оценке после брифа"}
                  </p>
                  <Link
                    href={`/services/${serviceSlug(s)}`}
                    className="mt-6 inline-flex text-sm text-[var(--accent)] hover:underline"
                  >
                    Подробнее →
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>
    </div>
  );
}
