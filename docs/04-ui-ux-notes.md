# Sales Radar — UI / UX Notes

**Version:** April 2026 · **Surfaces:** `cmd/web/templates` (legacy), `frontend/` (Next.js)

---

## 1. Brand & typography

| Token | Value | Usage |
|-------|--------|--------|
| Primary | `#216ab7` | Primary buttons, links, active nav, key accents |
| Primary hover | `#1b5a9b` | Hover states |
| Primary active | `#174c83` | Active press |
| Primary soft | `#eaf3fb` | Selected rows, soft fills, focus pairing |
| Primary accent | `#c7dbef` | Light borders |
| Page bg | `#f8fafc` | Canvas |
| Surface | `#ffffff` | Cards, sidebar |
| Text | `#0f172a` / `#64748b` | Body / muted |

**Font:** **Inter** (Google Fonts in HTML; `next/font` in Next). System fallbacks: `system-ui`, sans-serif.

**Principle:** Primary blue is **sparing**—actions, selection, and focus—not full-bleed color backgrounds.

---

## 2. Legacy HTML UI (`cmd/web/templates`)

### 2.1 Shell

- **Sidebar:** Lead list, Settings; collapsible; logo image when expanded; icon-only when collapsed.
- **Main:** Page header, optional KPI grid after a run, filters drawer (`<details>`), toolbar (Generate, Export, Debug link), lead **cards** in a CSS grid.

### 2.2 Lead list row

- Columns: rank, company + domain, industry, **source** label (from trace), **signal** (truncated why-now), **readiness** badge, **priority** pill, **action** label, kebab menu (Edit → detail; Delete disabled).

### 2.3 Drawer

- Opens from card click / keyboard; **backdrop** and **Escape** close.
- **Active row**: primary soft fill + left border while drawer open.
- Drawer content: suggested action, narratives, badges, structured fields (implementation in `list.html` + `drawer_i18n.go` helpers).

### 2.4 Settings

- Discovery rows with name, description, toggle, **badge** (Configured / Missing config / Disabled / Enabled).
- ICP: multi-select industries, signals, region, weights, employee fields.

### 2.5 Detail page (`detail.html`)

- Narrow readable column; suggested action panel; definition lists; external **LinkedIn** link when present.

### 2.6 Debug (`debug.html`)

- Run meta, summary text, integration table, breakdown table, provider rows.

### 2.7 Motion

- `prefers-reduced-motion` respected for sidebar transition (see template CSS).

---

## 3. Next.js UI (`frontend/`)

### 3.1 Shell

- **AppShell:** left sidebar — Lead list, Settings, Debug; **Sales Radar** + “BAWANA · Sales” sublabel.
- **NavLink** highlights active route (including `/leads/*` under Lead list).

### 3.2 Lead list (`/leads`)

- **Page header** + **Generate leads** (client: POST pipeline, then URL merge for KPIs).
- **KPI strip** when `summary.last_run` or pipeline query params present.
- **Filters:** GET form to `/leads` with same query keys as legacy; **hidden** inputs preserve pipeline KPI params when changing filters.
- **Table:** #, company + domain, industry, source, signal, readiness, priority, action.
- **Export CSV** (client): `GET /api/v1/export.csv?` + current search params.
- **Debug** link: `/debug` + query echo.

**Parity note:** The **drawer** and **card grid** from HTML are **not** fully replicated in Next Phase 3; detail is via **`/leads/[id]`**. Future work may add drawer for parity.

### 3.3 Tokens file

- `styles/tokens.css` imported from `app/globals.css` for shared CSS variables.

---

## 4. Accessibility checklist

| Requirement | HTML | Next |
|-------------|------|------|
| Keyboard operable list rows / menus | ✓ (implemented) | Table links + buttons |
| Focus visible | ✓ | Tailwind focus rings on form controls |
| Semantic headings / landmarks | ✓ | PageHeader + `section` + `aria-labelledby` where used |
| Drawer Escape | ✓ | N/A until drawer added |

---

## 5. Responsive behavior

- **HTML:** Complex grid breakpoints in `list.html` (wide table-like columns).
- **Next:** Horizontal scroll for table on small viewports (`overflow-x-auto`).

---

## 6. Error & empty states

- **API/config missing:** Next shows inline **alert-style** message if `NEXT_PUBLIC_API_BASE_URL` unset or fetch fails (`ApiError`).
- **Empty DB / no run:** KPI strip replaced by **hint** copy (aligned with PRD).
- **No rows for filters:** Dedicated message on list.

---

*Last updated April 2026.*
