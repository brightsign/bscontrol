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
