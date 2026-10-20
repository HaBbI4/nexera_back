import type { MetadataRoute } from "next";
import { getServices, serviceSlug } from "@/lib/api";
import { site } from "@/lib/site";

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const services = await getServices();
  const staticRoutes = ["", "/services", "/cases", "/about", "/contact", "/brief", "/privacy"].map(
    (path) => ({
      url: `${site.url}${path}`,
      lastModified: new Date(),
      changeFrequency: "weekly" as const,
      priority: path === "" ? 1 : 0.7,
    }),
  );

  const serviceRoutes = services.map((s) => ({
    url: `${site.url}/services/${serviceSlug(s)}`,
    lastModified: new Date(),
    changeFrequency: "weekly" as const,
    priority: 0.6,
  }));

  return [...staticRoutes, ...serviceRoutes];
}
