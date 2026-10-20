import type { Metadata } from "next";
import { QuickContactForm } from "@/components/forms/quick-contact";
import { MagneticLink } from "@/components/ui/magnetic-button";
import {
  contactsBanner,
  getDisplayContacts,
  legalEntityPlaceholder,
} from "@/content/contacts";

export const metadata: Metadata = {
  title: "Контакты",
  description: "Связаться со студией Nexera — форма и каналы связи.",
};

export default function ContactPage() {
  const contacts = getDisplayContacts();

  return (
    <div className="pt-[var(--header-h)]">
      <section className="atmosphere section-pad border-b border-[var(--line)] py-20 md:py-28">
        <div className="container-site">
          <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Контакты</p>
          <h1 className="font-display mt-4 max-w-[12ch] text-[clamp(2.4rem,6vw,4.5rem)] font-semibold leading-[0.95]">
            Давайте поговорим о продукте
          </h1>
          <p className="prose-mute mt-6 max-w-2xl text-base leading-relaxed">
            Короткая форма или подробный бриф — оба пути сохраняют заявку в backend.
          </p>
        </div>
      </section>

      <section className="section-pad py-16 md:py-24">
        <div className="container-site grid gap-14 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <h2 className="font-display text-2xl font-semibold">Каналы связи</h2>
            <p className="mt-2 text-[10px] uppercase tracking-[0.16em] text-[var(--accent)]">
              Макетные данные · isMock
            </p>
            <p className="prose-mute mt-4 text-sm leading-relaxed">{contactsBanner}</p>
            <ul className="mt-6 space-y-4">
              {contacts.map((c) => (
                <li key={c.id}>
                  <p className="text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                    {c.label}
                    {c.isMock ? " · mock" : ""}
                  </p>
                  {c.href && c.href !== "#" ? (
                    <a href={c.href} className="text-lg hover:text-[var(--accent)]">
                      {c.value}
                    </a>
                  ) : (
                    <span className="text-lg">{c.value}</span>
                  )}
                </li>
              ))}
            </ul>
            <p className="prose-mute mt-6 text-sm">
              Юрлицо: <span className="text-[var(--porcelain)]">{legalEntityPlaceholder}</span>
            </p>
            <MagneticLink href="/brief" className="mt-10" variant="ghost">
              Открыть подробный бриф
            </MagneticLink>
          </div>
          <div className="lg:col-span-7">
            <h2 className="font-display mb-8 text-2xl font-semibold">Форма обращения</h2>
            <QuickContactForm source="contact_form" />
          </div>
        </div>
      </section>
    </div>
  );
}
