/**
 * Portfolio entries for the marketing site.
 * `demo: true` = concept showcase only — replace with real cases when ready.
 * No client names, no fabricated metrics/results.
 */
export type CaseContent = {
  slug: string;
  /** When true, UI shows «Концепт» badge; easy to filter out later. */
  demo: true | false;
  title: string;
  productType: string;
  industry: string;
  summary: string;
  context: string;
  challenge: string;
  solution: string;
  uxNotes: string;
  techNotes: string;
  /** Goals / approach — not claimed outcomes. */
  goals: string[];
  technologies: string[];
  /** Abstract visual theme for CSS mock UI */
  visual: "shop" | "corporate" | "mobile" | "saas" | "landing" | "ops";
  accent: string;
};

export const casesCatalog: CaseContent[] = [
  {
    slug: "concept-shop-glow",
    demo: true,
    title: "Витрина нишевого e-commerce",
    productType: "Интернет-магазин",
    industry: "Ритейл / D2C",
    summary:
      "Концепт магазина с акцентом на каталог, чекаут и повторные покупки — без привязки к реальному бренду.",
    context:
      "Типичная задача: офлайн или соцсети уже продают, нужен собственный канал с каталогом, оплатой и понятной админкой.",
    challenge:
      "Собрать быстрый и понятный путь «карточка → корзина → оплата», не перегружая первый визит фильтрами и акциями.",
    solution:
      "Спроектировали информационную архитектуру каталога, упрощённый чекаут, статусы заказа и админ-панель для контента и остатков.",
    uxNotes:
      "Крупные превью, липкая корзина на мобиле, явные шаги оформления, пустые состояния вместо «молчаливых» ошибок.",
    techNotes:
      "Витрина на современном фронтенде, API каталога и заказов, интеграции оплаты и доставки как отдельные адаптеры.",
    goals: [
      "Сократить трение на чекауте",
      "Дать команде простой способ вести каталог",
      "Заложить аналитику воронки покупки",
    ],
    technologies: ["Next.js", "Node API", "PostgreSQL", "Платёжный шлюз", "CDN"],
    visual: "shop",
    accent: "#2ee6a6",
  },
  {
    slug: "concept-b2b-portal",
    demo: true,
    title: "Корпоративный портал услуг",
    productType: "Корпоративный сайт",
    industry: "B2B / услуги",
    summary:
      "Концепт многостраничного сайта: экспертиза, кейсы-как-структура, прозрачный путь к брифу.",
    context:
      "Компании услуг часто звучат одинаково. Нужна ясная подача компетенций и удобный вход в диалог.",
    challenge:
      "Связать услуги, подход и доказательства доверия так, чтобы посетитель не потерялся в меню из 20 пунктов.",
    solution:
      "Редакционная структура разделов, сильный hero, услуги как narrative, процесс и FAQ, единая CTA-линия к брифу.",
    uxNotes:
      "Асимметричные секции, якорная навигация, крупные заголовки, формы без лишних полей на первом касании.",
    techNotes:
      "SSG/SSR для SEO, централизованный контент, формы в backend заявок, sitemap и метаданные по разделам.",
    goals: [
      "Ускорить понимание «чем вы полезны»",
      "Собрать квалифицированные заявки",
      "Упростить обновление контента",
    ],
    technologies: ["Next.js", "Tailwind", "CMS/API", "SEO schema"],
    visual: "corporate",
    accent: "#7dd3fc",
  },
  {
    slug: "concept-field-app",
    demo: true,
    title: "Мобильное приложение для полевых задач",
    productType: "Мобильное приложение",
    industry: "Операции / сервисы",
    summary:
      "Концепт приложения для сотрудников вне офиса: задачи, статусы, фото и офлайн-устойчивость.",
    context:
      "Полевые команды теряют время на чаты и таблицы. Нужен лёгкий клиент под телефон.",
    challenge:
      "Сделать интерфейс, который работает одной рукой, с понятными статусами и минимумом обязательных полей.",
    solution:
      "Сценарии смены статусов, загрузка медиа, синхронизация при сети, push о новых назначениях, кабинет координатора в вебе.",
    uxNotes:
      "Крупные зоны нажатия, цветовые статусы, короткие формы, режим плохого интернета.",
    techNotes:
      "Кроссплатформенный клиент, REST/WebSocket API, очередь синхронизации, роли и аудит действий.",
    goals: [
      "Убрать потерю статусов между сменами",
      "Ускорить закрытие типовых задач",
      "Дать координатору живую картину дня",
    ],
    technologies: ["React Native", "Go API", "PostgreSQL", "Push", "S3"],
    visual: "mobile",
    accent: "#fbbf24",
  },
  {
    slug: "concept-saas-ledger",
    demo: true,
    title: "SaaS MVP учёта и подписок",
    productType: "SaaS / MVP",
    industry: "B2B software",
    summary:
      "Концепт первой версии SaaS: онбординг, роли, биллинг и дашборд без лишней сложности.",
    context:
      "Команде нужно проверить спрос на подписочный продукт и собрать обратную связь от первых пользователей.",
    challenge:
      "Не расползтись по фичам: оставить ядро, которое доказывает ценность за первую сессию.",
    solution:
      "Узкий MVP: регистрация, workspace, базовые сущности, тарифы, админка и события для продуктовой аналитики.",
    uxNotes:
      "Пустой онбординг с подсказками, прогресс настройки, прозрачные лимиты тарифа.",
    techNotes:
      "Мультитенантность на уровне данных, биллинг через провайдера, feature flags для следующих итераций.",
    goals: [
      "Запустить первую оплачиваемую когорту",
      "Измерить activation и retention",
      "Сохранить архитектуру под рост",
    ],
    technologies: ["Next.js", "Go", "PostgreSQL", "Stripe/CloudPayments", "Redis"],
    visual: "saas",
    accent: "#c4b5fd",
  },
  {
    slug: "concept-landing-pulse",
    demo: true,
    title: "Лендинг запуска продукта",
    productType: "Лендинг",
    industry: "Продуктовый маркетинг",
    summary:
      "Концепт посадочной страницы под рекламный трафик: оффер, доказательства подхода, заявка.",
    context:
      "Нужно быстро проверить интерес к предложению до большой разработки.",
    challenge:
      "Удержать внимание с мобильного экрана и довести до одной целевой формы.",
    solution:
      "Короткий narrative: проблема → решение → как работаем → FAQ → бриф. Сильный первый экран и повторные CTA.",
    uxNotes:
      "Минимум отвлечений, крупные CTA, FAQ снимает типовые сомнения без ухода со страницы.",
    techNotes:
      "Лёгкий фронт, цели аналитики, A/B-готовая структура блоков, защита формы от спама.",
    goals: [
      "Собрать заявки с платного трафика",
      "Понять, какие блоки читают",
      "Подготовить почву для полной версии сайта",
    ],
    technologies: ["Next.js", "Motion", "Metrika/GA", "Lead API"],
    visual: "landing",
    accent: "#fb7185",
  },
  {
    slug: "concept-ops-console",
    demo: true,
    title: "Операционная веб-консоль",
    productType: "Веб-приложение",
    industry: "Внутренние системы",
    summary:
      "Концепт внутренней панели: очереди задач, фильтры, роли и интеграции с внешними API.",
    context:
      "Операции растут быстрее, чем Excel. Нужен рабочий инструмент для ежедневных процессов.",
    challenge:
      "Сделать плотный интерфейс читаемым: таблицы, статусы, действия без ощущения «простыни».",
    solution:
      "IA вокруг очередей и сущностей, боковые фильтры, массовые действия, журнал событий и права доступа.",
    uxNotes:
      "Плотная, но ритмичная сетка, keyboard-friendly действия, явные состояния загрузки и ошибок.",
    techNotes:
      "SPA/SSR гибрид, пагинация и поиск на API, аудиты, фоновые jobs для тяжёлых операций.",
    goals: [
      "Сократить время на рутинные операции",
      "Снизить ошибки ручного ввода",
      "Подключить смежные системы без копипаста",
    ],
    technologies: ["React", "Go API", "PostgreSQL", "Redis", "OpenAPI"],
    visual: "ops",
    accent: "#34d399",
  },
];

export function getCaseBySlug(slug: string): CaseContent | undefined {
  return casesCatalog.find((c) => c.slug === slug);
}

export function getPublishedCases(): CaseContent[] {
  return casesCatalog;
}

/** Drop all demo/concept cases when real portfolio arrives. */
export function getRealCasesOnly(): CaseContent[] {
  return casesCatalog.filter((c) => !c.demo);
}
