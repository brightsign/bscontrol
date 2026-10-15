# JSON Output Support Status

## Overview

This document tracks the implementation of `--json` flag support across all example programs. The goal is to ensure:
- JSON data goes to **stdout** (can be piped to other programs)
- Informational messages go to **stderr** (visible but don't interfere with piping)

## Completed Programs (42 programs)

### RDWS Programs with --json (22 programs)
- ✅ rdws-crashdump-get
- ✅ rdws-diagnostics
- ✅ rdws-dns-lookup
- ✅ rdws-files-list
- ✅ rdws-health
- ✅ rdws-info
- ✅ rdws-logs-get
- ✅ rdws-network-config
- ✅ rdws-network-neighborhood
- ✅ rdws-packet-capture
- ✅ rdws-ping
- ✅ rdws-reboot
- ✅ rdws-reformat-storage
- ✅ rdws-registry-get
- ✅ rdws-registry-set
- ✅ rdws-reprovision
- ✅ rdws-snapshot
- ✅ rdws-ssh
- ✅ rdws-telnet
- ✅ rdws-time
- ✅ rdws-time-set
- ✅ rdws-traceroute

### Main API Programs with --json (18 programs)
- ✅ main-content-delete
- ✅ main-content-list
- ✅ main-content-upload
- ✅ main-device-downloads
- ✅ main-device-info
- ✅ main-device-operations
- ✅ main-device-status
- ✅ main-group-info
- ✅ main-subscriptions-list
- ✅ main-presentation-count
- ✅ main-presentation-create
- ✅ main-presentation-delete-by-filter
- ✅ main-presentation-info
- ✅ main-presentation-info-by-name
- ✅ main-presentation-list
- ✅ main-presentation-update
- ✅ main-subscription-count
- ✅ main-subscription-operations

### BDeploy Programs with --json (1 program)
- ✅ bdeploy-get-setup

### Other Programs (1 program)
- ✅ main-content-download

**Total: 42 programs with working --json support**

---

## Programs Needing --json Support (31 programs)

### BDeploy Programs (10 programs)
1. ✅ bdeploy-add-setup - Outputs BDeploySetupResponse as JSON
2. ✅ bdeploy-associate - Outputs BDeployDevice as JSON
3. ✅ bdeploy-delete-device - Outputs success result as JSON
4. ✅ bdeploy-delete-setup - Outputs BDeploySetupResponse as JSON
5. ✅ bdeploy-find-device - Outputs search result with device/network as JSON
6. ✅ bdeploy-get-device - Outputs BDeployDevice as JSON
7. ✅ bdeploy-get-records - Outputs BDeployRecordsResponse as JSON
8. ✅ bdeploy-list-devices - Outputs BDeployDevicesResponse as JSON
9. ✅ bdeploy-list-setups - Outputs BDeployRecordsResponse as JSON
10. ✅ bdeploy-update-setup - Outputs BDeploySetupRecord as JSON

### Main API Programs (13 programs)
11. ✅ main-auth-info - Outputs auth info with token and networks as JSON
12. ✅ main-device-change-group - Outputs device and group change result as JSON
13. ✅ main-device-delete - Outputs deletion result as JSON
14. ✅ main-device-errors - Outputs device error list as JSON
15. ✅ main-device-local-dws - Outputs local DWS status/commands as JSON
16. ✅ main-group-delete - Outputs deletion result as JSON
17. ✅ main-group-update - Outputs updated group as JSON
18. ✅ main--devices-list - Outputs device list or specific device as JSON
19. ✅ main-local-dws - Outputs local DWS setup commands as JSON
20. ✅ main-presentation-delete - Outputs deletion result as JSON
21. ✅ main-endpoints-test - Outputs token and endpoint info as JSON
22. ✅ main-token-test - Outputs token analysis as JSON

=== All Main API programs complete (22/22) ===

### RDWS Programs (15 programs)
23. ✅ rdws-custom-data - Outputs custom data send result as JSON
24. ✅ rdws-dws-password - Outputs DWS password info or set result as JSON
25. ✅ rdws-files-create-folder - Outputs folder creation result as JSON
26. ✅ rdws-files-delete - Outputs file deletion result as JSON
27. ✅ rdws-files-rename - Outputs file rename result as JSON
28. ✅ rdws-files-upload - Outputs file upload result as JSON
29. ✅ rdws-firmware-download - Outputs firmware download initiation result as JSON
30. ✅ rdws-local-dws - Outputs local DWS status or set result as JSON
31. ✅ rdws-reboot - Outputs reboot request status as JSON (requires -y flag with --json)
32. ✅ rdws-reformat-storage - Outputs storage reformat result as JSON (requires -y flag with --json)
33. ✅ rdws-registry-set - Outputs registry operation result as JSON (requires -y flag with --json)
34. ✅ rdws-reprovision - Outputs reprovision status as JSON (requires -y flag with --json)
35. ✅ rdws-snapshot - Outputs snapshot result with base64 image data as JSON
36. ✅ rdws-time-set - Outputs time set result as JSON

**Total: 36 programs completed (1 program remaining: rdws-logs-get)**

---

## Implementation Progress

### Legend
- ✅ Complete and verified
- 🔧 In progress
- ⏳ Pending
- ❌ Skipped (not applicable)

---

## Notes

### Changes Made to Existing Programs
All programs with existing --json flags have been updated to:
1. Route JSON output to stdout using `json.NewEncoder(os.Stdout)`
2. Route informational messages to stderr using `fmt.Fprintf(os.Stderr, ...)`
3. Suppress informational messages when `--json` flag is set (using `!*jsonFlag` checks)
4. Fix network selection prompts to use stderr
5. Fix display/print functions to use stderr

### Implementation Pattern for New Programs
When adding --json support:
1. Add flag: `jsonFlag = flag.Bool("json", false, "Output as JSON")`
2. Import: Add `"encoding/json"` to imports if not present
3. Suppress messages: Wrap info messages with `if !*jsonFlag { ... }`
4. JSON output: Use `json.NewEncoder(os.Stdout).Encode(data)` for JSON mode
5. Regular output: Keep existing display logic for non-JSON mode
6. Update usage: Add example showing JSON usage
