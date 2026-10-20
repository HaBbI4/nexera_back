export const site = {
  name: "Nexera",
  tagline: "Цифровые продукты, которыми хочется пользоваться",
  description:
    "Nexera разрабатывает корпоративные сайты, интернет-магазины, лендинги, веб- и мобильные приложения, SaaS/MVP, интеграции и сопровождение продуктов.",
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
