# Fix-me: code review findings (2026-10-18)

Findings from a `high`-effort code review of the uncommitted changes that
fixed `Subscriptions.GetCount`, reworked error classification to use
`errors.As`, and removed the non-existent `Devices.GetOperations` endpoint.
Ordered by severity. Each item below is addressed by a corresponding section
in "Revised design and fixes".

## 1. `errors.As` wrapping made decode failures look like retryable network errors

**File:** `internal/errors/errors.go`, `internal/http/http.go`

Service methods now wrap the HTTP layer's error with `fmt.Errorf("...: %w",
err)` so predicates can see through to the real cause via `errors.As`. But the
HTTP layer's `Do()` (and three sibling methods) already collapses every
`request.Execute()` failure into `NewNetworkError`, including the case where a
response WAS received and only its body failed to decode into the caller's
result type. Before this change, the service layer discarded that typed cause
behind a status-0 `APIError`, so the predicates returned `false` by accident.
Now the wrapping passes the real `NetworkError` through, so a decode failure
is classified as both `IsNetworkError() == true` and `IsRetryableError() ==
true`.

**Why it matters:** the client's own retry condition
(`internal/http/http.go`, documented at the `AddRetryCondition` callback)
already refuses to retry a decode failure — it was added after a MEASURED
~0.3s→~7s regression on `GET /v1/system` from a firmware field returning a
string where an int was expected. A caller that builds its own retry/error
handling on `IsRetryableError`/`IsNetworkError` now reaches the same bad
outcome the SDK's internal retry logic was built to avoid, and is told "network
problem" when the real problem is response shape.

## 2. `.token` credential file is untracked but not ignored

**File:** `.token` (repo root)

A 2166-byte file that looks like a bearer/JWT credential sits in the repo
root. `git check-ignore .token` confirms `.gitignore` does not cover it. It is
currently untracked only because no one has run `git add -A` since it
appeared.

**Why it matters:** the next broad `git add` commits a live credential to
history. Violates the repo's own rule (`~/.claude/CLAUDE.md`): "Never commit
API keys, passwords, or credentials."

## 3. Removing `GetOperations` is a breaking public API change

**File:** `bscontrol.go`, `internal/services/devices.go`

`DeviceService.GetOperations`/`GetOperationsBySerial` and the public type
aliases `DeviceOperation`/`DeviceOperationList` were deleted outright. The
methods never worked (the endpoint does not exist at any API version — see
`docs/fixme.md` history / the canonical-sources research that drove the
removal), so no caller could have been using them successfully. Still, this is
a source-breaking change for anyone who compiled against the old surface
(including a mock implementing `DeviceService`).

## 4. Stale example counts and listings in docs

**Files:** `README.md` (lines 58, 336, 345), `examples/README.md` (line 251)

- `README.md` says "66 working example programs" / "a comprehensive Go SDK
  ... with 66 working example programs" in two places; the repo now has 65
  (`examples/bsn-device-operations` was removed).
- `README.md`'s category table lists "Account device management | 11 | ...,
  operations, ..." — the method and its example no longer exist.
- `examples/README.md` section header reads "## Device Management (9)" but
  lists 8 entries after the removal.

## 5. `SubscriptionCount.UnmarshalJSON` silently accepts `null`

**File:** `internal/types/types.go`

The custom unmarshaler added to decode both the bare-number and
`{"count": N}` response shapes falls through to `json.Unmarshal(data,
&count)` (an `int` target) for anything not starting with `{`. A JSON `null`
body does not start with `{`, and `json.Unmarshal([]byte("null"), &anInt)`
succeeds as a documented Go no-op, leaving `Count` at its zero value with no
error. The type's own doc comment promises "any other shape is a contract
violation and surfaces as an error" — `null` violates that promise silently.

**Why it matters:** `GET /Subscriptions/Count` returning `null` would make
`bsn-subscription-count` report zero subscriptions instead of failing loudly,
which is indistinguishable from "the network genuinely has zero
subscriptions."

## 6. `GetErrorsBySerial`'s 404 fallback uses a direct type assertion

**File:** `internal/services/devices.go` (around the `GetErrorsBySerial`
404-to-empty-list special case)

Every predicate in `internal/errors` was migrated to `errors.As` specifically
so a wrapped error is still classified correctly. This one call site still
does `err.(*errors.APIError)`, a raw type assertion that fails silently (hits
the `ok == false` branch) the moment anything upstream starts wrapping this
particular error. It happens to work today only because `GetWithAuth` returns
the HTTP layer's error unwrapped at this exact point — nothing wraps it yet.

**Why it matters:** this is exactly the fragility the project just spent 77
call-site conversions eliminating elsewhere. Any future refactor that adds a
layer of wrapping before this check (consistent with the rest of the
codebase's style) silently breaks the 404-means-empty-list behavior.

## 7. Pattern-2 "envelope" errors still report a fake `StatusCode: 0`

**Files:** `internal/services/bdeploy.go` (6 sites), `internal/services/
provisioning.go` (1 site), `internal/services/rdws.go` (1 site,
`GetCrashDump`)

These 8 sites were deliberately left out of the `fmt.Errorf("%w", ...)`
conversion because they don't wrap an HTTP-layer error — they detect an
application-level error embedded in an otherwise successful (2xx) response
body (a B-Deploy JSON envelope's own `error` field, an empty token field, a
crash-dump result that decodes as an error string). They still construct
`errors.NewAPIError(0, ...)`, which means they're represented as the same
`*APIError` type the status-code predicates inspect, but with a status code
that was never real.

**Why it matters:** `StatusCode: 0` is not a meaningful HTTP status — it's
the zero value. A caller inspecting `apiErr.StatusCode` for one of these
errors sees `0`, which reads as "no status available" and is
indistinguishable from an uninitialized error. These are a genuinely
different error category (an application/business error riding inside a
successful transport response) and deserve their own type rather than
borrowing `APIError`'s shape with a sentinel that was never a real status.

## 8. Duplicated serial number in wrapped `rdws_supervisor.go` error messages

**File:** `internal/services/rdws_supervisor.go`

`GetStoredSupervisors` and `GetCrashDumpFiles` both call `ListFiles`, which
already wraps its own failure as `"failed to list files for device with
serial '%s' at path '%s': %w"`. Both callers then wrap that again with their
own `"...on device with serial '%s': %w"`, so the final message repeats the
serial number twice, e.g.:

```
failed to list stored supervisors on device with serial 'X': failed to list
files for device with serial 'X' at path 'sys/supervisors': <cause>
```

**Why it matters:** noisy, harder-to-read error messages in CLI output and
logs. Minor, but free to fix alongside the other error-message work in this
pass.

---

## Revised design and fixes

1. **Decode-vs-network classification** (item 1): add a new error type,
   `DecodeError`, in `internal/errors/errors.go`, for "a response was
   received but its body did not decode into the caller's type." It
   deliberately implements no predicate match (not retryable, not a network
   error) by omission — no predicate needs to change, since `errors.As`
   against `*NetworkError` or `*APIError` simply won't match a `*DecodeError`.
   In `internal/http/http.go`, add a helper that classifies an
   `Execute`/`Post`-family error by checking whether a response was actually
   received (`resp != nil && resp.StatusCode() != 0`, the same test the retry
   condition already uses) and returns `DecodeError` instead of `NetworkError`
   in that case. Applied at `Do`, `PostForm`, and `PostFormWithAuth`, each of
   which calls `SetResult` and can genuinely hit this ambiguity (verified with
   a failing-then-passing test at each). `GetBytesWithAuth` sets no `Result` —
   confirmed by test that an unparseable error body there does not make resty
   return `err != nil` at all (it falls through to the existing
   `handleAPIError` path with a zero-valued error struct) — so the helper is
   applied there only for defensive consistency, not because a bug was
   reproduced.

2. **`.token` ignored** (item 2): add `.token` to `.gitignore`.

3. **Breaking API change** (item 3): documented here as an intentional,
   already-decided breaking change (the methods could never succeed against
   the real API — see the canonical-sources research in the session that
   produced this diff). No code change; noted for the release notes / version
   bump decision when this ships.

4. **Stale docs** (item 4): update `README.md` (66→65, the "11" row, drop
   "operations" from the device-management example list) and
   `examples/README.md` ("(9)"→"(8)").

5. **`null` handling** (item 5): `SubscriptionCount.UnmarshalJSON` explicitly
   rejects a `null` (or any non-numeric, non-object) body with an error
   instead of silently decoding it as `0`.

6. **404 fallback hardening** (item 6): change the direct type assertion in
   `GetErrorsBySerial` to `errors.As`, matching every other predicate in the
   codebase. Behavior-preserving refactor (verified by the full test suite
   staying green, per the Hippocratic-oath refactoring standard — there is no
   active bug today since nothing wraps the error at that point yet).

7. **Envelope errors need their own type** (item 7): add `AppError` to
   `internal/errors/errors.go` for an application-level error that arrived
   inside a successful transport response — no `StatusCode` field at all,
   since it would always be a lie. Replace all 8 Pattern-2
   `NewAPIError(0, ...)` call sites with `NewAppError(...)`.

8. **De-duplicate serial in wrapped messages** (item 8): `GetStoredSupervisors`
   and `GetCrashDumpFiles` drop the redundant `"on device with serial
   '%s'"` from their own wrap, since `ListFiles`'s error already carries the
   serial and the path.
