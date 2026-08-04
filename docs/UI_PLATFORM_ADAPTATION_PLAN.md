# LightBridge UI Platform Adaptation Plan

## 1. Objective

Build a complete, extensible UI platform for LightBridge rather than merely replacing existing components with a Luma component set.

The platform must support independent, composable switching of:

- UI mode and component style
- layout shell and page composition
- base color and semantic color tokens
- chart palette
- typography and heading system
- icon provider
- radius and surface treatment
- density and spacing
- navigation/menu presentation and size
- motion profile
- data-table presentation
- custom UI packages

The existing LightBridge UI remains available throughout migration. The built-in modern UI uses the fixed `luma` component style. Other supported parameters remain configurable by users and may also be supplied by safe UI packages.

## 2. Product rules

### Built-in Luma defaults

- Style: `luma` — fixed and not user configurable
- Base color: `natural`
- Chart color: `natural`
- Heading: `natural`
- Font: `inter`
- Icon library: `lucide`
- Radius: `default`
- Menu: `default`
- Menu size: `suitable`

### Compatibility

- The legacy UI remains selectable.
- Existing ZIP and GitHub UI package import remains supported.
- Existing package CSS, assets, Markdown pages, and menu extensions remain supported.
- UI package settings are generated from manifest parameters instead of requiring users to edit raw JSON.
- Packages cannot execute JavaScript or inject Vue components.
- A package cannot change the fixed built-in Luma style through a `style` parameter.

## 3. UI platform model

### 3.1 Independent axes

The platform must avoid one monolithic `theme` setting. It uses independent axes:

```ts
interface UIProfile {
  mode: 'legacy' | 'modern' | 'package'
  componentStyle: 'luma'
  layout: string
  baseColor: string
  chartColor: string
  heading: string
  font: string
  iconLibrary: string
  radius: string
  density: string
  menu: string
  menuSize: string
  motion: string
  tableStyle: string
  activePackageId?: string
}
```

This allows, for example, the same Luma components to run with a compact admin layout, a different font, a high-density table system, and a custom chart palette without producing a new full theme.

### 3.2 Configuration sources and precedence

1. LightBridge built-in defaults
2. Administrator defaults
3. Active UI package defaults and constraints
4. Browser-local fallback
5. Authenticated user preferences
6. Temporary preview overrides

The resolver validates every layer against the active capability registry before applying it.

### 3.3 Registries

Create typed registries for:

- component styles
- layouts
- base-color palettes
- chart palettes
- heading presets
- fonts
- icon providers
- radius presets
- density presets
- menu presets
- menu-size resolvers
- motion presets
- table presets
- UI package capabilities

Each registry entry contains metadata, defaults, compatibility declarations, preview information, and the function or tokens required to apply it.

### 3.4 Capability contract

Every layout, preset, or package declares capabilities instead of assuming all features exist.

Examples:

- `supportsDarkMode`
- `supportsRTL`
- `supportsCompactSidebar`
- `supportsMobileDrawer`
- `supportsDenseTables`
- `supportsCustomChartPalette`
- `supportsUserOverrides`
- `supportsPortalSurfaces`

Unsupported combinations are disabled in settings and resolved safely to defaults.

## 4. Layered architecture

1. Shared business layer: router, API, stores, permissions, i18n, feature flags.
2. UI profile resolver: defaults, validation, persistence, preview, rollback.
3. Semantic token layer: color, typography, spacing, radius, shadow, motion, z-index.
4. Primitive components: buttons, fields, overlays, navigation, data display.
5. Application components: account status, model badge, provider icon, balance display.
6. Layout shells: admin, user, auth, public, setup, payment.
7. Feature-page compositions.
8. Safe UI package overlay: tokens, CSS, assets, menu/page extensions.

Business logic must not be duplicated between legacy and modern pages. Page controllers/composables are shared; layouts and view composition may differ.

## 5. Scope and isolation

- Legacy styles remain available under a legacy shell.
- Modern tokens and components are scoped to a modern shell.
- Portal-based dialogs, popovers, menus, sheets, and tooltips render into a dedicated UI portal root carrying the active profile.
- Custom package CSS is scoped and loaded after compatible built-in tokens.
- Global reset rules are audited and minimized.
- A formal z-index contract is shared by both systems.
- Dark mode and reduced-motion accessibility settings remain globally coordinated.

## 6. UI package v2 extension

Extend the current `lightbridge-ui.json` schema without breaking v1 packages.

Suggested fields:

```json
{
  "schema_version": 2,
  "id": "example-package",
  "name": "Example Package",
  "version": "1.0.0",
  "entry_css": "style.css",
  "preview": "preview.png",
  "capabilities": {
    "ui_modes": ["legacy", "modern"],
    "supports_dark_mode": true,
    "supports_user_overrides": true
  },
  "defaults": {
    "base_color": "natural",
    "chart_color": "natural",
    "font": "inter"
  },
  "config": []
}
```

