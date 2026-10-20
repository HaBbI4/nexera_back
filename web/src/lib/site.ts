export const site = {
  name: "Nexera",
  tagline: "Цифровые продукты, которыми хочется пользоваться",
  description:
    "Nexera проектирует и разрабатывает веб-приложения, мобильные продукты и SaaS-системы: от идеи и UX до инженерии и развития после запуска.",
  url: process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000",
  locale: "ru_RU",
  nav: [
    { href: "/services", label: "Услуги" },
    { href: "/cases", label: "Кейсы" },
    { href: "/about", label: "О студии" },
    { href: "/brief", label: "Бриф" },
    { href: "/contact", label: "Контакты" },
  ],
} as const;

export type NavItem = (typeof site.nav)[number];
