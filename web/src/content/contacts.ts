/**
 * Single source for public contacts on the marketing site.
 * Prefer this file until admin `contact_info` is filled; then swap or merge.
 *
 * All entries below are MOCK — replace before production.
 * Do not use real third-party numbers/emails (incl. other studios).
 */
export type SiteContact = {
  id: string;
  type: "phone" | "email" | "telegram" | "whatsapp" | "vk" | "city";
  label: string;
  value: string;
  href?: string;
  /** Mock placeholder — easy to filter; excluded from structured data. */
  isMock: boolean;
};

export const siteContacts: SiteContact[] = [
  {
    id: "email",
    type: "email",
    label: "Email",
    value: "hello@example.com",
    href: "mailto:hello@example.com",
    isMock: true,
  },
  {
    id: "phone",
    type: "phone",
    label: "Телефон",
    value: "+7 (000) 000-00-00",
    href: "tel:+70000000000",
    isMock: true,
  },
  {
    id: "telegram",
    type: "telegram",
    label: "Telegram",
    value: "@your_studio",
    href: "https://t.me/your_studio",
    isMock: true,
  },
  {
    id: "city",
    type: "city",
    label: "Город",
    value: "Москва",
    isMock: true,
  },
  // Socials: href "#" — hide from structured data; optional UI links marked mock
  {
    id: "vk",
    type: "vk",
    label: "VK",
    value: "VK (заглушка)",
    href: "#",
    isMock: true,
  },
];

export const legalEntityPlaceholder = "ООО «Название»";

export const contactsBanner =
  "Контакты отмечены как макетные (isMock). Замените в `content/contacts.ts` или опубликуйте реальные через API contact_info.";

export function getDisplayContacts(): SiteContact[] {
  return siteContacts.filter((c) => c.type !== "vk" || (c.href && c.href !== "#"));
}

/** Contacts safe for Organization JSON-LD — excludes isMock. */
export function getStructuredDataContacts(): SiteContact[] {
  return siteContacts.filter((c) => !c.isMock);
}
