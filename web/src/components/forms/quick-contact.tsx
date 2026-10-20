"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { normalizePhone } from "@/lib/utils";
import { submitLead, type LeadSource } from "@/lib/leads";
import { MagneticButton } from "@/components/ui/magnetic-button";

const schema = z.object({
  name: z.string().trim().min(1, "Укажите имя").max(100),
  contact: z.string().trim().min(5, "Укажите телефон или email"),
  message: z.string().trim().min(1, "Кратко опишите задачу").max(5000),
  consent: z.literal(true, {
    errorMap: () => ({ message: "Нужно согласие на обработку данных" }),
  }),
  website: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

function looksLikeEmail(v: string) {
  return v.includes("@");
}

export function QuickContactForm({
  source = "quick_contact",
  className,
}: {
  source?: LeadSource;
  className?: string;
}) {
  const [status, setStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { website: "", consent: undefined as unknown as true },
  });

  const onSubmit = handleSubmit(async (values) => {
    if (status === "loading") return;
    setStatus("loading");
    setError(null);

    const contact = values.contact.trim();
    const phone = looksLikeEmail(contact) ? "+70000000000" : normalizePhone(contact);
    // Backend requires phone E.164. If user gave email-only, we still need a phone —
    // ask them to provide phone in the contact field when not email... Better UX:
    // require phone always; contact field is phone, optional email separate.
    // Spec says: name, contact, message. We'll treat contact as phone if not email,
    // and if email, put in email and ask for phone pattern via message prefix.

    if (looksLikeEmail(contact)) {
      setStatus("error");
      setError("Укажите телефон в международном формате, например +7 999 123-45-67. Email можно добавить в описание.");
      return;
    }
    if (!/^\+[1-9]\d{6,14}$/.test(phone)) {
      setStatus("error");
      setError("Телефон: международный формат с +, например +79991234567.");
      return;
    }

    const result = await submitLead({
      name: values.name,
      phone,
      email: undefined,
      message: values.message,
      source,
      website: values.website ?? "",
    });

    if (result.ok) {
      setStatus("success");
      reset({ name: "", contact: "", message: "", website: "", consent: undefined as unknown as true });
      return;
    }
    setStatus("error");
    setError(result.message);
  });

  if (status === "success") {
    return (
      <div className={className}>
        <p className="font-display text-2xl text-[var(--accent)]">Заявка принята</p>
        <p className="prose-mute mt-3 text-sm leading-relaxed">
          Мы свяжемся с вами по указанному контакту. Можно параллельно заполнить подробный{" "}
          <a href="/brief" className="text-[var(--accent)] underline-offset-4 hover:underline">
            бриф
          </a>
          .
        </p>
        <button
          type="button"
          className="mt-6 text-sm text-[var(--porcelain-muted)] underline-offset-4 hover:underline"
          onClick={() => setStatus("idle")}
        >
          Отправить ещё одну
        </button>
      </div>
    );
  }

  return (
    <form onSubmit={onSubmit} className={className} noValidate>
      <div className="honeypot" aria-hidden="true">
        <label>
          Website
          <input type="text" tabIndex={-1} autoComplete="off" {...register("website")} />
        </label>
      </div>

      <div className="grid gap-5">
        <Field label="Имя" error={errors.name?.message}>
          <input
            className="field"
            autoComplete="name"
            {...register("name")}
          />
        </Field>
        <Field label="Телефон" error={errors.contact?.message}>
          <input
            className="field"
            autoComplete="tel"
            placeholder="+7 999 123-45-67"
            {...register("contact")}
          />
        </Field>
        <Field label="Кратко о задаче" error={errors.message?.message}>
          <textarea className="field min-h-28 resize-y" {...register("message")} />
        </Field>
        <label className="flex items-start gap-3 text-sm text-[var(--porcelain-muted)]">
          <input type="checkbox" className="mt-1" {...register("consent")} />
          <span>
            Согласен(на) с{" "}
            <a href="/privacy" className="text-[var(--accent)] underline-offset-4 hover:underline">
              политикой обработки данных
            </a>
          </span>
        </label>
        {errors.consent ? (
          <p className="text-sm text-[var(--danger)]">{errors.consent.message}</p>
        ) : null}
        {error ? <p className="text-sm text-[var(--danger)]">{error}</p> : null}
        <MagneticButton type="submit" disabled={status === "loading"} className="w-fit">
          {status === "loading" ? "Отправляем…" : "Получить консультацию"}
        </MagneticButton>
      </div>
    </form>
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-2 block text-xs uppercase tracking-[0.14em] text-[var(--porcelain-muted)]">
        {label}
      </span>
      {children}
      {error ? <span className="mt-1 block text-sm text-[var(--danger)]">{error}</span> : null}
    </label>
  );
}
