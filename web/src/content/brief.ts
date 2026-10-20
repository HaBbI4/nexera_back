/**
 * Brief wizard config.
 * `budgetRanges` — EDITABLE placeholders, not a public price list / offer.
 */
export const budgetRanges = [
  { id: "need_estimate", label: "Нужна оценка" },
  { id: "undecided", label: "Пока не определился" },
  { id: "range_1", label: "До 300 000 ₽" },
  { id: "range_2", label: "300 000 – 800 000 ₽" },
  { id: "range_3", label: "800 000 – 2 000 000 ₽" },
  { id: "range_4", label: "Более 2 000 000 ₽" },
] as const;

/** Hint shown under budget step — keep visible so ranges aren't read as fixed quotes. */
export const budgetStepHint =
  "Диапазоны хранятся в конфиге `brief.ts` и нужны только как ориентир для разговора. Это не прайс и не оферта — точную оценку дадим после брифа.";

export const productTypes = [
  { id: "corporate", label: "Корпоративный сайт" },
  { id: "shop", label: "Интернет-магазин" },
  { id: "landing", label: "Лендинг" },
  { id: "web", label: "Веб-приложение" },
  { id: "mobile", label: "Мобильное приложение" },
  { id: "saas", label: "SaaS / MVP" },
  { id: "integrations", label: "Интеграции и автоматизация" },
  { id: "design", label: "UX/UI дизайн" },
  { id: "support", label: "Поддержка существующего продукта" },
  { id: "unsure", label: "Пока не определился" },
] as const;

export const productStepHint =
  "Если сомневаетесь между лендингом и полноценным сайтом — выберите ближайший вариант, уточним на созвоне.";

export const projectStages = [
  { id: "idea", label: "Есть только идея" },
  { id: "spec", label: "Есть описание или ТЗ" },
  { id: "design", label: "Есть дизайн или прототип" },
  { id: "existing", label: "Есть существующий продукт" },
  { id: "greenfield", label: "Нужна полная разработка с нуля" },
] as const;

export const stageStepHint =
  "Даже сырая идея подходит: подробное ТЗ на старте не обязательно.";

export const featureOptions = [
  { id: "auth", label: "Регистрация и личные кабинеты" },
  { id: "payments", label: "Платежи и подписки" },
  { id: "catalog", label: "Каталог или магазин" },
  { id: "admin", label: "Административная панель" },
  { id: "chat", label: "Чаты и уведомления" },
  { id: "integrations", label: "Интеграции с CRM / API" },
  { id: "analytics", label: "Аналитика и отчётность" },
  { id: "seo", label: "SEO и контентные разделы" },
  { id: "ai", label: "AI-функциональность" },
  { id: "geo", label: "Геолокация" },
  { id: "other", label: "Другое" },
  { id: "consult", label: "Нужна консультация по составу" },
] as const;

export const featuresStepHint =
  "Отметьте всё, что кажется нужным — лишнее отсечем при оценке скоупа.";

export const ideaStepHint =
  "Расскажите задачу своими словами. Техническое задание прикладывать не обязательно — хватит целей, аудитории и ограничений.";

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
