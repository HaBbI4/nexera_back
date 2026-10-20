import type { Metadata } from "next";
import { QuickContactForm } from "@/components/forms/quick-contact";
import { getContactInfo, type ContactInfo } from "@/lib/api";
import { MagneticLink } from "@/components/ui/magnetic-button";

export const metadata: Metadata = {
  title: "Контакты",
  description: "Связаться со студией Nexera — форма и опубликованные каналы связи.",
};

function hrefFor(item: ContactInfo): string | null {
  if (!item.value) return null;
  switch (item.type) {
    case "phone":
      return `tel:${item.value.replace(/\s/g, "")}`;
    case "email":
      return `mailto:${item.value}`;
    case "telegram":
      return item.value.startsWith("http")
        ? item.value
        : `https://t.me/${item.value.replace(/^@/, "")}`;
    case "whatsapp":
      return item.value.startsWith("http")
        ? item.value
        : `https://wa.me/${item.value.replace(/\D/g, "")}`;
    case "vk":
      return item.value.startsWith("http") ? item.value : `https://vk.com/${item.value}`;
    default:
      return item.value;
  }
}

export default async function ContactPage() {
  const contacts = await getContactInfo();

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
            {contacts.length === 0 ? (
              <p className="prose-mute mt-4 text-sm leading-relaxed">
                Публичные контакты ещё не опубликованы в API (`contact_info`). Заполните их в
                админке или см. CONTENT_REQUIRED.md. Пока удобнее оставить заявку формой.
              </p>
            ) : (
              <ul className="mt-6 space-y-3">
                {contacts.map((c) => {
                  const href = hrefFor(c);
                  const label = c.label || c.value || c.type;
                  return (
                    <li key={c.id}>
                      {href ? (
                        <a href={href} className="text-lg hover:text-[var(--accent)]">
                          {label}
                        </a>
                      ) : (
                        <span className="text-lg">{label}</span>
                      )}
                      <span className="ml-2 text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                        {c.type}
                      </span>
                    </li>
                  );
                })}
              </ul>
            )}
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
