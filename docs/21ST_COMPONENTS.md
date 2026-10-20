# 21st.dev components used / adapted

**MCP status (agent session):** 21st.dev MCP tools were **not** available in the Cursor agent tool catalog after user reconnect (`GetDynamicTools` showed only `cursor`, `cursor-cloud`, `cursor-subscriptions`). Work proceeded via **`21st` CLI** (`search` / `get`) with the same catalog API.

**AI generate:** disabled on account (`aiGenerationEnabled: false`) — no `generate` / `iterate`.

## Design direction chosen

**Signal Forge** — graphite near-black (`#0C0E10`), cool porcelain light (`#E8EDF2`), electric mint accent (`#2EE6A6`). Display font **Syne**, body **Outfit**. Editorial asymmetry, kinetic type, no purple-AI / cream-terracotta / broadsheet clichés.

## Per-block selections (multiple options reviewed)

| Block | Options reviewed (21st ids) | Chosen / inspired by | Adaptation notes |
|---|---|---|---|
| Hero | 28280 Agency Hero, 5590 Animated Hero (particle grid), 4051 Hero Section | **5590** particle/cursor energy + brand-first composition (not trust-avatar clutter from 28280) | Full-bleed graphite, Syne kinetic headline, dual CTAs, engineering UI-layer visual, reduced-motion safe |
| Kinetic typography | (hero 5590 + custom) | Custom Motion split lines inspired by agency heroes | Word/line reveal, respects `prefers-reduced-motion` |
| Nav + mobile | 18191 Rich Nav, 21349 Motion Nav, sticky sheet from 28280 | Custom sticky header + sheet menu inspired by **28280/21349** | Russian labels, Signal Forge tokens, no mega-menu |
| Case showcase | 9607 Project Showcase (cursor preview), 29523 Portfolio Grid | **9607** list + floating preview | Empty-state when no cases; no fake projects |
| Service cards | 8223 Service Card, 7795 Services Card | **8223** hover motion language | Asymmetric editorial rows, not identical card grid |
| Process timeline | 26902 Vertical How It Works, 28381 Process Timeline | **26902** dashed rail + numbered nodes | 6 Nexera process steps, Russian copy |
| Multi-step brief | 23576 Wizard Steps, 28569 Questionnaire | **23576** progress + direction transitions (Questionnaire depends on proprietary `@shadcn/react/questionnaire`) | Custom RHF+Zod wizard; sources `project_brief` |
| Buttons | 1507 Magnetic Button, 26725 Magnetic Button (reduced-motion) | **26725** | Strength-tuned mint primary; `asChild` for Link |
| FAQ | 23530 Accordion, 31351 Accordion | **23530** spring accordion | API FAQ + cautious fallbacks |
| Final CTA | 19355 CTA Section, 28155 CTA Section | Large typography CTA inspired by both; avoid card-in-card from 19355 | Full-bleed mint/graphite finale |
| Footer | 21474 Agency Footer | **21474** multi-column sitemap | Only real contact channels from API |

## Install commands (reference; code adapted manually)

```text
21st get 5590   # Animated Hero
21st get 26725  # Magnetic Button
21st get 9607   # Project Showcase
21st get 8223   # Service Card
21st get 26902  # Vertical timeline
21st get 23576  # Wizard Steps
21st get 23530  # Accordion
21st get 21474  # Agency Footer
```