Rules:

- v1 remains valid.
- Reserved fields such as `style` cannot be declared as mutable config.
- Config types remain `color`, `text`, `select`, `number`, and `boolean` initially.
- Future field types are version-gated.
- Package variables are namespaced and converted to CSS custom properties.
- Settings UI is schema-generated with validation, preview, reset, and unsaved-change handling.

## 7. Work plan

### Phase 0 — Full inventory and baseline

- Inventory routes, views, layouts, shared components, dialogs, forms, tables, charts, menus, and responsive behaviors.
- Classify every UI surface by admin, user, auth, public, setup, payment, and system.
- Capture desktop, tablet, and mobile baselines for critical flows.
- Identify global CSS leakage and duplicated UI patterns.
- Establish accessibility, keyboard, performance, and bundle baselines.

### Phase 1 — UI platform foundation

- Add typed UI profile, defaults, registries, and capability contracts.
- Add preference resolver and persistence adapters.
- Add fixed Luma style metadata.
- Add DOM profile attributes and a dedicated portal root.
- Add semantic token sheets for light/dark modes.
- Preserve legacy startup and current package injection.
- Add unit tests for resolution, validation, precedence, and fallback.

### Phase 2 — Package manager and settings generator

- Parse manifest config fields into typed frontend models.
- Replace raw JSON configuration with generated controls.
- Add package metadata, preview, compatibility, source, version, and capability views.
- Add reserved-field validation and safe fallback.
- Add live preview and rollback.
- Preserve CLI and Admin API compatibility.

### Phase 3 — Primitive component system

Cover and verify:

- Button and icon button
- Input, textarea, label, help, and error text
- Checkbox, radio, switch, select, combobox
- Card, badge, avatar, separator, skeleton
- Dialog, alert dialog, sheet, drawer
- Dropdown, context menu, popover, tooltip
- Tabs, accordion, collapsible
- Toast and progress
- Breadcrumb and pagination
- Calendar and date controls
- Data-table foundations

Each primitive requires light/dark, disabled/loading/error, keyboard, screen-reader, mobile, RTL readiness, and reduced-motion coverage.

### Phase 4 — Layout system

Create interchangeable shells and layout contracts:

- Admin shell
- User shell
- Auth shell
- Public shell
- Setup shell
- Payment shell

Optimize information hierarchy, content widths, page headers, navigation, action placement, sticky regions, drawers, breadcrumbs, command/search access, and responsive behavior.

`menuSize=suitable` is a resolver, not a fixed width. It chooses the appropriate presentation based on viewport, locale, menu labels, and selected density.

### Phase 5 — Administrator UI coverage

1. Dashboard and analytics
2. Accounts and account groups
3. Users and permissions
4. Model catalog, mapping, and pricing
5. API keys and redemption
6. Monitoring, logs, and error analysis
7. Billing, orders, payments, and subscriptions
8. Announcements and custom pages
9. System, authentication, email, storage, proxy, backup, and version settings
10. UI platform settings and package manager

Each page is redesigned at the workflow and layout level before primitive replacement.

### Phase 6 — User UI coverage

1. Dashboard and usage
2. API keys
3. Balance, recharge, orders, and subscriptions
4. Profile and security
5. Payment-provider flows
6. Announcements and custom pages
7. Loading, empty, callback, degraded, and error states

### Phase 7 — Charts, tables, and dense workflows

- Map chart presets to semantic variables.
- Keep Chart.js where appropriate while sourcing colors from tokens.
- Standardize legends, tooltips, filters, date ranges, loading, no-data, and export actions.
- Standardize virtualized tables, bulk actions, filters, column controls, row actions, and mobile alternatives.

### Phase 8 — User customization experience

- Add controls for mode, layout, base color, chart color, heading, font, icon library, radius, density, menu, menu size, motion, and table style.
- Keep Luma style fixed and read-only or hidden.
- Add live preview, save, reset, account sync, local fallback, admin defaults, and package-provided constraints.
- Make unsupported combinations visibly unavailable rather than silently failing.

### Phase 9 — Validation and rollout

- Unit tests for profile parsing, registry resolution, capabilities, package schema, and reserved fields.
- Component interaction and accessibility tests.
- Route smoke tests in legacy, modern, and package modes.
- Visual regression across desktop, tablet, and mobile.
- Keyboard-only and screen-reader checks.
- Bundle-size and runtime-performance budgets.
- Progressive rollout, telemetry, rollback, and legacy deprecation criteria.

## 8. Completion criteria

A UI surface is covered only when:

- its layout and information hierarchy have been reviewed;
- its primitives use the approved system;
- responsive behavior is verified;
- keyboard and accessibility behavior is verified;
- light and dark modes are verified;
- loading, empty, error, and permission states are covered;
- legacy and package-mode compatibility is known;
- visual regression coverage exists.

The project is complete only when every route is listed in the coverage matrix and no unclassified UI surface remains.
