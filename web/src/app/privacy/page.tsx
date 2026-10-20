import type { Metadata } from "next";
import { legalEntityPlaceholder } from "@/content/contacts";

export const metadata: Metadata = {
  title: "Политика обработки данных",
  description: "Информация об обработке персональных данных при отправке заявок на сайте Nexera.",
};

export default function PrivacyPage() {
  return (
    <div className="pt-[var(--header-h)]">
      <section className="section-pad py-20 md:py-28">
        <article className="container-site prose-mute max-w-3xl space-y-6 text-sm leading-relaxed md:text-base">
          <h1 className="font-display text-4xl font-semibold text-[var(--porcelain)]">
            Политика обработки данных
          </h1>
          <p>
            Оператор: <strong className="text-[var(--porcelain)]">{legalEntityPlaceholder}</strong>{" "}
            — юридическая заглушка до утверждения реквизитов.
          </p>
          <p>
            При отправке форм (`/brief`, `/contact` и быстрый контакт) вы передаёте имя, телефон и
            необязательные email/описание задачи. Данные сохраняются в backend Nexera (`leads`) для
            обработки обращения.
          </p>
          <p>
            Не передавайте избыточные персональные сведения. Полный документ политики, ИНН/ОГРН,
            адрес оператора и сроки хранения нужно заменить утверждённой редакцией — см.
            CONTENT_REQUIRED.md.
          </p>
        </article>
      </section>
    </div>
  );
}
