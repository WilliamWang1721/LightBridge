# Local Module Marketplace

LightBridge's default Marketplace source is `local`.

- By default it scans `${LIGHTBRIDGE_DATA_DIR}/MODULES` for `*.zip`.
- You can override the scan folder with `LIGHTBRIDGE_MODULES_DIR`.
- Optionally, if the current working directory contains a folder named `MODULES` (exact casing), `local` will scan `./MODULES`.
