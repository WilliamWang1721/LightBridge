# Phase 2 protocol routing rollout

This phase is intentionally incremental. No routing behavior is changed until the repository baseline is green.

## Invariants

1. `Group.platform` remains compatibility metadata during migration.
2. Message routing decisions must eventually be based on normalized inbound protocol, account capabilities, relay mode, and available adapters.
3. Embeddings, image, realtime, files, batch, and rerank endpoints remain behind explicit capability gates until dedicated adapters exist.
4. Every routing change must include focused regression tests and must keep the full CI and Security Scan green.
5. No temporary workflow may write back to the branch or default branch.

## Rollout order

1. Establish a green baseline and synchronize dependency lockfiles.
2. Introduce read-only protocol capability helpers and tests.
3. Migrate one endpoint family at a time.
4. Observe scheduler diagnostics and compatibility telemetry.
5. Remove legacy routing authority only after all endpoint families have capability adapters.
