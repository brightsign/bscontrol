# BCN-22133: rDWS passthrough is missing `GET /system/supervisors/` and `GET /logs/crash-dumps/`

Filed against the **BSN.cloud** backend (`brightsign/bsn-cloud`, `RdwsDeviceController` + gateway routing):
<https://brightsign.atlassian.net/browse/BCN-22133>. This file documents the
bscontrol-side workaround that exists because of the gap.

| Field | Value |
| --- | --- |
| Ticket | BCN-22133 |
| Project | BCN (Software) |
| Issue type | Defect |
| Assignee | Gareth Parris |
| Component | rDWS passthrough / `RdwsDeviceController` |
| Affects | All networks, all subscription tiers, all players |
| Priority | Medium (suggested -- no data loss, but two documented capabilities are unreachable) |
| Reporter | Eamon McNamee / Greg Herlein |
| Reported by SDK | bscontrol, commit `e1665cb` |
| Measured | 2026-08-07 |

---

## Summary

Two rDWS routes that exist in the player's local DWS API are not exposed by the
BSN.cloud rDWS passthrough. Any cloud client calling them gets a 404:

- `GET /system/supervisors/` -- list supervisor builds downloaded to the player
- `GET /logs/crash-dumps/` -- enumerate crash dumps the player is holding

These are not tier-gated, not player-specific, and not intermittent. The routes
were never implemented on the backend at all.

## Environment

- BSN.cloud rDWS passthrough (`{RDWSBaseURL}/...?destinationType=player&destinationName={serial}`)
- Networks exercised: `emac-test-network`, `us-canary-prod`
- Player: `UTD37F000049`
- Client: bscontrol Go SDK, `internal/services/rdws_supervisor.go`

## Steps to reproduce

```bash
# 1. Authenticate and set network context as usual for rDWS.
# 2. Call either route against any player on any network:

curl -H "Authorization: Bearer $TOKEN" \
  "$RDWS_BASE_URL/system/supervisors/?destinationType=player&destinationName=UTD37F000049"

curl -H "Authorization: Bearer $TOKEN" \
  "$RDWS_BASE_URL/logs/crash-dumps/?destinationType=player&destinationName=UTD37F000049"
```

**Expected:** the same payloads the player's local DWS returns -- a
`{ success, builds[] }` object for supervisors, and an array of
`{ fileName, ctime }` entries for crash dumps.

**Actual:** HTTP 404 for both, on every network tried.

## Evidence

Three independent confirmations, all on 2026-08-07:

1. **Gateway routing config.** Pulled `ntrada.yml` from `brightsign/bsn-cloud`.
   Zero matches for `system/supervisors` or `logs/crash-dumps`. The paths are
   not routed to the rDWS controller at all.
2. **Backend's own API reference.** `.github/docs/rdws-api-reference.md` in the
   same repo lists the implemented rDWS endpoints. Neither route appears.
3. **Live measurement.** Both routes 404 against `emac-test-network` and
   `us-canary-prod`, i.e. the failure is not a tier or environment artifact.

For contrast, `GET {RDWS_BASE_URL}/files/{path}/` (`ListFiles`) works correctly against the
same player in the same session, and returns the underlying data both missing
routes were supposed to surface (see workaround below).

## Impact

- Cloud clients cannot ask a player which supervisor builds it has downloaded.
  This is the observability half of a supervisor update: you can trigger
  `POST /update/sync`, but you cannot confirm what landed without falling back
  to a raw file listing.
- Cloud clients cannot cheaply answer "did anything crash since I last looked".
- Every SDK, tool, or service written against the documented DWS API surface
  will fail on these two calls with no indication that the route simply does not
  exist on the cloud side. The 404 is indistinguishable from a bad serial or a
  transient gateway problem.

## Current workaround (bscontrol `e1665cb`)

Both SDK methods were reimplemented on top of the already-working
`GET {RDWS_BASE_URL}/files/{path}/` route. The on-device locations were confirmed directly
against a live player (`UTD37F000049`):

