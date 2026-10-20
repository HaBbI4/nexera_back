import type { CaseContent } from "@/content/cases";
import { cn } from "@/lib/utils";

/** Abstract UI mock — no real product screenshots. */
export function CaseVisual({
  visual,
  accent,
  className,
}: {
  visual: CaseContent["visual"];
  accent: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "relative overflow-hidden border border-[var(--line)] bg-[var(--bg-elevated)]",
        className,
      )}
      style={{
        backgroundImage: `radial-gradient(ellipse 70% 60% at 80% 0%, ${accent}33, transparent 55%)`,
      }}
      aria-hidden
    >
      {visual === "mobile" ? <MobileMock accent={accent} /> : null}
      {visual === "shop" ? <ShopMock accent={accent} /> : null}
      {visual === "corporate" || visual === "landing" ? (
        <SiteMock accent={accent} dense={visual === "landing"} />
      ) : null}
      {visual === "saas" || visual === "ops" ? (
        <AppMock accent={accent} dense={visual === "ops"} />
      ) : null}
    </div>
  );
}

function MobileMock({ accent }: { accent: string }) {
  return (
    <div className="flex h-full min-h-[280px] items-center justify-center p-8">
      <div className="h-64 w-36 border border-[var(--line)] bg-[var(--bg)] p-3 shadow-2xl">
        <div className="mx-auto mb-3 h-1.5 w-10 rounded-full bg-[var(--line)]" />
        <div className="space-y-2">
          <div className="h-16" style={{ background: `${accent}44` }} />
          <div className="h-2 w-3/4 bg-[var(--line)]" />
          <div className="h-2 w-1/2 bg-[var(--line)]" />
          <div className="mt-4 grid gap-2">
            <div className="h-8 border border-[var(--line)]" />
            <div className="h-8 border border-[var(--line)]" style={{ borderColor: accent }} />
          </div>
        </div>
      </div>
    </div>
  );
}

function ShopMock({ accent }: { accent: string }) {
  return (
    <div className="grid min-h-[280px] grid-cols-3 gap-3 p-6">
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="border border-[var(--line)] bg-[var(--bg)] p-2">
          <div
            className="mb-2 aspect-square"
            style={{ background: i % 2 === 0 ? `${accent}55` : `${accent}22` }}
          />
          <div className="h-1.5 w-3/4 bg-[var(--line)]" />
          <div className="mt-1 h-1.5 w-1/3 bg-[var(--line)]" />
        </div>
      ))}
    </div>
  );
}

function SiteMock({ accent, dense }: { accent: string; dense?: boolean }) {
  return (
    <div className="flex min-h-[280px] flex-col gap-3 p-6">
      <div className="flex items-center justify-between border-b border-[var(--line)] pb-3">
        <div className="h-2 w-16" style={{ background: accent }} />
        <div className="flex gap-2">
          <div className="h-1.5 w-8 bg-[var(--line)]" />
          <div className="h-1.5 w-8 bg-[var(--line)]" />
        </div>
      </div>
      <div className="h-20 border border-[var(--line)]" style={{ background: `${accent}22` }} />
      <div className={cn("grid gap-3", dense ? "grid-cols-1" : "grid-cols-3")}>
        {(dense ? [0, 1] : [0, 1, 2]).map((i) => (
          <div key={i} className="h-16 border border-[var(--line)] bg-[var(--bg)]" />
        ))}
      </div>
    </div>
  );
}

function AppMock({ accent, dense }: { accent: string; dense?: boolean }) {
  return (
    <div className="grid min-h-[280px] grid-cols-12 gap-3 p-6">
      <div className="col-span-3 space-y-2 border border-[var(--line)] bg-[var(--bg)] p-3">
        {[0, 1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-2"
            style={{ background: i === 1 ? accent : "var(--line)", width: `${70 - i * 10}%` }}
          />
        ))}
      </div>
      <div className="col-span-9 space-y-2">
        <div className="flex gap-2">
          <div className="h-8 flex-1 border border-[var(--line)]" />
          <div className="h-8 w-20" style={{ background: accent }} />
        </div>
        {(dense ? [0, 1, 2, 3, 4] : [0, 1, 2]).map((i) => (
          <div key={i} className="flex h-8 items-center gap-3 border border-[var(--line)] bg-[var(--bg)] px-3">
            <div className="h-2 w-2 rounded-full" style={{ background: accent }} />
            <div className="h-1.5 flex-1 bg-[var(--line)]" />
            <div className="h-1.5 w-10 bg-[var(--line)]" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function ConceptBadge({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center border border-[color-mix(in_oklab,currentColor_35%,transparent)] px-2 py-0.5 text-[10px] font-medium uppercase tracking-[0.16em]",
        className,
      )}
    >
      Концепт
    </span>
  );
}
