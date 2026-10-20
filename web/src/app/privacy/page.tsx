import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Политика обработки данных",
  description: "Информация об обработке персональных данных при отправке заявок на сайте Nexera.",
  robots: { index: true, follow: true },
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
            Этот текст — рабочая заглушка до утверждения юридической редакции компании. При
            отправке форм (`/brief`, `/contact` и быстрый контакт) вы передаёте имя, телефон и
            необязательные email/описание задачи.
          </p>
          <p>
            Данные сохраняются в backend Nexera (`leads`) для обработки обращения. Не передавайте
            в формах избыточные персональные сведения.
          </p>
          <p>
            Полный документ политики, реквизиты оператора и сроки хранения нужно предоставить
            команде — см. <code className="text-[var(--porcelain)]">docs/CONTENT_REQUIRED.md</code>.
          </p>
        </article>
      </section>
    </div>
  );
}
