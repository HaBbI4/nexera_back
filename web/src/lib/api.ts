import { site } from "./site";

/** Server-side base URL for backend (Docker/local). Browser uses same-origin rewrites. */
function serverApiBase(): string {
  return (
    process.env.API_INTERNAL_URL?.replace(/\/$/, "") ||
    process.env.NEXT_PUBLIC_API_BASE?.replace(/\/$/, "") ||
    "http://localhost:8080"
  );
}

export type Service = {
  id: number;
  name: string;
  description?: string | null;
  price?: number | null;
  position: number;
  is_published: boolean;
};

export type FAQ = {
  id: number;
  question: string;
  answer: string;
  position: number;
};

export type PublicReview = {
  id: number;
  name: string;
  rating: number;
  text: string;
  created_at: string;
};

export type ReviewSummary = {
  count: number;
  average_rating: number;
};

export type ContactInfo = {
  id: number;
  type: "phone" | "email" | "telegram" | "whatsapp" | "vk";
  value?: string | null;
  label?: string | null;
  position: number;
};

export type Banner = {
  id: number;
  page_id: number;
  title: string;
  subtitle?: string | null;
  description?: string | null;
  link_title?: string | null;
  link_url?: string | null;
  image_url: string;
  position: number;
};

export type Page = {
  id: number;
  slug: string;
  name: string;
  banners?: Banner[];
};

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T | null> {
  const url = `${serverApiBase()}${path}`;
  try {
    const res = await fetch(url, {
      ...init,
      next: { revalidate: 60 },
      headers: {
        Accept: "application/json",
        ...(init?.headers ?? {}),
      },
    });
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null;
  }
}

export async function getServices(): Promise<Service[]> {
  return (await apiFetch<Service[]>("/api/v1/services?limit=100")) ?? [];
}

export async function getService(id: number): Promise<Service | null> {
  return apiFetch<Service>(`/api/v1/services/${id}`);
}

export async function getFaqs(): Promise<FAQ[]> {
  return (await apiFetch<FAQ[]>("/api/v1/faq?limit=100")) ?? [];
}

export async function getReviews(): Promise<PublicReview[]> {
  return (await apiFetch<PublicReview[]>("/api/v1/reviews?limit=50")) ?? [];
}

export async function getReviewSummary(): Promise<ReviewSummary | null> {
  return apiFetch<ReviewSummary>("/api/v1/reviews/summary");
}

export async function getContactInfo(): Promise<ContactInfo[]> {
  return (await apiFetch<ContactInfo[]>("/api/v1/contact-info?limit=50")) ?? [];
}

export async function getPages(): Promise<Page[]> {
  return (await apiFetch<Page[]>("/api/v1/pages")) ?? [];
}

export async function getBanners(pageId?: number): Promise<Banner[]> {
  const q = pageId ? `?page_id=${pageId}&limit=20` : "?limit=20";
  return (await apiFetch<Banner[]>(`/api/v1/banners${q}`)) ?? [];
}

export function mediaUrl(path: string): string {
  if (!path) return "";
  if (path.startsWith("http")) return path;
  const base =
    process.env.NEXT_PUBLIC_API_BASE?.replace(/\/$/, "") ||
    (typeof window === "undefined" ? serverApiBase() : site.url);
  return `${base}${path.startsWith("/") ? path : `/${path}`}`;
}

export function serviceSlug(service: Pick<Service, "id" | "name">): string {
  const base = service.name
    .toLowerCase()
    .replace(/ё/g, "е")
    .replace(/[^a-zа-я0-9]+/gi, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 60);
  return `${base || "service"}-${service.id}`;
}

export function parseServiceIdFromSlug(slug: string): number | null {
  const m = slug.match(/-(\d+)$/);
  if (!m) return null;
  const id = Number(m[1]);
  return Number.isFinite(id) ? id : null;
}
