# UI Migration Validation

The LightBridge Luma migration is protected by the following maintained gates:

- Full-site semantic utility audit for Modern and Package modes.
- ESLint and Vue TypeScript validation.
- Critical frontend Vitest suites, including UI profile, route-surface, primitive, icon-provider, and chart-theme coverage.
- Production Vite build.
- Go unit and integration tests.
- golangci-lint, security scans, runtime contract checks, and full-stack smoke tests.

Legacy mode remains available as a compatibility fallback. Modern mode is the built-in default, while Package mode uses the same semantic component and token contract with validated package overrides.
