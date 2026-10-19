---
name: brightsign-control-cli
description: >-
  Operate BrightSign players and fleets from the command line using the bscontrol
  example CLIs (the `bsn-*` programs built from this repo). Use this whenever a task
  involves talking to BrightSign Control (formerly BSN.cloud / B-Deploy / rDWS) from a
  shell or script: listing or querying registered players, provisioning new players with
  B-Deploy setups, rebooting / reprovisioning / snapshotting a live player, reading or
  writing its registry, pulling logs/crashdumps/diagnostics, managing groups or
  subscriptions, enabling SSH/telnet/local-DWS, minting a registration token, or
  joining/placing a fielded player onto a network by registry keys ("opt-out" provisioning),
  or scripting any fleet operation. Trigger on mentions of BrightSign, BSN.cloud, BrightSign
  Control, B-Deploy, rDWS, registration token, provisioning, a player serial number,
  "the bsn-* tools", bscontrol CLIs, or "run it against the player/fleet" — even when the
  exact command name is not given.
---

# BrightSign Control CLI (`bsn-*` tools)

A suite of 65 single-purpose command-line programs for operating BrightSign players and
fleets through **BrightSign Control** (the cloud platform formerly called BSN.cloud,
including its **B-Deploy** provisioning API and **rDWS** remote-player API). Each program
does one thing and shares a common flag and auth convention (with the per-command
exceptions this skill calls out), and can emit JSON for scripting.

