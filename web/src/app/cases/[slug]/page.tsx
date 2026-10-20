import type { Metadata } from "next";
import { notFound } from "next/navigation";

type Props = { params: Promise<{ slug: string }> };

export const metadata: Metadata = {
  title: "Кейс",
  robots: { index: false, follow: true },
};

/** Until portfolio API exists, any slug is 404 — never invent a case. */
export default async function CaseDetailPage({ params }: Props) {
  await params;
  notFound();
}
