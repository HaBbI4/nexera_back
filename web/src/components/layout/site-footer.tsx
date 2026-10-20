/** Inspired by 21st Agency Footer (id 21474) — multi-column sitemap, real contacts only. */
import Link from "next/link";
import { site } from "@/lib/site";
import type { ContactInfo } from "@/lib/api";

function contactHref(item: ContactInfo): string | null {
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

export function SiteFooter({ contacts }: { contacts: ContactInfo[] }) {
  const year = new Date().getFullYear();
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
          {contacts.length === 0 ? (
            <p className="prose-mute mt-4 text-sm">
              Контакты появятся после публикации в админ-панели. Пока можно{" "}
              <Link href="/brief" className="text-[var(--accent)] underline-offset-4 hover:underline">
                оставить бриф
              </Link>
              .
            </p>
          ) : (
            <ul className="mt-4 space-y-2">
              {contacts.map((c) => {
                const href = contactHref(c);
                const label = c.label || c.value || c.type;
                return (
                  <li key={c.id}>
                    {href ? (
                      <a href={href} className="hover:text-[var(--accent)]">
                        {label}
                      </a>
                    ) : (
                      <span>{label}</span>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </div>
      <div className="section-pad container-site flex flex-col gap-2 border-t border-[var(--line)] py-6 text-xs text-[var(--porcelain-muted)] sm:flex-row sm:justify-between">
        <span>
          © {year} {site.name}
        </span>
        <span>Разработка цифровых продуктов</span>
      </div>
    </footer>
  );
}
