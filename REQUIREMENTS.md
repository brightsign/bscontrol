# Requirements

## generate-token-json Example

### Purpose

Provide a standalone tool that authenticates with BSN.cloud, generates a device
registration token, and outputs the complete `bsnDeviceRegistrationTokenEntity`
JSON object ready to be pasted into a setup JSON file for use with
`AddSetupRecordRaw`.

### Output Format

When invoked with `--json` (or by default), the tool writes to stdout a JSON
object with exactly these four fields:

```json
{
  "token": "<generated token string>",
  "scope": "cert",
  "validFrom": "2026-04-08T15:00:20.000Z",
  "validTo": "2028-04-07T15:00:20.000Z"
}
```

This output can be directly used as the value of `bsnDeviceRegistrationTokenEntity`
in a setup JSON file.

### Workflow

1. Authenticate with BSN.cloud using `BS_CLIENT_ID` / `BS_SECRET` credentials.
2. Resolve the network name from `--network` flag or `BS_NETWORK` env var.
3. Set the network context via `client.BDeploy.SetNetworkContext`.
4. Call `client.Provisioning.GenerateDeviceToken` to generate a new token.
5. Output the token entity JSON to stdout.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--network` / `-n` | `BS_NETWORK` env | Network name (required) |
| `--timeout` | 30 | HTTP request timeout in seconds |
| `--json` | true | Output token entity as JSON to stdout (default behavior) |
| `--verbose` | false | Show extra details on stderr |
| `--help` | false | Display usage information |

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `BS_CLIENT_ID` | Yes | BSN.cloud API client ID |
| `BS_SECRET` | Yes | BSN.cloud API client secret |
| `BS_NETWORK` | No | Default network name |

### Status and progress messages

All status/progress messages go to stderr so stdout contains only the JSON output.

### Exit Codes

- 0: success
- 1: any error (auth failure, network error, token generation failure)

---

## render-setup-template Example

### Purpose

Provide a standalone tool that takes a Go-template JSON file, substitutes
template variables from CLI flags and environment variables, dynamically
generates a fresh device registration token, and outputs the fully rendered
setup JSON to stdout. The output is ready to be posted via `AddSetupRecordRaw`
or saved to a file.

### Workflow

1. Read the template JSON file specified as a positional argument.
2. Resolve template variable values from CLI flags, falling back to environment
   variables.
3. Authenticate with BSN.cloud using `BS_CLIENT_ID` / `BS_SECRET`.
4. Set the network context via `client.BDeploy.SetNetworkContext`.
5. Call `client.Provisioning.GenerateDeviceToken` to get a fresh token.
6. Populate the `TemplateVars` struct with all resolved values including the
   generated token fields (`RegistrationToken`, `TokenValidFrom`, `TokenValidTo`).
7. Execute the Go template with those variables.
8. Validate the rendered output is valid JSON.
9. Write the rendered JSON to stdout.

### Template Variables Substituted

| Template Placeholder | Source | Env Fallback | Default |
|---|---|---|---|
| `{{.Username}}` | `--username` | `BS_USERNAME` | (required) |
| `{{.NetworkName}}` | `--network` | `BS_NETWORK` | (required) |
| `{{.PackageName}}` | `--package-name` | (none) | (required) |
| `{{.SetupType}}` | `--setup-type` | `BS_SETUP_TYPE` | `"bsn"` |
| `{{.DeviceName}}` | `--device-name` | (none) | `""` |
| `{{.DeviceDescription}}` | `--device-description` | (none) | `""` |
| `{{.GroupName}}` | `--group` | `BS_GROUP_NAME` | `"Default"` |
| `{{.RegistrationToken}}` | (generated) | - | - |
| `{{.TokenValidFrom}}` | (generated) | - | - |
| `{{.TokenValidTo}}` | (generated) | - | - |

`BS_USERNAME` is the BSN.cloud user email (e.g. `user@brightsign.biz`), which
is distinct from `BS_CLIENT_ID` (the OAuth client ID used for authentication).

`PackageName`, `DeviceName`, and `DeviceDescription` come only from CLI flags.

The token fields are always dynamically generated; they cannot be overridden
via flags or environment variables.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--username` | `BS_USERNAME` env | BSN.cloud username/email for bDeploy.username |
| `--network` | `BS_NETWORK` env | Network name (required) |
| `--package-name` | (none) | Setup package name (required, CLI only) |
| `--setup-type` | `bsn` | Setup type (`bsn`, `standalone`, `lfn`) |
| `--device-name` | `""` | Device name (CLI only) |
| `--device-description` | `""` | Device description (CLI only) |
| `--group` | `Default` | BSN group name |
| `--timeout` | 30 | HTTP request timeout in seconds |
| `--verbose` | false | Show extra details on stderr |
| `--help` | false | Display usage information |

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `BS_CLIENT_ID` | Yes | BSN.cloud OAuth client ID (for authentication only) |
| `BS_SECRET` | Yes | BSN.cloud API client secret (for authentication only) |
| `BS_USERNAME` | Yes (if no flag) | BSN.cloud username/email for bDeploy.username |
| `BS_NETWORK` | Yes (if no flag) | Network name |
| `BS_SETUP_TYPE` | No | Setup type override |
| `BS_GROUP_NAME` | No | BSN group name |

