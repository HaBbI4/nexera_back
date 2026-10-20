import { getStructuredDataContacts, legalEntityPlaceholder } from "@/content/contacts";
import { site } from "@/lib/site";

/**
 * Organization schema. Mock contacts (isMock) are intentionally omitted —
 * do not publish placeholder phone/email into structured data.
 */
export function OrganizationJsonLd() {
  const real = getStructuredDataContacts();
  const email = real.find((c) => c.type === "email")?.value;
  const telephone = real.find((c) => c.type === "phone")?.value;
  const city = real.find((c) => c.type === "city")?.value;

  const data: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: site.name,
    url: site.url,
    description: site.description,
    // Legal name stays placeholder until confirmed — not a real entity claim
    legalName: legalEntityPlaceholder,
  };

  if (email) data.email = email;
  if (telephone) data.telephone = telephone;
  if (city) {
    data.address = {
      "@type": "PostalAddress",
      addressLocality: city,
      addressCountry: "RU",
    };
  }

  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}
