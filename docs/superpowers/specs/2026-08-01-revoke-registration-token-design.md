# Design: Revoke Registration Token

Date: 2026-08-01

## Problem

bscontrol implements two of the three BSN.cloud Provisioning token endpoints:

| Endpoint | Method | Status |
| --- | --- | --- |
| `POST /2022/06/REST/Provisioning/Setups/Tokens/` | `GenerateDeviceToken` | Implemented |
| `GET /2022/06/REST/Provisioning/Setups/Tokens/{token}/` | `ValidateDeviceToken` | Implemented |
| `DELETE /2022/06/REST/Provisioning/Setups/Tokens/{token}/` | — | Missing |

`docs/all-apis.md` already tracks the DELETE endpoint as `[NOT-DONE]`. There is no
way to revoke a registration token through the SDK or any CLI in this repo.

## Goal

Close the gap: a library method, a CLI binary, tests, and updated coverage docs.

## Non-Goals

- Revoke-all / bulk revoke. BSN.cloud exposes no endpoint that lists tokens, so
  there is nothing to enumerate.
- Lookup by anything other than the token value itself, for the same reason.
- Introducing HTTP mocking to the test suite. No service in this repo mocks HTTP;
  doing it for one method would be an inconsistent new pattern.

## Design

### Service method

`internal/services/provisioning.go`

Add to the `ProvisioningService` interface, following its two siblings:

```go
RevokeRegistrationToken(ctx context.Context, token string) error
```

The implementation mirrors `ValidateDeviceToken`: same guard sequence, same URL
builder, `DeleteWithAuth` in place of `GetWithAuth`.

Three decisions worth recording:

1. **Returns bare `error`, not `(bool, error)`.** Matches `deviceService.Delete`
   (`internal/services/devices.go:245`). The `(bool, error)` shape used across the
   rDWS service exists because rDWS responses wrap a `result.success` field; this
   endpoint returns no such body.
2. **Passes `nil` as the result to `DeleteWithAuth`.** Same as
   `internal/services/devices.go:270` — a 2xx with no body means success. If the
   API does return a body, it is discarded harmlessly.
3. **Does not call `EnsureNetworkSet`.** Neither `GenerateDeviceToken` nor
   `ValidateDeviceToken` calls it; network context rides in the OAuth token.
   Consistency beats inventing a rule for one method.

No new types. Nothing is unmarshalled, so `internal/types/types.go` and the alias
block in `bscontrol.go` are untouched. `client.Provisioning.RevokeRegistrationToken`
becomes reachable as soon as the interface method exists.

### CLI

`examples/bsn-regtoken-revoke/` — named to pair with the existing
`main-get-regtoken`. The Makefile globs `examples/*/` (`Makefile:12`), so
`bin/bsn-regtoken-revoke` builds with no Makefile change.

Flags:

| Flag | Meaning |
| --- | --- |
| `--token <value>` | Token to revoke (required) |
| `--force` | Skip the confirmation prompt |
| `--json` | Machine-readable output; implies non-interactive |
| `--verbose` | Detailed output |
| `--timeout <sec>` | Request timeout, default 30 |
| `--network`, `-n` | Override `BS_NETWORK` |

Flow: authenticate, set network context, and then — unless `--force` or `--json` —
call `ValidateDeviceToken` first and print the token's scope and validity window so
the operator sees what they are about to destroy, prompt, and only then revoke.

The fetch-then-confirm shape is taken from `examples/bsn-bdeploy-setup-delete/main.go:215-227`,
which reads the record before prompting. The extra round trip earns its keep here
because `cert`-scope tokens are shared by every player provisioned with them: an
unconfirmed revoke can break registration for a whole fleet.

### Tests

`internal/services/provisioning_test.go` (new file — this service currently has no
tests at all). Validation-only, matching the style of `internal/services/rdws_test.go`:

- `TestProvisioningServiceInterface` — compile-time interface satisfaction check.
- `TestProvisioningService_RevokeRegistrationToken` — empty token yields a
  validation error; an unauthenticated call yields an error.

This is the honest ceiling without a mock server. See Non-Goals.

### Documentation

- `docs/all-apis.md` — flip the DELETE line from `[NOT-DONE]` to `[DONE]` and cite
  `main-revoke-regtoken` as its example.
- `docs/all-apis.md` — Provisioning coverage `2/3 (67%)` becomes `3/3 (100%)`.
- `README.md` — B-Deploy Provisioning example count `13` becomes `14`.
- `docs/all-apis.md` — the POST and GET lines currently credit `main-token-test`
  as their example, but that binary only inspects a local `.token` file and never
  calls the provisioning API. Correct both to `main-get-regtoken`.

## Error Handling

- Empty token: `errors.NewValidationError` before any network call.
- Auth failure: propagated unchanged from `authManager`.
- API failure: wrapped in `errors.NewAPIError(0, "token_revoke_failed", ...)`,
  matching the `token_generation_failed` / `token_validation_failed` siblings.
- CLI: `log.Fatalf` on any error, consistent with every other example binary.

## Verification

`make test` and `make build` both pass; `bin/bsn-regtoken-revoke --help` renders
the documented flags.