Use this skill to pick the right command, invoke it correctly, and chain commands into
fleet workflows. For the complete command list with one-line purposes — including which
exact flag skips a confirmation prompt, since that varies by command — read
`references/commands.md`. For per-command flag documentation beyond `--help`, the repo's
`examples/README.md` covers most commands (a few newer ones aren't in it yet).

## Build and locate the tools

The programs are example mains in `examples/`; build them with the Makefile — never
`go build` directly:

```bash
make build          # compiles every examples/<name> into bin/<name>
ls bin/bsn-*        # the whole toolkit
./bin/bsn-rdws-reboot --help
```

Every binary name equals its source directory name. If a binary is missing, run
`make build` rather than looking for it elsewhere.

## Authentication and environment

All tools use the OAuth2 client-credentials flow and read credentials from the
environment. Set these once per shell:

| Variable | Meaning | Required |
|---|---|---|
| `BS_CLIENT_ID` | BrightSign Control API client ID | yes |
| `BS_SECRET` | BrightSign Control API client secret | yes |
| `BS_NETWORK` | Default network (account) name | optional; override with `--network`/`-n` |

Start by confirming auth works, which also writes a reusable `.token` file:

```bash
./bin/bsn-auth-info          # authenticate + show equivalent curl calls
./bin/bsn-token-test         # verify the token is valid
```

Treat missing/invalid credentials as the first thing to check when a command fails —
most non-zero exits that are not usage errors are auth or network-selection problems.

## How the names are built (so you can guess the command)

Names follow `bsn-[<subsystem>-]<object>-<verb>`. The verb comes last; read-only
"info/status" commands may omit an explicit verb. The optional subsystem segment tells you
which API surface — and which mental model — applies:

- **(no subsystem) `bsn-<object>-<verb>`** — BrightSign Control **account management**:
  the cloud's view of devices, groups, subscriptions, registration tokens, auth.
  Example: `bsn-device-list`, `bsn-group-update`, `bsn-subscription-count`.
- **`bsn-bdeploy-<object>-<verb>`** — **B-Deploy provisioning**: setup records and the
  device associations that let a player self-configure on first boot.
  Example: `bsn-bdeploy-setup-add`, `bsn-bdeploy-device-associate`.
- **`bsn-rdws-<object>-<verb>`** — **Remote DWS**: commands that act on a **single live
  player** right now (reboot, files, registry, diagnostics, network, firmware).
  Example: `bsn-rdws-reboot`, `bsn-rdws-registry-get`.

This matters because the same noun means different things per subsystem: a B-Deploy
"device" is a provisioning record, while an account "device" is a registered, possibly
online player. When a user says "the device," figure out whether they mean the fleet
inventory (account), its provisioning (B-Deploy), or the box itself (rDWS).

## Choosing the right command

Map the user's intent to a subsystem first, then to a verb:

| The user wants to... | Reach for |
|---|---|
| See/inspect what's in the account inventory | `bsn-device-list`, `bsn-device-info`, `bsn-device-status`, `bsn-device-errors` |
| Find a serial but don't know which network it's on | `bsn-device-find` (account) / `bsn-bdeploy-device-find` (B-Deploy) — both search **every network you have access to**; neither takes/needs `--network` |
| Move a player between groups, delete it from the account | `bsn-device-group-change`, `bsn-device-delete` |
| Manage groups or subscriptions | `bsn-group-*`, `bsn-subscription-*` |
| Mint/revoke a provisioning registration token | `bsn-regtoken-get`, `bsn-regtoken-revoke` |
| Provision new players (define how they configure) | `bsn-bdeploy-setup-*`, `bsn-render-setup-template` |
| Bind a specific serial to a setup | `bsn-bdeploy-device-associate` |
| Do something to a live player *now* | the `bsn-rdws-*` family (see below) |

Within `bsn-rdws-*`, group by job:

- **Lifecycle:** `reboot`, `reprovision`, `storage-reformat`, `firmware-download`
- **Files:** `files-list`, `files-upload`, `files-delete`, `files-rename`, `files-create-folder`
- **Registry:** `registry-get`, `registry-set`
- **Inspect / diagnose:** `info`, `health`, `status`-like via account tools, `logs-get`, `crashdump-get`, `diagnostics`, `snapshot`
- **Network:** `network-config`, `network-neighborhood`, `ping`, `dns-lookup`, `traceroute`, `packet-capture`
- **Time:** `time-get`, `time-set`
- **Access/security:** `ssh`, `telnet`, `dws-password`, `local-dws`
- **Misc:** `custom-data`

Local DWS appears in two places on purpose: `bsn-rdws-local-dws` / `bsn-rdws-dws-password`
act directly on the player, while `bsn-device-ldws-enable` / `bsn-device-ldws-password`
drive the same state through the cloud account API. Prefer the account (`bsn-device-ldws-*`)
path when you only have cloud credentials and a serial; use the rDWS path when you are
already operating on that live player.

## Invocation conventions (shared by nearly every tool)

Run `--help` on any command before first use — it is the source of truth for that tool's
flags. Most of the flags below are shared, with the exceptions noted:

| Flag | Purpose |
|---|---|
| `--help` | Usage and flags |
| `--network <name>` / `-n` | Select the network/account (overrides `BS_NETWORK`). Not accepted by `bsn-device-find`/`bsn-bdeploy-device-find` (they search every network) or by a few account tools (`bsn-device-group-change` among them) that read `BS_NETWORK` only, with no per-call override |
| `--serial <serial>` | Target a player by serial number |
| `--id <id>` | Target a device by numeric ID (where supported) |
| `--json` | Machine-readable output — use this whenever you will parse the result |
| `--verbose` | Extra detail for debugging (not every command has it) |
| `--timeout <seconds>` | Request timeout (default 30, where supported) |
| `-y` and/or `--force` | Skip a confirmation prompt — **not interchangeable aliases on every command.** Some commands accept only `-y`, some only `--force`, some both (true aliases). Passing the wrong one is a flag-parsing error, not a no-op. `references/commands.md` names the exact flag(s) for every command that needs one. |

Rules that keep scripted use safe and predictable:

- **Parse JSON, not prose.** Add `--json` for any output you intend to read programmatically;
  the human-readable format is not a stable contract.
- **Destructive/state-changing commands prompt by default, and the skip flag isn't uniform.**
  `bsn-device-delete`/`bsn-group-delete`/`bsn-rdws-reboot`/`bsn-rdws-registry-set` accept
  either `-y` or `--force` (true aliases). `bsn-bdeploy-device-delete`,
  `bsn-bdeploy-setup-delete`, `bsn-rdws-files-delete`, and `bsn-regtoken-revoke` accept
  **only** `--force`. `bsn-device-ldws-enable`, `bsn-device-ldws-password`,
  `bsn-rdws-firmware-download`, `bsn-rdws-reprovision`, and `bsn-rdws-storage-reformat`
  accept **only** `-y`. Check `references/commands.md` before scripting one you haven't
  used; in automation, pass the flag deliberately — never to silence a prompt you did not
  read. Several of these also **require** the skip-confirmation flag whenever `--json` is
  set, since they cannot show an interactive prompt in that mode — `bsn-rdws-reboot`,
  `bsn-rdws-registry-set`, `bsn-rdws-reprovision`, `bsn-rdws-storage-reformat`,
  `bsn-rdws-firmware-download`, and `bsn-rdws-files-delete`.
- **`bsn-device-group-change` has no skip flag at all.** Assigning a device to a group name
  that doesn't already exist blocks on an interactive y/N prompt with no way to bypass it —
  don't target an unverified group name from a script.
- **Exit codes are meaningful.** A non-zero exit means the operation failed; check it in
  scripts rather than grepping stdout. See `examples/README.md` for the exit-code table.
- **One player vs. the fleet.** The `bsn-rdws-*` tools act on a single `--serial`. To apply
  an rDWS operation across many players, enumerate with `bsn-device-list --json` and loop.

## Common workflows

**Provision a brand-new player:**
```bash
./bin/bsn-bdeploy-setup-add setup.json                 # create the setup record
./bin/bsn-bdeploy-device-associate \
    --serial <SERIAL> --setup-name "<package-name>"    # bind the serial to it
# player self-configures on next boot
```

**Enable remote diagnostics on a live player, then pull logs:**
```bash
./bin/bsn-device-ldws-password --serial <SERIAL> -p <PASSWORD>   # enable DWS + set password
./bin/bsn-rdws-logs-get --serial <SERIAL> --json
./bin/bsn-rdws-crashdump-get --serial <SERIAL>
```

**Audit the fleet, then act on a subset (scripting pattern):**
```bash
./bin/bsn-device-list --json \
  | jq -r '.devices[] | select(.online==false) | .serial' \
  | while read s; do ./bin/bsn-rdws-reboot --serial "$s" -y; done
```

**Change a player's group assignment:**
```bash
./bin/bsn-device-group-change --serial <SERIAL> --group "<GROUP_NAME>"
```
Only safe to script when `<GROUP_NAME>` is a group you already confirmed exists (e.g. via
`bsn-group-info`) — a nonexistent name triggers an interactive y/N prompt this command has
no flag to skip, and it will hang waiting on stdin. There is also no `--network` flag here;
set `BS_NETWORK` first.

**Join a fielded player to a network by registry keys ("opt-out"):** add an
already-running player to a BrightSign Control network in place — no wipe, no re-sync. Mint
a reusable registration token, write **five `networking` registry keys**
(`bsnce=true`, `a=<network>`, `bsnrt=<token>`, `sut=Standalone`, `inp=yes`), then reboot;
the player self-registers over its existing WebSocket.

```bash
# Easiest: generate a ready-to-run script (token already embedded); --file or redirect stdout:
./bin/bsn-device-reg-script --network "My Network" --type sh  --file register-player.sh   # registry-CLI
./bin/bsn-device-reg-script --network "My Network" --type brs > register-player.brs        # BrightScript
# then run it on the player (sh is for an insecure player's shell, e.g. bash /storage/sd/register-player.sh)
until ./bin/bsn-rdws-info --serial "$SERIAL" --json; do sleep 10; done   # verify (backoff)
```

`bsn-device-reg-script` mints the token and writes the whole script to stdout; of the other
`bsn-*` tools only `bsn-regtoken-get` (mint alone) and `bsn-rdws-info` (verify) participate.
**Do not try to write the keys with `bsn-rdws-registry-set`** — the `bsn-rdws-*` tools only
address players *already registered on your network*, and an opt-out target is not
registered yet; the keys are delivered on the player (local DWS, the `registry` CLI over
SSH/telnet, or BrightScript). This is a privileged operation with real requirements and
caveats — read `references/opt-out-provisioning.md` before running it.

## When you need more detail

- `references/commands.md` — every `bsn-*` command grouped by subsystem with a one-line
  purpose. Read this to discover or confirm a command name.
- `references/opt-out-provisioning.md` — the full recipe for joining a fielded player to a
  network by registry keys (the five keys, token minting, delivery methods, verification,
  and the scope/OS requirements). Read this before attempting opt-out.
- `examples/README.md` (repo root of the examples) — per-command flags, configuration
  file formats, pagination, exit codes, and worked examples for most commands (a few
  newer ones — `bsn-generate-token-json`, `bsn-regtoken-get`/`-revoke`,
  `bsn-device-find`/`bsn-bdeploy-device-find`, `bsn-endpoints-test`,
  `bsn-bdeploy-setup-add-raw`, `bsn-render-setup-template` — aren't documented there yet;
  use `--help` for those). Go here when `--help` is not enough.
