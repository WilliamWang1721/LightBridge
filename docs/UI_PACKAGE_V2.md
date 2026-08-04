# LightBridge UI Package v2

UI Package v2 extends the existing safe ZIP/GitHub theme package format. Version 1 packages remain supported.

## Package layout

```text
lightbridge-ui.json
style.css
preview.png
pages/
images/
fonts/
```

JavaScript, Vue components, arbitrary HTML execution, external CSS imports, remote asset URLs, and data URLs remain prohibited.

## Manifest

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
    "heading": "natural",
    "font": "inter",
    "icon_library": "lucide",
    "radius": "default",
    "menu": "default",
    "menu_size": "suitable"
  },
  "config": [
    {
      "key": "primary_color",
      "label": "Primary color",
      "type": "color",
      "default": "#2563eb"
    },
    {
      "key": "surface_radius",
      "label": "Surface radius",
      "type": "number",
      "default": 8
    },
    {
      "key": "menu_treatment",
      "label": "Menu treatment",
      "type": "select",
      "default": "default",
      "options": ["default", "subtle", "outlined"]
    },
    {
      "key": "glass_surfaces",
      "label": "Glass surfaces",
      "type": "boolean",
      "default": false
    }
  ]
}
```

## Configuration fields

Supported field types:

- `color`
- `text`
- `select`
- `number`
- `boolean`

The administrator UI renders these fields as typed controls. Submitted values are validated again by the backend against the installed manifest.

Unknown fields and invalid value types are rejected.

## Fixed component style

The built-in modern component style is always `luma`.

Packages and users may customize supported visual axes, but package configuration may not declare or submit any of the following reserved keys:

- `style`
- `ui_style`
- `ui-style`
- `componentStyle`
- `component_style`
- `component-style`

Packages remain CSS/token overlays. They cannot replace the trusted Vue component implementation or execute application logic.

## CSS variables

Configuration keys are converted to namespaced custom properties:

```text
primary_color -> --theme-primary-color
surface_radius -> --theme-surface-radius
```

Package CSS should consume these variables with safe fallbacks:

```css
[data-ui-mode='package'][data-ui-package='example-package'] {
  --package-primary: var(--theme-primary-color, #2563eb);
  --package-radius: var(--theme-surface-radius, 8px);
}
```

## Compatibility behavior

- Legacy mode keeps the original LightBridge UI.
- Modern mode uses trusted LightBridge Luma components and layouts.
- Package mode applies the active safe UI package over the shared UI platform.
- When a package is removed or unavailable, the resolver falls back to Modern mode.
- User-adjustable axes are independent from package-specific fields.
