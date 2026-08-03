# Codebase inventory maintenance

`CODEBASE_INVENTORY.tsv` is a deterministic inventory of committed text files used by the release audit gate.

The `Codebase Inventory` GitHub Actions workflow regenerates the inventory after changes land on `main`. It stages only `docs/architecture/CODEBASE_INVENTORY.tsv`, verifies the generated file with `tools/codebase_inventory.py --check`, runs `git diff --check`, and commits the refreshed inventory with `[skip ci]` when required.

The inventory file itself is excluded from the workflow trigger to prevent commit loops. Release jobs continue to run `tools/codebase_inventory.py --check`; a stale inventory therefore remains a hard release failure rather than being silently ignored.
