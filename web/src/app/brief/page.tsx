import type { Metadata } from "next";
import { BriefWizard } from "@/components/forms/brief-wizard";

export const metadata: Metadata = {
  title: "Бриф",
  description:
    "Интерактивный бриф на разработку цифрового продукта — заявка сохраняется в Nexera backend.",
};

export default function BriefPage() {
  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-16 md:py-24">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Бриф</p>
          <h1 className="font-display mt-4 max-w-[16ch] text-[clamp(2.2rem,5.5vw,4rem)] font-semibold leading-[0.95]">
            Интерактивная консультация по вашему продукту
          </h1>
          <p className="prose-mute mt-5 max-w-2xl text-base leading-relaxed">
            Семь коротких шагов. Ответы сохраняются между шагами, заявка уходит на сервер только
            после успешного ответа API.
          </p>
        </div>
      </section>
      <section className="section-pad py-12 md:py-16">
        <div className="container-site max-w-3xl">
          <BriefWizard />
        </div>
      </section>
    </div>
  );
}
