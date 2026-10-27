# `bsn-*` command index

All 77 command-line programs, grouped by subsystem. Each builds to `bin/<name>` via
`make build`. Run `<name> --help` for exact flags; `examples/README.md` has per-command
flag detail for most commands (a handful of newer commands aren't in it yet — `--help`
always works).

Naming: `bsn-[<subsystem>-]<object>-<verb>` — verb last; read-only `info`/`status`
commands may omit the verb.

**Skip-confirmation flag varies by command.** Destructive/state-changing commands prompt
unless you pass a flag, but which flag is not uniform: `-y` only, `--force` only, or both
as true aliases, depending on the command. The Purpose column below names the exact flag(s)
each one accepts — passing the wrong one is a hard "flag provided but not defined" error,
not a silent no-op. Several of these (noted below) also *require* the skip-confirmation
flag when `--json` is set, since they cannot show an interactive prompt in that mode.

## Authentication and account helpers (4)

| Command | Purpose |
|---|---|
| `bsn-auth-info` | Authenticate with BrightSign Control and display equivalent curl calls; writes a reusable `.token` file. |
| `bsn-token-test` | Test that authentication succeeds and the token is valid. |
| `bsn-generate-token-json` | Generate a device **registration** token as a `bsnDeviceRegistrationTokenEntity` JSON object, ready to embed in a B-Deploy raw setup JSON (for `bsn-bdeploy-setup-add-raw`). Not the auth `.token` file `bsn-auth-info` writes — see `bsn-regtoken-get` for the equivalent that returns just the bare token string. |
| `bsn-endpoints-test` | Probe BrightSign Control API endpoints for reachability and validity. |

## Account device management (10)

The cloud's inventory view of registered players.

| Command | Purpose |
|---|---|
| `bsn-device-list` | List and query registered devices, with filtering and pagination. Requires `--network`/`BS_NETWORK`. |
| `bsn-device-find` | Find a registered device by serial number by searching **across all networks** you have access to — no `--network` needed or accepted. Use this when you don't know which network a serial belongs to. |
| `bsn-device-info` | Get real-time device information (via rDWS under the hood). |
| `bsn-device-status` | Get device status information. |
| `bsn-device-errors` | Retrieve device error logs. |
| `bsn-device-downloads` | View content downloads on devices. |
| `bsn-device-delete` | Delete devices from BrightSign Control (destructive; `--force` or `-y`, true aliases). |
| `bsn-device-group-change` | Change a device's group assignment. No `--network` flag — set `BS_NETWORK` in the environment. Assigning to a group name that doesn't already exist blocks on an interactive y/N prompt with **no flag to skip it** — don't use an unverified group name in a script. |
| `bsn-device-ldws-enable` | Get / enable / disable a device's local Diagnostic Web Server via the cloud (`-y` to skip the enable/disable confirmation). Unlike the `bsn-rdws-*` family, `--json` alone silently skips the prompt too — no error, no `-y` required. |
| `bsn-device-ldws-password` | Enable local DWS and set/change/clear its password via the cloud (`-y` to skip confirmation). Same `--json`-silently-skips-prompt behavior as `bsn-device-ldws-enable`. |

## Groups (3)

| Command | Purpose |
|---|---|
| `bsn-group-info` | Retrieve group information. |
| `bsn-group-update` | Update group settings. |
| `bsn-group-delete` | Delete groups (destructive; `--force` or `-y`, true aliases). |

## Subscriptions (3)

| Command | Purpose |
|---|---|
| `bsn-subscription-list` | List device subscriptions on a network. |
| `bsn-subscription-count` | Count device subscriptions on a network. |
| `bsn-subscription-operations` | List subscription operations and permissions. |

## Content library, presentations and schedules (12)

All print JSON to stdout and status to stderr; the delete tools ask first unless `--force` or `-y`.

| Command | Purpose |
|---|---|
| `bsn-content-list` | List library content (`--folder`, `--filter`, `--all`). |
| `bsn-content-get` | One content item's metadata (`--id`). |
| `bsn-content-delete` | Delete content by `--id` or `--filter` (destructive). |
| `bsn-content-upload` | Upload a file (`--file`, `--path`, `--media-type`); reuses an identical file already at that path. |
| `bsn-content-download` | Download a file or `--thumbnail` to `--output` (or `-`). |
| `bsn-presentation-list` | List presentations. |
| `bsn-presentation-get` | One presentation by `--id` or `--name`. |
| `bsn-presentation-create` | Create a presentation (`--name`, or a full request with `--from-json`). |
| `bsn-presentation-delete` | Delete by `--id`, `--name` or `--filter` (destructive; unschedule first). |
| `bsn-schedule-list` | A group's scheduled presentations (`--group`). |
| `bsn-schedule-add` | Schedule a presentation on a group (default all day, every day). |
| `bsn-schedule-delete` | Remove a schedule entry (`--group`, `--id`; destructive). |

## Registration and provisioning (3)

