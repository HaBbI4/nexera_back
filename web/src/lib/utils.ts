import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** Normalize phone for API (E.164-friendly); keeps leading +. */
export function normalizePhone(value: string): string {
  const trimmed = value.trim();
  const digits = trimmed.replace(/[^\d+]/g, "");
  if (digits.startsWith("+")) {
    return "+" + digits.slice(1).replace(/\D/g, "");
  }
  const only = digits.replace(/\D/g, "");
  if (only.startsWith("8") && only.length === 11) {
    return "+7" + only.slice(1);
  }
  if (only.startsWith("7") && only.length === 11) {
    return "+" + only;
  }
  return digits.startsWith("+") ? digits : only ? `+${only}` : trimmed;
}
