"use client";

/**
 * Multi-step brief inspired by 21st Wizard Steps (23576) progress + direction transitions.
 * Questionnaire (28569) skipped — proprietary dependency. Custom RHF state + Motion.
 */
import { useMemo, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import {
  budgetRanges,
  contactPrefs,
  featureOptions,
  productTypes,
  projectStages,
  timelineOptions,
} from "@/content/brief";
import { normalizePhone, cn } from "@/lib/utils";
import { submitLead } from "@/lib/leads";
import { MagneticButton } from "@/components/ui/magnetic-button";

type Draft = {
  product?: string;
  stage?: string;
  features: string[];
  idea: string;
  timeline?: string;
  budget?: string;
  name: string;
  email: string;
  phone: string;
  company: string;
  contactPref?: string;
  consent: boolean;
  website: string;
};

const TOTAL = 7;

export function BriefWizard() {
  const reduced = useReducedMotion();
  const [index, setIndex] = useState(0);
  const [direction, setDirection] = useState<1 | -1>(1);
  const [draft, setDraft] = useState<Draft>({
    features: [],
    idea: "",
    name: "",
    email: "",
    phone: "",
    company: "",
    consent: false,
    website: "",
  });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  const progress = ((index + 1) / TOTAL) * 100;

  const go = (next: number) => {
    setDirection(next > index ? 1 : -1);
    setIndex(next);
    setError(null);
  };

  const validateStep = (): boolean => {
    switch (index) {
      case 0:
        if (!draft.product) {
          setError("Выберите тип продукта");
          return false;
        }
        break;
      case 1:
        if (!draft.stage) {
          setError("Выберите стадию проекта");
          return false;
        }
        break;
      case 2:
        if (draft.features.length === 0) {
          setError("Отметьте хотя бы один пункт");
          return false;
        }
        break;
      case 3:
        if (draft.idea.trim().length < 10) {
          setError("Расскажите чуть подробнее — достаточно нескольких предложений");
          return false;
        }
        break;
      case 4:
        if (!draft.timeline) {
          setError("Выберите ориентир по срокам");
          return false;
        }
        break;
      case 5:
        if (!draft.budget) {
          setError("Выберите диапазон или «Нужна оценка»");
          return false;
        }
        break;
      case 6: {
        if (!draft.name.trim()) {
          setError("Укажите имя");
          return false;
        }
        const phone = normalizePhone(draft.phone);
        if (!/^\+[1-9]\d{6,14}$/.test(phone)) {
          setError("Телефон в международном формате, например +79991234567");
          return false;
        }
        if (draft.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(draft.email)) {
          setError("Проверьте email");
          return false;
        }
        if (!draft.contactPref) {
          setError("Выберите предпочтительный способ связи");
          return false;
        }
        if (!draft.consent) {
          setError("Нужно согласие на обработку данных");
          return false;
        }
        break;
      }
    }
    setError(null);
    return true;
  };

  const message = useMemo(() => {
    const label = (list: readonly { id: string; label: string }[], id?: string) =>
      list.find((x) => x.id === id)?.label ?? id ?? "—";
    return [
      "=== Бриф Nexera ===",
      `Тип: ${label(productTypes, draft.product)}`,
      `Стадия: ${label(projectStages, draft.stage)}`,
      `Возможности: ${draft.features
        .map((id) => featureOptions.find((f) => f.id === id)?.label ?? id)
        .join("; ")}`,
      `Идея: ${draft.idea.trim()}`,
      `Сроки: ${label(timelineOptions, draft.timeline)}`,
      `Бюджет: ${label(budgetRanges, draft.budget)}`,
      `Компания: ${draft.company.trim() || "—"}`,
      `Предпочтительная связь: ${label(contactPrefs, draft.contactPref)}`,
    ].join("\n");
  }, [draft]);

  const submit = async () => {
    if (!validateStep() || submitting) return;
    setSubmitting(true);
    setError(null);
    const result = await submitLead({
      name: draft.name.trim(),
      phone: normalizePhone(draft.phone),
      email: draft.email.trim() || undefined,
      message,
      source: "project_brief",
      website: draft.website,
    });
    setSubmitting(false);
    if (result.ok) {
      setDone(true);
      return;
    }
    setError(result.message);
  };

  if (done) {
    return (
      <div className="border border-[var(--line)] bg-[var(--bg-elevated)] p-8 md:p-12">
        <p className="text-xs uppercase tracking-[0.2em] text-[var(--accent)]">Готово</p>
        <h2 className="font-display mt-4 text-3xl font-semibold md:text-4xl">
          Ваша идея уже на шаг ближе к реализации
        </h2>
        <p className="prose-mute mt-4 max-w-xl text-sm leading-relaxed">
          Бриф сохранён. Мы свяжемся удобным для вас способом и уточним детали перед оценкой.
        </p>
      </div>
    );
  }

  return (
    <div className="border border-[var(--line)] bg-[var(--bg-elevated)]">
      <div className="border-b border-[var(--line)] px-6 py-4 md:px-10">
        <div className="flex items-center justify-between gap-4 text-xs uppercase tracking-[0.16em] text-[var(--porcelain-muted)]">
          <span>
            Шаг {index + 1} / {TOTAL}
          </span>
          <span>{Math.round(progress)}%</span>
        </div>
        <div className="mt-3 h-1 overflow-hidden bg-[var(--bg-soft)]">
          <motion.div
            className="h-full bg-[var(--accent)]"
            animate={{ width: `${progress}%` }}
            transition={{ duration: reduced ? 0 : 0.35, ease: [0.23, 1, 0.32, 1] }}
          />
        </div>
      </div>

      <div className="honeypot" aria-hidden="true">
        <label>
          Website
          <input
            type="text"
            tabIndex={-1}
            autoComplete="off"
            value={draft.website}
            onChange={(e) => setDraft((d) => ({ ...d, website: e.target.value }))}
          />
        </label>
      </div>

      <div className="relative min-h-[28rem] overflow-hidden px-6 py-8 md:px-10 md:py-10">
        <AnimatePresence mode="wait" custom={direction}>
          <motion.div
            key={index}
            custom={direction}
            initial={reduced ? false : { opacity: 0, x: direction * 40 }}
            animate={{ opacity: 1, x: 0 }}
            exit={reduced ? undefined : { opacity: 0, x: direction * -40 }}
            transition={{ duration: reduced ? 0 : 0.35, ease: [0.23, 1, 0.32, 1] }}
          >
            {index === 0 && (
              <Step title="Что нужно разработать?">
                <ChoiceGrid
                  options={productTypes}
                  value={draft.product}
                  onChange={(product) => setDraft((d) => ({ ...d, product }))}
                />
              </Step>
            )}
            {index === 1 && (
              <Step title="На какой стадии проект?">
                <ChoiceGrid
                  options={projectStages}
                  value={draft.stage}
                  onChange={(stage) => setDraft((d) => ({ ...d, stage }))}
                />
              </Step>
            )}
            {index === 2 && (
              <Step
                title="Какие возможности необходимы?"
                hint="Можно выбрать несколько вариантов"
              >
                <ChoiceGrid
                  multiple
                  options={featureOptions}
                  values={draft.features}
                  onToggle={(id) =>
                    setDraft((d) => ({
                      ...d,
                      features: d.features.includes(id)
                        ? d.features.filter((x) => x !== id)
                        : [...d.features, id],
                    }))
                  }
                />
              </Step>
            )}
            {index === 3 && (
              <Step
                title="Опишите идею"
                hint="Подробное ТЗ необязательно — достаточно задачи, которую должен решать продукт"
              >
                <textarea
                  className="min-h-40 w-full border border-[var(--line)] bg-[var(--bg)] p-4 text-base"
                  value={draft.idea}
                  onChange={(e) => setDraft((d) => ({ ...d, idea: e.target.value }))}
                  placeholder="Расскажите, какую задачу должен решать ваш будущий продукт"
                />
              </Step>
            )}
            {index === 4 && (
              <Step title="Сроки">
                <ChoiceGrid
                  options={timelineOptions}
                  value={draft.timeline}
                  onChange={(timeline) => setDraft((d) => ({ ...d, timeline }))}
                />
              </Step>
            )}
            {index === 5 && (
              <Step
                title="Ориентир по бюджету"
                hint="Диапазоны настраиваемые и не являются офертой. Можно выбрать «Нужна оценка»."
              >
                <ChoiceGrid
                  options={budgetRanges}
                  value={draft.budget}
                  onChange={(budget) => setDraft((d) => ({ ...d, budget }))}
                />
              </Step>
            )}
            {index === 6 && (
              <Step title="Контакты">
                <div className="grid gap-4 md:grid-cols-2">
                  <label className="block md:col-span-1">
                    <span className="mb-2 block text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                      Имя *
                    </span>
                    <input
                      className="w-full border border-[var(--line)] bg-[var(--bg)] p-3"
                      value={draft.name}
                      onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
                      autoComplete="name"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-2 block text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                      Телефон *
                    </span>
                    <input
                      className="w-full border border-[var(--line)] bg-[var(--bg)] p-3"
                      value={draft.phone}
                      onChange={(e) => setDraft((d) => ({ ...d, phone: e.target.value }))}
                      autoComplete="tel"
                      placeholder="+7 999 123-45-67"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-2 block text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                      Email
                    </span>
                    <input
                      className="w-full border border-[var(--line)] bg-[var(--bg)] p-3"
                      value={draft.email}
                      onChange={(e) => setDraft((d) => ({ ...d, email: e.target.value }))}
                      autoComplete="email"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-2 block text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                      Компания
                    </span>
                    <input
                      className="w-full border border-[var(--line)] bg-[var(--bg)] p-3"
                      value={draft.company}
                      onChange={(e) => setDraft((d) => ({ ...d, company: e.target.value }))}
                      autoComplete="organization"
                    />
                  </label>
                </div>
                <p className="mt-6 text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
                  Предпочтительный способ связи
                </p>
                <div className="mt-3">
                  <ChoiceGrid
                    options={contactPrefs}
                    value={draft.contactPref}
                    onChange={(contactPref) => setDraft((d) => ({ ...d, contactPref }))}
                  />
                </div>
                <label className="mt-6 flex items-start gap-3 text-sm text-[var(--porcelain-muted)]">
                  <input
                    type="checkbox"
                    className="mt-1"
                    checked={draft.consent}
                    onChange={(e) => setDraft((d) => ({ ...d, consent: e.target.checked }))}
                  />
                  <span>
                    Согласен(на) с{" "}
                    <a href="/privacy" className="text-[var(--accent)] hover:underline">
                      политикой обработки данных
                    </a>
                  </span>
                </label>
              </Step>
            )}
          </motion.div>
        </AnimatePresence>

        {error ? <p className="mt-6 text-sm text-[var(--danger)]">{error}</p> : null}

        <div className="mt-10 flex flex-wrap items-center justify-between gap-3">
          <button
            type="button"
            className="text-sm text-[var(--porcelain-muted)] hover:text-[var(--porcelain)] disabled:opacity-40"
            disabled={index === 0 || submitting}
            onClick={() => go(index - 1)}
          >
            ← Назад
          </button>
          {index < TOTAL - 1 ? (
            <MagneticButton
              type="button"
              onClick={() => {
                if (validateStep()) go(index + 1);
              }}
            >
              Далее
            </MagneticButton>
          ) : (
            <MagneticButton type="button" disabled={submitting} onClick={submit}>
              {submitting ? "Отправляем…" : "Отправить бриф"}
            </MagneticButton>
          )}
        </div>
      </div>
    </div>
  );
}

function Step({
  title,
  hint,
  children,
}: {
  title: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <h2 className="font-display text-2xl font-semibold tracking-tight md:text-3xl">{title}</h2>
      {hint ? <p className="prose-mute mt-2 text-sm">{hint}</p> : null}
      <div className="mt-8">{children}</div>
    </div>
  );
}

function ChoiceGrid({
  options,
  value,
  values,
  onChange,
  onToggle,
  multiple,
}: {
  options: readonly { id: string; label: string }[];
  value?: string;
  values?: string[];
  onChange?: (id: string) => void;
  onToggle?: (id: string) => void;
  multiple?: boolean;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {options.map((opt) => {
        const selected = multiple ? values?.includes(opt.id) : value === opt.id;
        return (
          <button
            key={opt.id}
            type="button"
            onClick={() => (multiple ? onToggle?.(opt.id) : onChange?.(opt.id))}
            className={cn(
              "border px-4 py-4 text-left text-sm transition-colors md:text-base",
              selected
                ? "border-[var(--accent)] bg-[var(--accent-dim)] text-[var(--porcelain)]"
                : "border-[var(--line)] hover:border-[var(--porcelain-muted)]",
            )}
            aria-pressed={selected}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
