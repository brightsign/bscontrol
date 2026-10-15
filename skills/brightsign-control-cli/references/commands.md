# `bsn-*` command index

All 66 command-line programs, grouped by subsystem. Each builds to `bin/<name>` via
`make build`. Run `<name> --help` for exact flags; `examples/README.md` has the exhaustive
per-command reference.

Naming: `bsn-[<subsystem>-]<object>-<verb>` — verb last; read-only `info`/`status`
commands may omit the verb.

## Authentication and account helpers (4)

| Command | Purpose |
|---|---|
| `bsn-auth-info` | Authenticate with BrightSign Control and display equivalent curl calls; writes a reusable `.token` file. |
| `bsn-token-test` | Test that authentication succeeds and the token is valid. |
| `bsn-generate-token-json` | Generate a `.token` JSON file for reuse by other tools and scripts. |
| `bsn-endpoints-test` | Probe BrightSign Control API endpoints for reachability and validity. |

## Account device management (11)

The cloud's inventory view of registered players.

| Command | Purpose |
|---|---|
| `bsn-device-list` | List and query registered devices, with filtering and pagination. |
| `bsn-device-find` | Find a registered device by serial number or other criteria. |
| `bsn-device-info` | Get real-time device information (via rDWS under the hood). |
| `bsn-device-status` | Get device status information. |
| `bsn-device-errors` | Retrieve device error logs. |
| `bsn-device-operations` | View operations queued/performed on devices. |
| `bsn-device-downloads` | View content downloads on devices. |
| `bsn-device-delete` | Delete devices from BrightSign Control (destructive). |
| `bsn-device-group-change` | Change a device's group assignment. |
| `bsn-device-ldws-enable` | Get / enable / disable a device's local Diagnostic Web Server via the cloud. |
| `bsn-device-ldws-password` | Enable local DWS and set/change/clear its password via the cloud. |

## Groups (3)

| Command | Purpose |
|---|---|
| `bsn-group-info` | Retrieve group information. |
| `bsn-group-update` | Update group settings. |
| `bsn-group-delete` | Delete groups (destructive). |

## Subscriptions (3)

| Command | Purpose |
|---|---|
| `bsn-subscription-list` | List device subscriptions on a network. |
| `bsn-subscription-count` | Count device subscriptions on a network. |
| `bsn-subscription-operations` | List subscription operations and permissions. |

## Registration and provisioning (3)

| Command | Purpose |
|---|---|
| `bsn-regtoken-get` | Create/fetch a device registration token used for provisioning. |
| `bsn-regtoken-revoke` | Revoke a device registration token. |
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
| `bsn-bdeploy-setup-delete` | Delete setup records by ID or package name (destructive). |
| `bsn-bdeploy-device-associate` | Associate or dissociate a device with a B-Deploy setup. |
| `bsn-bdeploy-device-list` | List B-Deploy devices on a network, optionally filtered by setup. |
| `bsn-bdeploy-device-get` | Get a B-Deploy device setup record by serial number. |
| `bsn-bdeploy-device-find` | Find a B-Deploy device by serial number. |
| `bsn-bdeploy-device-delete` | Permanently remove a device from B-Deploy (destructive). |
| `bsn-render-setup-template` | Render a B-Deploy setup template into a concrete JSON config. |

## Remote DWS — live single-player control (30)

Every `bsn-rdws-*` command targets one player by `--serial` and acts on it now.

### Lifecycle
| Command | Purpose |
|---|---|
| `bsn-rdws-reboot` | Remotely reboot a player (supports normal/crash/factory-reset/disable-autorun modes). |
| `bsn-rdws-reprovision` | Re-provision a player. |
| `bsn-rdws-storage-reformat` | Remotely reformat a storage device on a player (destructive). |
| `bsn-rdws-firmware-download` | Remotely download and apply a firmware update. |

### Files
| Command | Purpose |
|---|---|
| `bsn-rdws-files-list` | List files and directories on player storage. |
| `bsn-rdws-files-upload` | Upload a file to player storage. |
| `bsn-rdws-files-delete` | Delete a file from player storage (destructive). |
| `bsn-rdws-files-rename` | Rename a file on player storage. |
| `bsn-rdws-files-create-folder` | Create a folder on player storage. |

### Registry
| Command | Purpose |
|---|---|
| `bsn-rdws-registry-get` | Read player registry values. |
| `bsn-rdws-registry-set` | Write player registry values. |

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
