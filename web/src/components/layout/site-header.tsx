"use client";

/** Nav / mobile sheet inspired by 21st Agency Hero sticky header (28280) + Motion Nav (21349). */
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { site } from "@/lib/site";
import { cn } from "@/lib/utils";
import { MagneticLink } from "@/components/ui/magnetic-button";

export function SiteHeader() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const reduced = useReducedMotion();

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 12);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  useEffect(() => {
    document.body.style.overflow = open ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  return (
    <header
      className={cn(
        "fixed inset-x-0 top-0 z-50 transition-[background,border-color,backdrop-filter] duration-300",
        scrolled || open
          ? "border-b border-[var(--line)] bg-[color-mix(in_oklab,var(--bg)_88%,transparent)] backdrop-blur-md"
          : "border-b border-transparent bg-transparent",
      )}
    >
      <div className="section-pad container-site flex h-[var(--header-h)] items-center justify-between gap-6">
        <Link
          href="/"
          className="font-display text-xl font-semibold tracking-tight md:text-2xl"
          aria-label={`${site.name} — на главную`}
        >
          {site.name}
          <span className="text-[var(--accent)]">.</span>
        </Link>

        <nav className="hidden items-center gap-8 lg:flex" aria-label="Основная навигация">
          {site.nav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "text-sm tracking-wide text-[var(--porcelain-muted)] transition-colors hover:text-[var(--porcelain)]",
                pathname === item.href || pathname.startsWith(item.href + "/")
                  ? "text-[var(--porcelain)]"
                  : "",
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-3 lg:flex">
          <MagneticLink href="/brief" variant="primary" className="!px-5 !py-2.5 text-xs uppercase tracking-[0.14em]">
            Обсудить проект
          </MagneticLink>
        </div>

        <button
          type="button"
          className="inline-flex h-10 w-10 items-center justify-center border border-[var(--line)] lg:hidden"
          aria-expanded={open}
          aria-controls="mobile-nav"
          aria-label={open ? "Закрыть меню" : "Открыть меню"}
          onClick={() => setOpen((v) => !v)}
        >
          <span className="sr-only">Меню</span>
          <span aria-hidden className="relative block h-3 w-5">
            <span
              className={cn(
                "absolute left-0 top-0 h-px w-full bg-[var(--porcelain)] transition-transform",
                open && "top-1.5 rotate-45",
              )}
            />
            <span
              className={cn(
                "absolute left-0 top-1.5 h-px w-full bg-[var(--porcelain)] transition-opacity",
                open && "opacity-0",
              )}
            />
            <span
              className={cn(
                "absolute left-0 top-3 h-px w-full bg-[var(--porcelain)] transition-transform",
                open && "top-1.5 -rotate-45",
              )}
            />
          </span>
        </button>
      </div>

      <AnimatePresence>
        {open ? (
          <motion.div
            id="mobile-nav"
            initial={reduced ? false : { height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={reduced ? undefined : { height: 0, opacity: 0 }}
            transition={{ duration: reduced ? 0 : 0.35, ease: [0.23, 1, 0.32, 1] }}
            className="overflow-hidden border-t border-[var(--line)] bg-[var(--bg)] lg:hidden"
          >
            <nav className="section-pad flex flex-col gap-1 py-6" aria-label="Мобильная навигация">
              {site.nav.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="font-display py-3 text-2xl font-semibold tracking-tight"
                >
                  {item.label}
                </Link>
              ))}
              <Link
                href="/brief"
                className="mt-4 inline-flex items-center justify-center bg-[var(--accent)] px-5 py-3 text-sm font-medium text-[var(--ink)]"
              >
                Обсудить проект
              </Link>
            </nav>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </header>
  );
}
