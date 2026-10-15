# Programs Missing --json Output Flag

Programs that do not yet support the `--json` flag for JSON-only output.

**Total:** 35 of 60 programs (58%)

**Programs with --json:** 25 of 60 programs (42%)

## Summary by Category

### B-Deploy Programs (10)
Programs for B-Deploy provisioning and device management:

- `bdeploy-add-setup` - Create new B-Deploy setup records
- `bdeploy-associate` - Associate devices with setup records
- `bdeploy-delete-device` - Delete devices from B-Deploy
- `bdeploy-delete-setup` - Delete B-Deploy setup records
- `bdeploy-find-device` - Find devices in B-Deploy
- `bdeploy-get-device` - Get B-Deploy device information
- `bdeploy-get-records` - Get B-Deploy records
- `bdeploy-list-devices` - List devices in B-Deploy
- `bdeploy-list-setups` - List B-Deploy setup records
- `bdeploy-update-setup` - Update B-Deploy setup records

### Main API Programs (11)
Programs for BSN.cloud main REST API:

- `main-auth-info` - Display authentication information
- `main-device-change-group` - Change device group assignment
- `main-device-delete` - Delete devices from network
- `main-device-errors` - List device errors
- `main-device-local-dws` - Manage device local DWS
- `main-group-delete` - Delete groups
- `main-group-update` - Update group properties
- `main--devices-list` - List devices on network
- `main-local-dws` - Manage local DWS settings
- `main-endpoints-test` - Test API endpoint connectivity
- `main-token-test` - Test and validate tokens

### RDWS Programs (14)
Programs for Remote Diagnostic Web Server API:

- `rdws-custom-data` - Send custom data to player via UDP
- `rdws-dws-password` - Get/set DWS password
- `rdws-files-create-folder` - Create folders on player storage
- `rdws-files-delete` - Delete files from player storage
- `rdws-files-rename` - Rename files on player storage
- `rdws-files-upload` - Upload files to player storage
- `rdws-firmware-download` - Download and apply firmware updates
- `rdws-local-dws` - Enable/disable local DWS
- `rdws-reboot` - Reboot player
- `rdws-reformat-storage` - Reformat player storage
- `rdws-registry-set` - Set player registry values
- `rdws-reprovision` - Re-provision player with B-Deploy
- `rdws-snapshot` - Take player screenshot
- `rdws-time-set` - Set player date/time

## Implementation Notes

### Benefits of --json Flag

1. **Scripting** - Easy parsing in shell scripts with `jq`
2. **Integration** - Clean integration with other tools/systems
3. **Automation** - Machine-readable output for CI/CD pipelines
4. **Consistency** - Uniform output format across all programs

### Typical Implementation Pattern

```go
var (
    jsonFlag = flag.Bool("json", false, "Output raw JSON response")
)

// ... later in code ...

if *jsonFlag {
    jsonData, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        log.Fatalf("Failed to marshal JSON: %v", err)
    }
    fmt.Println(string(jsonData))
} else {
    // Display formatted output
    displayResults(result)
}
```

### Programs That Have --json

Examples of programs that properly implement `--json`:

- `main-device-info` - Get device info with JSON output
- `main-device-status` - Get device status with JSON output
- `rdws-info` - Get player info with JSON output
- `rdws-health` - Get player health with JSON output
- `rdws-diagnostics` - Run diagnostics with JSON output
- `rdws-network-config` - Get network config with JSON output

See these programs for implementation examples.

## Priority for Implementation

### High Priority (Frequently Used)
1. `main--devices-list` - Most commonly used listing command
2. `bdeploy-list-setups` - Important for B-Deploy workflows
3. `bdeploy-list-devices` - B-Deploy device listing
4. `main-device-errors` - Critical for monitoring
5. `main-auth-info` - Useful for debugging auth

### Medium Priority (Data Modification)
6. `bdeploy-add-setup` - Setup creation
7. `bdeploy-update-setup` - Setup modification
8. `main-device-change-group` - Device management
9. `main-group-update` - Group management
10. `rdws-registry-set` - Registry modification

### Lower Priority (Simple Operations)
11. Delete operations (already have success/failure output)
12. File operations (upload, rename, delete)
13. Simple control operations (reboot, reprovision, snapshot)

## Related Documentation

- See `examples/*/main.go` for implementation examples
- Use `--help` flag on any program to see available options
- JSON output typically includes the full API response object
