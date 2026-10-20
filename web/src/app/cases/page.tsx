import type { Metadata } from "next";
import Link from "next/link";
import { MagneticLink } from "@/components/ui/magnetic-button";

export const metadata: Metadata = {
  title: "Кейсы",
  description: "Портфолио цифровых продуктов студии Nexera. Реальные проекты — без вымышленных кейсов.",
};

export default function CasesPage() {
  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Кейсы</p>
          <h1 className="font-display mt-4 max-w-[14ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            Портфолио без вымысла
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">
            В текущем backend нет сущности кейсов. Пока список пуст — структура страниц и
            карточек готова к подключению данных.
          </p>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site border border-dashed border-[var(--line)] p-10 md:p-14">
          <p className="font-display text-3xl font-semibold">Пока нет опубликованных проектов</p>
          <p className="prose-mute mt-4 max-w-2xl text-sm leading-relaxed">
            Чтобы заполнить раздел, подготовьте материалы из{" "}
            <code className="text-[var(--porcelain)]">docs/CONTENT_REQUIRED.md</code> или
            добавьте домен кейсов в API. Мы сознательно не показываем чужие или вымышленные работы.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <MagneticLink href="/brief" variant="primary">
              Хотите похожий продукт?
            </MagneticLink>
            <Link href="/services" className="inline-flex items-center px-4 text-sm text-[var(--accent)] hover:underline">
              Смотреть услуги
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