| SDK method | Now calls | Extracts |
| --- | --- | --- |
| `GetStoredSupervisors` | `ListFiles(serial, "sys/supervisors")` | entries with `type == "dir"` (timestamp-named build directories, e.g. `2026-06-19T16-28-05.816Z`) |
| `GetCrashDumpFiles` | `ListFiles(serial, "sd/brightsign-dumps")` | entries with `type == "file"`, plus `stat.ctime` when present |

Public signatures and return types are unchanged, so no consumer code moved.
Verified against real hardware: both methods now return real data (one stored
supervisor build, one crash dump file) instead of a 404.

### Why the workaround is not a fix

- **Hardcoded on-device paths.** `sys/supervisors` and `sd/brightsign-dumps` are
  now baked into the SDK. If BrightSignOS relocates either, every client using
  this workaround breaks silently and returns an empty list rather than an error.
- **Storage-device assumption.** `sd/brightsign-dumps` assumes dumps land on the
  SD card. Players booting from other storage are not covered.
- **"Missing" and "empty" collapse.** `ListFiles` renders a path it cannot find
  as HTTP 200 with an empty result, so "no supervisors downloaded" and "the
  directory does not exist" are indistinguishable to the caller.
- **Device-reported errors are lost.** The removed `parseCrashDumpList` could
  surface a player-side error string from the real route. There is no equivalent
  signal in a file listing.
- **No server-side semantics.** The real routes could filter, sort, or exclude
  the firmware-bundled supervisor. A directory listing is a raw approximation
  that the SDK has to interpret by convention.
- **Two round trips of ambiguity.** Any consumer wanting real crash-dump
  metadata must follow up with additional file operations.

## Requested fix

Implement both routes in `RdwsDeviceController` and route them in the gateway
config, passing through to the player's local DWS:

- `GET /system/supervisors/` -> `{ "success": bool, "builds": [string] }`
- `GET /logs/crash-dumps/` -> `[{ "fileName": string, "ctime": string }]`

### Acceptance criteria

1. Both routes return 200 with the documented payloads against a player holding
   at least one stored supervisor build and at least one crash dump.
2. Both routes are present in `ntrada.yml` and in
   `.github/docs/rdws-api-reference.md`.
3. A player with no stored builds / no crash dumps returns an empty collection
   with 200, not a 404.
4. Player-side errors surface as a non-2xx or as the documented error payload,
   distinguishably from "nothing to report".
5. Behavior is identical across subscription tiers, or the tier requirement is
   documented explicitly.

Once shipped, bscontrol reverts `GetStoredSupervisors` and `GetCrashDumpFiles`
to the direct routes and deletes `storedSupervisorsFromFileList` /
`crashDumpEntriesFromFileList`.

## Related / unverified

- **`POST /system/supervisors/delete/`** is still called directly by
  `DeleteSupervisors` (`internal/services/rdws_supervisor.go:120`). It is in the
  same route family as the two confirmed-missing routes and has **not** been
  measured against a live player. It may have the same problem. Worth checking
  as part of this ticket.
- **`GET /system`** (`GetSystemInfo`) is used as an Internal-tier, undocumented
  route -- it is the only source for the running supervisor version. It works,
  but its status should be confirmed rather than assumed.
- Dead types left behind by the workaround, to be removed or restored depending
  on the outcome: `RDWSStoredSupervisorsResponse` and `RDWSCrashDumpListResponse`
  in `internal/types/types.go`.

## References

- bscontrol commit `e1665cb` -- `fix(rdws): implement GetStoredSupervisors/GetCrashDumpFiles via ListFiles`
- bscontrol commit `4beed97` -- `feat(rdws): add supervisor-update methods and fix GetLogs` (added the original, non-working calls)
- `internal/services/rdws_supervisor.go`, `internal/services/rdws_supervisor_test.go`
- `brightsign/bsn-cloud`: `ntrada.yml`, `.github/docs/rdws-api-reference.md`
