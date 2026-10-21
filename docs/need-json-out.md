# `--json` Output Coverage

This document used to track a `--json` rollout across the example programs (under their
pre-`bsn-*` names: `main-*`, `bdeploy-*`, `rdws-*`). That rollout is complete: every
current `bsn-*` example that produces a result worth parsing supports `--json`.

**Exceptions, by design, not oversight:**

- `bsn-device-reg-script` — emits a registration *script* (`.brs` or `.sh`), not data;
  a `--json` mode doesn't fit its purpose.
- `bsn-generate-token-json` — always emits the token entity as JSON to stdout; there's no
  separate human-readable mode to flag away from.
- `bsn-render-setup-template` — renders a template into a concrete JSON config file; the
  output format is inherent to the tool, not a flag-gated choice.

See `examples/README.md` or `skill/brightsign-control-cli/references/commands.md` for the
current command list, and run `--help` on any command to confirm its exact flags.
