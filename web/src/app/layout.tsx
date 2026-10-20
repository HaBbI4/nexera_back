import type { Metadata } from "next";
import { Outfit, Syne } from "next/font/google";
import { SiteHeader } from "@/components/layout/site-header";
import { SiteFooter } from "@/components/layout/site-footer";
import { getContactInfo } from "@/lib/api";
import { site } from "@/lib/site";
import "./globals.css";

const syne = Syne({
  subsets: ["latin", "latin-ext"],
  variable: "--font-syne",
  display: "swap",
});

const outfit = Outfit({
  subsets: ["latin", "latin-ext"],
  variable: "--font-outfit",
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: new URL(site.url),
  title: {
    default: `${site.name} — разработка цифровых продуктов`,
    template: `%s · ${site.name}`,
  },
  description: site.description,
  openGraph: {
    type: "website",
    locale: site.locale,
    siteName: site.name,
    title: site.name,
    description: site.description,
    url: site.url,
  },
  alternates: { canonical: site.url },
  robots: { index: true, follow: true },
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const contacts = await getContactInfo();

  return (
    <html lang="ru">
      <body className={`${syne.variable} ${outfit.variable} antialiased`}>
        <SiteHeader />
        <main id="main">{children}</main>
        <SiteFooter contacts={contacts} />
      </body>
    </html>
  );
}
