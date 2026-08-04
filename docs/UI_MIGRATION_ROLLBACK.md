# UI Migration Rollback

LightBridge keeps the Legacy profile as a supported compatibility fallback while Luma is the built-in default.

For a user-level rollback, select the Legacy UI mode in the profile settings. The profile resolver persists the choice and removes Modern/Package semantic overrides without changing business data or routes.

For an operator-level rollback, set the administrative UI default to Legacy. Existing user overrides continue to follow the configured precedence rules. Custom packages remain isolated behind Package mode and cannot replace the trusted component runtime.

A code rollback should revert the migration commit as a unit. Do not partially remove semantic tokens, portal scoping, route-surface classification, chart theming, or the compatibility bridges, because these layers are designed and validated together.
