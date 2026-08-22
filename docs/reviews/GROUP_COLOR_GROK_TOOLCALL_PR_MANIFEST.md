# Group Color and Grok Build Tool Calling PR Manifest

This draft tracks the reviewed Stage 12 implementation for:

- Persistent custom group colors with strict `#RRGGBB` validation.
- Stable automatic fallback colors for existing groups.
- Group selector layout fixes so protocol badges no longer obscure group names.
- Grok Build tool-calling compatibility fixes for `tool_choice`, sequential tool calls, argument completion, terminal response snapshots, sparse output indexes, and duplicate terminal events.
- A dedicated Grok Tool Calling account probe.
- Migration `161_add_group_color.sql` and Ent runtime descriptor alignment.

## Reviewed source archive

- Google Drive: https://drive.google.com/file/d/1NqJAVDm61KlMZh76TKScGi6FQqez0BiQ/view?usp=drivesdk
- Local artifact: `LightBridge-0.2.80-group-color-grok-toolcall-stage12-full.zip`

The archive was created after repository-level static checks and is retained as the source-of-truth payload for this draft.

## Validation completed in the sandbox

- Go formatting and AST parsing.
- Frontend TypeScript/Vue syntax parsing.
- YAML/JSON parsing and duplicate-key checks.
- GitHub Actions shell syntax checks.
- Release configuration invariant checks.
- Codebase inventory validation.
- Secret scan.
- ZIP CRC/integrity validation.

Full Go 1.26.5 compilation, pnpm build, and real Grok Build E2E tool-calling tests remain required before this PR can be marked ready for merge.

## Important delivery note

The ChatGPT GitHub connector cannot upload a sandbox ZIP as a repository tree in one operation. This draft is intentionally not presented as merge-ready until the reviewed payload is applied to the branch without overwriting newer `main` changes.
