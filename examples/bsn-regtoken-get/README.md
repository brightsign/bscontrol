# Get Device Registration Token

A tool to generate a device registration token for BSN.cloud.

## Overview

Device registration tokens allow BrightSign players to register themselves with BSN.cloud. The token has 'cert' scope and is valid for 2 years by default. Multiple devices can use the same token.

This example demonstrates:
- Authenticating with BSN.cloud using OAuth2 client credentials
- Setting network context
- Generating a device registration token
- Displaying token details and usage information

## Prerequisites

- BSN.cloud API credentials (client ID and secret)
- Access to at least one BSN.cloud network
- Required scope: `bsn.api.main.devices.setups.token.create`

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `BS_CLIENT_ID` | Yes | BSN.cloud API client ID |
| `BS_SECRET` | Yes | BSN.cloud API client secret |
| `BS_NETWORK` | No | Default network name (can be overridden with `--network` flag) |

## Usage

### Basic Usage

Generate a registration token for a specific network:

```bash
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"

./bin/bsn-regtoken-get --network "My Network"
```

### Using Environment Variable

Set the network via environment variable:

```bash
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"
export BS_NETWORK="My Network"

./bin/bsn-regtoken-get
```

### Verbose Output

Show detailed information including the full token:

```bash
./bin/bsn-regtoken-get --network "My Network" --verbose
```

### JSON Output

Get machine-readable JSON output:

```bash
./bin/bsn-regtoken-get --network "My Network" --json
```

Save to file:

```bash
./bin/bsn-regtoken-get --network "My Network" --json > regtoken.json
```

## Command-Line Options

| Option | Alias | Default | Description |
|--------|-------|---------|-------------|
| `--help` | | false | Display usage information |
| `--network <name>` | `-n` | | Network name (overrides BS_NETWORK) |
| `--json` | | false | Output as JSON only |
| `--verbose` | | false | Show detailed information including full token |
| `--timeout <seconds>` | | 30 | Request timeout in seconds |

## Output Format

### Standard Output

The tool displays:
- Token details (network, scope, validity period)
- Token preview (or full token with `--verbose`)
- Usage instructions
- curl command examples
- Next steps

### JSON Output

When using `--json` flag, the output contains:

```json
{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "scope": "cert",
  "validFrom": "2024-05-21T00:00:00Z",
  "validTo": "2026-05-21T00:00:00Z"
}
```

## Token Details

- **Scope**: `cert` - Allows multiple devices to use the same token
- **Validity**: Typically 2 years from generation
- **Usage**: Embed in B-Deploy setup records to enable player registration

## API Endpoint

This tool calls:
- **Endpoint**: `POST https://api.bsn.cloud/2020/10/REST/Provisioning/Setups/Tokens/`
- **Required Scope**: `bsn.api.main.devices.setups.token.create`
- **Authentication**: OAuth2 Bearer token

## Examples

### Generate Token for Production Network

```bash
./bin/bsn-regtoken-get --network "Production" --verbose
```

### Get Token as JSON and Parse with jq

```bash
./bin/bsn-regtoken-get --network "Production" --json | jq -r '.token'
```

### Use Token in B-Deploy Setup

After generating a token, you can use it in a B-Deploy setup configuration:

```json
{
  "bDeploy": {
    "networkName": "My Network",
    "username": "admin@example.com",
    "packageName": "my-setup-v1"
  },
  "setupType": "lfn-ethernet",
  "bsnDeviceRegistrationTokenEntity": {
    "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "scope": "cert",
    "validFrom": "2024-05-21T00:00:00Z",
    "validTo": "2026-05-21T00:00:00Z"
  }
}
```

Then create the setup:

```bash
./bin/bsn-bdeploy-setup-add config.json
```

## Related Examples

- `bsn-bdeploy-setup-add` - Create a complete B-Deploy setup record (includes automatic token generation)
- `bsn-auth-info` - Get BSN.cloud authentication information
- `bsn-bdeploy-setup-list` - List all B-Deploy setup records

## Troubleshooting

### Error: "Failed to generate device registration token"

- Verify your API credentials are correct
- Ensure you have the required scope: `bsn.api.main.devices.setups.token.create`
- Check network name is spelled correctly
- Try with `--verbose` flag for more details

### Error: "Failed to set network context"

- Verify the network name exists in your BSN.cloud account
- Check that you have access to the specified network
- Use `bsn-auth-info` to list available networks

### Error: "Multiple networks available"

- Specify a network using `--network` flag
- Or set `BS_NETWORK` environment variable
- Run `bsn-auth-info` to see available networks

## Notes

- The generated token is valid for 2 years by default
- Multiple devices can use the same token
- The token has 'cert' scope which allows certificate-based registration
- Tokens are tied to a specific network
- If you don't provide a token when creating a B-Deploy setup, one will be auto-generated
