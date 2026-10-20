import type { Metadata } from "next";
import Link from "next/link";
import { servicesCatalog } from "@/content/services";
import { MagneticLink } from "@/components/ui/magnetic-button";

export const metadata: Metadata = {
  title: "Услуги",
  description:
    "Корпоративные сайты, магазины, лендинги, веб- и мобильные приложения, SaaS/MVP, интеграции, UX/UI и поддержка — услуги Nexera.",
};

export default function ServicesPage() {
  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Услуги</p>
          <h1 className="font-display mt-4 max-w-[16ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            Что мы проектируем и собираем
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">
            От лендинга и корпоративного сайта до мобильного приложения и SaaS. Стоимость —
            после брифа; фиксированного прайса на сайте нет.
          </p>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site">
          <ul className="grid gap-6 lg:grid-cols-2">
            {servicesCatalog.map((s, i) => (
              <li key={s.slug} className="border border-[var(--line)] bg-[var(--bg-elevated)] p-8">
                <p className="font-mono text-xs text-[var(--porcelain-muted)]">
                  {String(i + 1).padStart(2, "0")}
                </p>
                <h2 className="font-display mt-3 text-2xl font-semibold tracking-tight">
                  <Link href={`/services/${s.slug}`} className="hover:text-[var(--accent)]">
                    {s.name}
                  </Link>
                </h2>
                <p className="prose-mute mt-4 text-sm leading-relaxed">{s.short}</p>
                <Link
                  href={`/services/${s.slug}`}
                  className="mt-6 inline-flex text-sm text-[var(--accent)] hover:underline"
                >
                  Подробнее →
                </Link>
              </li>
            ))}
          </ul>
          <div className="mt-12">
            <MagneticLink href="/brief" variant="primary">
              Обсудить задачу
            </MagneticLink>
          </div>
        </div>
      </section>
    </div>
  );
}