### Output

The complete rendered JSON is written to stdout. All status/progress messages
go to stderr so the output can be redirected cleanly.

### Exit Codes

- 0: success
- 1: any error (missing required values, auth failure, template error, invalid JSON)

---

## Content, Presentation and Schedule Examples

Twelve small tools cover the Content API: `bsn-content-{list,get,delete,upload,download}`,
`bsn-presentation-{list,get,create,delete}` and `bsn-schedule-{list,add,delete}`. They share one
behaviour, described here once, and each entry below lists only what is specific to it. The shared
plumbing lives in `examples/internal/cli`.

### Common behaviour

| Flag | Default | Description |
|---|---|---|
| `--network` / `-n` | `BS_NETWORK` env | Network name (required, from the flag or the env var) |
| `--timeout` | 30 | HTTP request timeout in seconds |
| `--json` | true | Write the result as JSON to stdout; `--json=false` writes a short tab-separated summary |
| `--verbose` | false | Show extra details on stderr |
| `--help` | false | Display usage information |
| `--force` / `-y` | false | Delete tools only: skip the confirmation prompt |

| Variable | Required | Description |
|---|---|---|
| `BS_CLIENT_ID` | Yes | BSN.cloud API client ID |
| `BS_SECRET` | Yes | BSN.cloud API client secret |
| `BS_NETWORK` | Yes (if no flag) | Network name |

Status and progress go to stderr; stdout carries only the result. Exit codes: 0 success, 1 any error
(bad flags, auth failure, API error, declined confirmation).

### bsn-content-list

Lists content in the library. `--folder` lists one virtual folder, `--filter`, `--sort`,
`--page-size` and `--marker` shape one page, and `--all` follows pagination to return everything
(not combinable with `--folder`). Calls `Content.List`, `Content.ListFolder` or `Content.ListAll`.

### bsn-content-get

Prints one item's metadata. `--id` (required). Calls `Content.GetByID`.

### bsn-content-delete

Deletes content. Exactly one of `--id` (one item) or `--filter` (everything the expression matches).
Asks first unless `--force`. Calls `Content.DeleteByID` or `Content.Delete`.

### bsn-content-upload

Uploads a local file through the chunked Upload API.

| Flag | Default | Description |
|---|---|---|
| `--file` | (required) | Local file to upload |
| `--path` | root | Virtual folder, either slash style; converted to BSN's backslash form |
| `--media-type` | detected | Override the media type otherwise detected from the extension |

Prints the content ID and virtual path. A file BSN already holds at that path is reused without
sending any bytes. Calls `Upload.Upload`.

### bsn-content-download

Downloads a file (two steps: read the item's URL, then fetch it) or, with `--thumbnail`, its
thumbnail (`--max-width`, `--max-height`).

| Flag | Default | Description |
|---|---|---|
| `--id` | (required) | Content ID |
| `--output` | (required) | File to write, or `-` for stdout |
| `--overwrite` | false | Replace `--output` if it exists; otherwise an existing file is an error |

With `--output -` the bytes own stdout and no JSON summary is written. A failed download removes its
partial file. Calls `Content.DownloadTo` or `Content.GetThumbnail`.

### bsn-presentation-list

Lists presentations: `--filter`, `--sort`, `--page-size`, `--marker`, and `--all` to follow
pagination. Calls `Presentations.List` or `Presentations.ListAll`.

### bsn-presentation-get

Prints one presentation. Exactly one of `--id` or `--name`. Calls `Presentations.GetByID` or
`Presentations.GetByName`.

### bsn-presentation-create

Creates a presentation. `--name` alone makes an empty shell; `--from-json FILE` supplies a full
`PresentationCreateRequest` (unknown keys are rejected), and `--name`, `--device-model`,
`--language`, `--status` override the file. Calls `Presentations.Create`.

### bsn-presentation-delete

Deletes presentations. Exactly one of `--id`, `--name` or `--filter`. Asks first unless `--force`.
A presentation still scheduled on a group must be unscheduled first. Calls
`Presentations.DeleteByID`, `DeleteByName` or `DeleteByFilter`.

### bsn-schedule-list

Lists a group's scheduled presentations. `--group` (required). Calls `Schedules.GetGroupSchedule`.

### bsn-schedule-add

Schedules a presentation on a group. `--group` (required) and exactly one of `--presentation-id` or
`--presentation-name`. Defaults to a recurrent, all-day (`--start 00:00:00 --duration 23:59:59`),
every-day (`--days EveryDay`) schedule; `--recurrence-start`/`--recurrence-end` (YYYY-MM-DD) bound it,
and `--recurrent=false --event-date YYYY-MM-DD` makes a one-off. `--interrupt` interrupts current
playback. Prints the created schedule, including its ID. Calls `Schedules.AddScheduledPresentation`.

### bsn-schedule-delete

Removes one schedule entry. `--group` and `--id` (both required). Asks first unless `--force`. Calls
`Schedules.DeleteScheduledPresentation`.
