import type { MetadataRoute } from "next";
import { casesCatalog } from "@/content/cases";
import { servicesCatalog } from "@/content/services";
import { site } from "@/lib/site";

export default function sitemap(): MetadataRoute.Sitemap {
  const staticRoutes = ["", "/services", "/cases", "/about", "/contact", "/brief", "/privacy"].map(
    (path) => ({
      url: `${site.url}${path}`,
      lastModified: new Date(),
      changeFrequency: "weekly" as const,
      priority: path === "" ? 1 : 0.7,
    }),
  );

  const serviceRoutes = servicesCatalog.map((s) => ({
    url: `${site.url}/services/${s.slug}`,
    lastModified: new Date(),
    changeFrequency: "weekly" as const,
    priority: 0.6,
  }));

  // Demo cases: listed but low priority; robots noindex on detail when demo
  const caseRoutes = casesCatalog.map((c) => ({
    url: `${site.url}/cases/${c.slug}`,
    lastModified: new Date(),
    changeFrequency: "monthly" as const,
    priority: c.demo ? 0.3 : 0.6,
  }));

  return [...staticRoutes, ...serviceRoutes, ...caseRoutes];
}
