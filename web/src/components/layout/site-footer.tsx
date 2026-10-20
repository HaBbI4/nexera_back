/** Inspired by 21st Agency Footer (21474) — multi-column sitemap. */
import Link from "next/link";
import { site } from "@/lib/site";
import {
  contactsBanner,
  getDisplayContacts,
  legalEntityPlaceholder,
} from "@/content/contacts";

export function SiteFooter() {
  const year = new Date().getFullYear();
  const contacts = getDisplayContacts().filter((c) => c.type !== "vk");

  return (
    <footer className="border-t border-[var(--line)] bg-[var(--bg-elevated)]">
      <div className="section-pad container-site grid gap-12 py-16 md:grid-cols-12">
        <div className="md:col-span-5">
          <p className="font-display text-3xl font-semibold tracking-tight">
            {site.name}
            <span className="text-[var(--accent)]">.</span>
          </p>
          <p className="prose-mute mt-4 max-w-sm text-sm leading-relaxed">
            {site.description}
          </p>
          <p className="mt-4 text-xs text-[var(--porcelain-muted)]">
            {legalEntityPlaceholder}{" "}
            <span className="text-[var(--accent)]">(заглушка)</span>
          </p>
        </div>

        <div className="md:col-span-3">
          <p className="text-xs uppercase tracking-[0.18em] text-[var(--porcelain-muted)]">
            Навигация
          </p>
          <ul className="mt-4 space-y-2">
            {site.nav.map((item) => (
              <li key={item.href}>
                <Link href={item.href} className="hover:text-[var(--accent)]">
                  {item.label}
                </Link>
              </li>
            ))}
            <li>
              <Link href="/privacy" className="hover:text-[var(--accent)]">
                Политика данных
              </Link>
            </li>
          </ul>
        </div>

        <div className="md:col-span-4">
          <p className="text-xs uppercase tracking-[0.18em] text-[var(--porcelain-muted)]">
            Связь
          </p>
          <p className="mt-2 text-[10px] uppercase tracking-[0.14em] text-[var(--accent)]">
            Макетные контакты
          </p>
          <ul className="mt-4 space-y-2">
            {contacts.map((c) => (
              <li key={c.id} className="text-sm">
                {c.href && c.href !== "#" ? (
                  <a href={c.href} className="hover:text-[var(--accent)]">
                    {c.value}
                  </a>
                ) : (
                  <span>{c.value}</span>
                )}
                {c.isMock ? (
                  <span className="ml-2 text-[10px] uppercase tracking-[0.12em] text-[var(--porcelain-muted)]">
                    mock
                  </span>
                ) : null}
              </li>
            ))}
          </ul>
          <p className="prose-mute mt-4 text-xs leading-relaxed">{contactsBanner}</p>
        </div>
      </div>
      <div className="section-pad container-site flex flex-col gap-2 border-t border-[var(--line)] py-6 text-xs text-[var(--porcelain-muted)] sm:flex-row sm:justify-between">
        <span>
          © {year} {site.name} · {legalEntityPlaceholder}
        </span>
        <span>Разработка цифровых продуктов</span>
      </div>
    </footer>
  );
}
