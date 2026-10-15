# bscontrol v2 migration — fix ledger

Branch: `v2`. Module path bumped to `github.com/brightsign/bscontrol`.

This v2 fixes the cross-cutting defects found in the purplectl review (2026-10-07). Per-endpoint
code is unchanged except where a defect lives in it. Decisions (confirmed with maintainer):

- **Module path:** `/v2` suffix (Go semver for breaking changes).
- **Env:** opt-in. `New()` no longer reads env. Callers add `WithEnv()` (loads
  `BS_CLIENT_ID`/`BS_SECRET`/`BS_NETWORK`) or pass values explicitly.
- **Scope:** core fixes + tests + migrate all examples so `make build` stays green.

## Defects being fixed

| # | Defect | Fix | Files |
|---|---|---|---|
| 1 | Re-auth wipes network; `EnsureNetworkSet` silently re-applies `BS_NETWORK` → cross-wire | Track `desiredNetwork` (survives re-auth); re-apply it after refresh; never silent-switch | `internal/auth/auth.go` |
| 2 | B-Deploy keeps its own `currentNetwork`, sets session independently → two network states | `SetNetworkContext` delegates session to `authManager`; read network name from it for the query filter | `internal/services/bdeploy.go` |
| 3 | `New()` reads env unconditionally | `WithEnv()` option; remove implicit `LoadFromEnv` | `bscontrol.go`, `internal/config/config.go` |
| 4 | No token-expiry getter; no pluggable token source | `ExpiresAt()` getter; `TokenSource` hook via `WithTokenSource` | `internal/auth/auth.go`, `bscontrol.go`, `internal/config/config.go` |
| 5 | 77 `fmt.Sprintf` URL builds, unescaped path/query → parameter injection | `rdwsURL()` helper using `url.PathEscape`/`url.Values`; convert call sites | `internal/services/rdws.go`, others |
| 6 | Retry condition retries 5xx for ALL methods, incl. non-idempotent | Retry only idempotent methods (GET/HEAD/PUT/DELETE/OPTIONS); POST/PATCH only on transport error | `internal/http/http.go` |
| 7 | Stale deps: `golang.org/x/net v0.17.0` | Bump deps (`go get -u`, tidy) | `go.mod` |
| 8 | Makefile missing `run-tests`/`check` (house rule) | Add targets | `Makefile` |

## Phase status

- [x] P0: Makefile targets (`run-tests`, `check`)
- [x] P1: Module bump to `/v2` (go.mod + all imports) — verify build/test green
- [x] P2: config.go — `WithEnv()`, remove implicit env, `WithTokenSource`
- [x] P3: auth.go — desired-network restore, `ExpiresAt()`, token source
- [x] P4: http.go — idempotent-only retry
- [x] P5: rDWS/devices/provisioning URL escaping (escQuery/escSegment/escPath); bdeploy already safe
- [x] P6: bdeploy.go — unified onto authManager (SetNetworkContext delegates; filter reads DesiredNetworkName)
- [x] P7: dep bump (x/net 0.17→0.44, resty 2.11→2.17, go floor held at 1.24)
- [x] P8: tests — auth network-restore + token source + ExpiresAt; retry idempotency; URL escaping; config WithEnv/TokenSource
- [x] P9: migrated all 63 example New() sites to /v2 + WithEnv()
- [x] P10: `make check` green; build green; race-clean

## Consumer migration (v1 → v2)

Import path:

    github.com/brightsign/bscontrol        ->  github.com/brightsign/bscontrol

Environment is now opt-in. `New()` no longer reads `BS_CLIENT_ID`/`BS_SECRET`/`BS_NETWORK`:

    // before
    client, _ := bscontrol.New()
    // after
    client, _ := bscontrol.New(bscontrol.WithEnv())

New capabilities:

- `bscontrol.WithTokenSource(ts)` — supply a caller-owned `TokenSource` (owns token
  caching/sharing); credentials then not required.
- `client.TokenExpiresAt() time.Time` — read token expiry without decoding the JWT.

Behavioral changes (no API change):

- The selected network survives token refresh and is never silently replaced by `BS_NETWORK`.
- Non-idempotent requests (POST/PATCH) are no longer retried.
- rDWS/device URLs percent-encode interpolated values (no-op for well-formed input).

## Live verification (GregoryCHerlein network, 2026-10-09)

Smoke-tested end-to-end against the GregoryCHerlein network (18 networks visible, 5 devices)
with the `bscontrol2` credentials. All green:

- Auth + token (`main-auth-info`), devices list (`main--devices-list`), cross-network find
  (`main-device-find`, 18 networks searched) — network correctly scoped to id 32776.
- B-Deploy path (`bdeploy-list-setups` 27 setups, `bdeploy-list-devices`) — all records scoped
  to GregoryCHerlein, confirming the unified network state (defect #2) live.
- Live rDWS against online player F4E5D2000838 (firmware 9.1.82): `rdws-info`,
  `rdws-files-list --path sd` (real `/sd` listing), `rdws-registry-get networking/hostname`
  (well-formed response). This confirms the per-segment `escPath`/`escSegment` encoding does
  not break well-formed paths on a live player.

### Still needs deliberate live verification

- Firmware download: the `url=` query value is now `url.QueryEscape`d (was raw). Confirm the
  server accepts a fully percent-encoded firmware URL (not tested — would trigger a real
  firmware download).
- Registry/file paths containing reserved characters: confirmed no-op for well-formed input;
  a key or path legitimately containing reserved characters still wants one deliberate test.
