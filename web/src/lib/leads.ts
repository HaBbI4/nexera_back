export type LeadSource = "quick_contact" | "project_brief" | "case_cta" | "contact_form";

export type CreateLeadPayload = {
  name: string;
  phone: string;
  email?: string;
  message?: string;
  service_id?: number;
  source: LeadSource | string;
  website?: string;
};

export type LeadResult =
  | { ok: true }
  | { ok: false; status: number; message: string; retryAfter?: number };

/** Browser-only: POST lead via same-origin rewrite to backend. */
export async function submitLead(payload: CreateLeadPayload): Promise<LeadResult> {
  try {
    const res = await fetch("/api/v1/leads", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({
        name: payload.name,
        phone: payload.phone,
        email: payload.email || undefined,
        message: payload.message || undefined,
        service_id: payload.service_id,
        source: payload.source,
        website: payload.website ?? "",
      }),
    });

    if (res.status === 201) {
      return { ok: true };
    }

    let message = "Не удалось отправить заявку. Попробуйте ещё раз.";
    if (res.status === 400) {
      message = "Проверьте корректность заполненных полей.";
    } else if (res.status === 429) {
      message = "Слишком много попыток. Подождите немного и попробуйте снова.";
    } else if (res.status >= 500) {
      message = "Сервер временно недоступен. Попробуйте позже.";
    }

    const retryAfterHeader = res.headers.get("Retry-After");
    const retryAfter = retryAfterHeader ? Number(retryAfterHeader) : undefined;

    try {
      const body = (await res.json()) as { message?: string };
      if (body?.message && res.status === 400) {
        // Keep RU UX copy; technical EN message stays in network tab.
      }
    } catch {
      /* ignore */
    }

    return {
      ok: false,
      status: res.status,
      message,
      retryAfter: Number.isFinite(retryAfter) ? retryAfter : undefined,
    };
  } catch {
    return {
      ok: false,
      status: 0,
      message: "Нет соединения с сервером. Проверьте сеть и попробуйте снова.",
    };
  }
}
