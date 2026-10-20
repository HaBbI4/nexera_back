import Link from "next/link";
import { MagneticLink } from "@/components/ui/magnetic-button";

export default function NotFound() {
  return (
    <div className="section-pad flex min-h-[70svh] items-center pt-[var(--header-h)]">
      <div className="container-site">
        <p className="font-mono text-sm text-[var(--accent)]">404</p>
        <h1 className="font-display mt-4 text-4xl font-semibold md:text-5xl">Страница не найдена</h1>
        <p className="prose-mute mt-4 max-w-md text-sm leading-relaxed">
          Возможно, раздел ещё не заполнен контентом — например, детальный кейс до появления
          портфолио в API.
        </p>
        <div className="mt-8 flex flex-wrap gap-3">
          <MagneticLink href="/" variant="primary">
            На главную
          </MagneticLink>
          <Link href="/contact" className="inline-flex items-center text-sm text-[var(--accent)] hover:underline">
            Контакты
          </Link>
        </div>
      </div>
    </div>
  );
}
