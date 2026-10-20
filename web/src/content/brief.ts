/** Budget ranges are configurable placeholders — not claimed company pricing. */
export const budgetRanges = [
  { id: "need_estimate", label: "Нужна оценка" },
  { id: "undecided", label: "Пока не определился" },
  { id: "range_1", label: "До 500 000 ₽" },
  { id: "range_2", label: "500 000 – 1 500 000 ₽" },
  { id: "range_3", label: "1 500 000 – 3 000 000 ₽" },
  { id: "range_4", label: "Более 3 000 000 ₽" },
] as const;

export const productTypes = [
  { id: "mobile", label: "Мобильное приложение" },
  { id: "web", label: "Веб-приложение" },
  { id: "saas", label: "SaaS-платформа" },
  { id: "mvp", label: "MVP для стартапа" },
  { id: "enterprise", label: "Корпоративная система" },
  { id: "ai", label: "AI-продукт или автоматизация" },
  { id: "rewrite", label: "Доработать существующий проект" },
  { id: "unsure", label: "Пока не определился" },
] as const;

export const projectStages = [
  { id: "idea", label: "Есть только идея" },
  { id: "spec", label: "Есть описание или ТЗ" },
  { id: "design", label: "Есть дизайн или прототип" },
  { id: "existing", label: "Есть существующий продукт" },
  { id: "greenfield", label: "Нужна полная разработка с нуля" },
] as const;

export const featureOptions = [
  { id: "auth", label: "Регистрация и личные кабинеты" },
  { id: "payments", label: "Платежи и подписки" },
  { id: "catalog", label: "Каталог или маркетплейс" },
  { id: "admin", label: "Административная панель" },
  { id: "chat", label: "Чаты и уведомления" },
  { id: "integrations", label: "Интеграции с внешними API" },
  { id: "analytics", label: "Аналитика и отчётность" },
  { id: "ai", label: "AI-функциональность" },
  { id: "geo", label: "Геолокация" },
  { id: "other", label: "Другое" },
  { id: "consult", label: "Нужна консультация" },
] as const;

export const timelineOptions = [
  { id: "asap", label: "Как можно быстрее" },
  { id: "1_3", label: "В течение 1–3 месяцев" },
  { id: "3_6", label: "В течение 3–6 месяцев" },
  { id: "6_plus", label: "Более 6 месяцев" },
  { id: "unknown", label: "Сроки пока не определены" },
] as const;

export const contactPrefs = [
  { id: "phone", label: "Телефон" },
  { id: "email", label: "Email" },
  { id: "telegram", label: "Telegram" },
  { id: "whatsapp", label: "WhatsApp" },
] as const;