| Command | Purpose |
|---|---|
| `bsn-regtoken-get` | Create/fetch a device registration token used for provisioning. |
| `bsn-regtoken-revoke` | Revoke a device registration token (destructive; `--force` only — `-y` is not accepted). |
| `bsn-device-reg-script` | Mint a token and emit a ready-to-run **registration script** (`--type brs` or `sh`) that writes the five opt-out keys and reboots. Written to stdout, or to `--file <path>`. See `opt-out-provisioning.md`. |

## B-Deploy provisioning (12)

Setup records plus the device associations that let a player self-configure on first boot.

| Command | Purpose |
|---|---|
| `bsn-bdeploy-setup-add` | Create a B-Deploy setup record from a JSON configuration (Go-struct path). |
| `bsn-bdeploy-setup-add-raw` | Create a B-Deploy setup from raw JSON, preserving all fields verbatim. |
| `bsn-bdeploy-setup-get` | Retrieve a specific B-Deploy setup record. |
| `bsn-bdeploy-setup-list` | List all B-Deploy setup records for a network. |
| `bsn-bdeploy-setup-update` | Update an existing B-Deploy setup record. |
| `bsn-bdeploy-setup-delete` | Delete setup records by ID or package name (destructive; `--force` only — `-y` is not accepted). |
| `bsn-bdeploy-device-associate` | Associate or dissociate a device with a B-Deploy setup. |
| `bsn-bdeploy-device-list` | List B-Deploy devices on a network, optionally filtered by setup. |
| `bsn-bdeploy-device-get` | Get a B-Deploy device setup record by serial number. |
| `bsn-bdeploy-device-find` | Find a B-Deploy device by serial number by searching **across all networks** — no `--network` needed or accepted. |
| `bsn-bdeploy-device-delete` | Permanently remove a device from B-Deploy (destructive; `--force` only — `-y` is not accepted). |
| `bsn-render-setup-template` | Render a B-Deploy setup template into a concrete JSON config. |

## Remote DWS — live single-player control (30)

Every `bsn-rdws-*` command targets one player by `--serial` and acts on it now.

### Lifecycle
| Command | Purpose |
|---|---|
| `bsn-rdws-reboot` | Remotely reboot a player (`--type`/`-t` normal/crash/factoryreset/disableautorun — no hyphens in the value). Always prompts unless `-y`/`--force` (true aliases); **required** when combined with `--json`, which cannot prompt. |
| `bsn-rdws-reprovision` | Re-provision a player (destructive; `-y` only — `--force` is not accepted; required with `--json`). |
| `bsn-rdws-storage-reformat` | Remotely reformat a storage device on a player (destructive; `-y` only — `--force` is not accepted; required with `--json`). |
| `bsn-rdws-firmware-download` | Remotely download and apply a firmware update (`-y` only to skip confirmation; required with `--json`). |

### Files
| Command | Purpose |
|---|---|
| `bsn-rdws-files-list` | List files and directories on player storage. |
| `bsn-rdws-files-upload` | Upload a file to player storage. |
| `bsn-rdws-files-delete` | Delete a file from player storage (destructive; `--force` only — `-y` is not accepted; required with `--json`). |
| `bsn-rdws-files-rename` | Rename a file on player storage. |
| `bsn-rdws-files-create-folder` | Create a folder on player storage. |

### Registry
| Command | Purpose |
|---|---|
| `bsn-rdws-registry-get` | Read player registry values. |
| `bsn-rdws-registry-set` | Write player registry values (`-y`/`--force`, true aliases, to skip confirmation; required with `--json`). |

### Inspect and diagnose
| Command | Purpose |
|---|---|
| `bsn-rdws-info` | Retrieve player information. |
| `bsn-rdws-health` | Check player health status. |
| `bsn-rdws-logs-get` | Retrieve player log files. |
| `bsn-rdws-crashdump-get` | Retrieve player crash dumps. |
| `bsn-rdws-diagnostics` | Run network diagnostics on a player. |
| `bsn-rdws-snapshot` | Capture a screenshot from a player. |

### Network
| Command | Purpose |
|---|---|
| `bsn-rdws-network-config` | Manage network configuration on a player. |
| `bsn-rdws-network-neighborhood` | Discover the network neighborhood from a player. |
| `bsn-rdws-ping` | Ping hosts from a player. |
| `bsn-rdws-dns-lookup` | Perform DNS lookups from a player. |
| `bsn-rdws-traceroute` | Perform a traceroute from a player. |
| `bsn-rdws-packet-capture` | Capture packets on a player. |

### Time
| Command | Purpose |
|---|---|
| `bsn-rdws-time-get` | Read player time. |
| `bsn-rdws-time-set` | Set player time. |

### Access and security
| Command | Purpose |
|---|---|
| `bsn-rdws-ssh` | Manage SSH access on a player (enable/disable and password). |
| `bsn-rdws-telnet` | Manage telnet access on a player. |
| `bsn-rdws-dws-password` | Get/set the Diagnostic Web Server password directly on a player. |
| `bsn-rdws-local-dws` | Manage local DWS directly on a player. |

### Misc
| Command | Purpose |
|---|---|
| `bsn-rdws-custom-data` | Send custom data to a player via UDP. |
