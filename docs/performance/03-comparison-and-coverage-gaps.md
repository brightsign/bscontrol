# Comparison and coverage gaps: front-end audit vs. the measurement plan

**Goal:** Reconcile the static front-end latency audit of `bs-client-ui` with the
measurement methodology in documents 1 and 2, state where they corroborate each
other, and name what neither currently covers. This is the map that says which
document to trust for which question — and where the current plan is blind.

Prerequisite reading:
[`01-backend-latency-from-existing-observability.md`](./01-backend-latency-from-existing-observability.md),
[`02-latency-instrumentation-and-isolation-tests.md`](./02-latency-instrumentation-and-isolation-tests.md).

Companion artifact: the front-end audit lives in the `bsn-core` repo at
`bs-client-ui/PERFORMANCE-ANALYSIS.md` (seven-dimension static code audit across all
three apps). Finding IDs below (D*, R*, B*, C*, BN*) are that document's.

---

## 1. The two methods, and why they are complementary

| | Front-end audit (`PERFORMANCE-ANALYSIS.md`) | Documents 1 + 2 |
|---|---|---|
| Method | Static code audit, breadth-first | Empirical measurement, hypothesis-to-confirm |
| Output | Enumerated candidate causes + fixes, cited `file:line` | PromQL runbook + a build plan (Go harness, Playwright, `Server-Timing`) |
| Apps covered | All three (control-panel-ui, player-cloud-control, mc-ui) | Almost entirely **mc-ui**; control-panel-ui set aside; **player-cloud-control absent** |
| Backend | Blind — inferred from the client | **Visible** — two .NET stacks, OTel, legacy blind spot, suspect SQL |
| Core weakness | Cannot say which candidate actually dominates | Scoped narrow; framed around page *load*, not interaction |

Documents 1 and 2 were evidently written **from** a front-end analysis of this kind:
they cite the same files the audit flags — `main.tsx:27-41` (react-query config),
`libs/domain/src/server/api/rest/utils.ts:7-20` (interceptor hub),
`libs/keycloak/.../client.ts:17` (`updateToken(5)`),
`DeviceListTableBody.tsx:18` (unvirtualized rows), and the Faro
`SessionInstrumentation`-only detail. So the right mental model is:

> **The audit is the hypothesis list. Documents 1-2 are the apparatus to confirm and
> quantify it — for the cloud UI and the backend.** The gaps below are where that
> apparatus does not yet reach.

---

## 2. Where they corroborate (trust these conclusions)

Independent agreement between a static audit and a measurement plan is the strongest
signal available before any number is pulled. Both land on the same mc-ui findings:

| Finding | Audit ID | Documents 1-2 |
|---|---|---|
| Bootstrap waterfall: Keycloak token → `self/networks` → `self/session/network` → `playersStatus`, serial, `enabled`-gated | D6 | 02 §1 (the mermaid chain) |
| `updateToken(5)` blocking inside the axios request interceptor | §3 data layer | 02 B.1; appendix `client.ts:17` |
| mc-ui QueryClient: no `staleTime`, `refetchOnWindowFocus` left on | D1 | appendix `main.tsx:27-41` |
| DeviceList: no virtualization, `getFacetedUniqueValues` + sort/filter over the full fleet, Radix tooltip per row | R5 | 02 B.4 |
| Dashboard `usePlayersStatus` unmemoized filter/map/sort | R6/R8 | 02 B.4 |
| Faro present but only `SessionInstrumentation` | §1 (ruled out as overhead) | 02 B.5 (named as an unused measurement hook) |

The mc-ui conclusions can be acted on with confidence; the only open question there is
*magnitude*, which document 2's harness answers.

---

## 3. What documents 1-2 add that the audit cannot

These are the measurement plan's genuine strengths — the things no front-end code read
could ever produce.

1. **Backend visibility.** The audit can say "no axios timeout → a hung endpoint
   freezes the action" (D4) but cannot name the endpoint or the cause. Document 1 pulls
   that from existing Grafana/OTel and surfaces what is invisible to any client audit:
   **two backend stacks**, a **legacy `BSN.WebServices` with zero observability**, and
   **suspect raw T-SQL** (`GetDevices` CROSS APPLY TVF per row; `GetDevicesSecure`
   `ROW_NUMBER` + correlated subqueries; a separate COUNT re-running the filter). This
   is the audit's single largest blind spot, fully covered here.
2. **An attribution model.** The audit ends at "go profile." Document 2 gives
   `T_total = T_backend + T_network + T_serialization + T_client` with a decision rule
   per term — a way to *prove* where time goes, not just enumerate suspects.
3. **The bscontrol Go harness (parallel vs. serial mode).** Quantifies the cost the
   front-end waterfall adds, with no browser, by running the page's call set both ways.
   This measures directly what D6 only asserts.
4. **The `Server-Timing` header as a correlation key** — and the sharp observation that
   it is the *only* lever for the legacy stack and for control-panel-ui's server-action
   latency.

---

## 4. Coverage gaps — what neither document measures (critique)

Each gap below is a place where the current measurement plan would return "nothing
wrong" or "no data" even though the audit found a concrete, often severe, cause.

### 4.1 player-cloud-control is entirely absent — the largest blind spot

If "slow on *some actions*" refers to the mobile/player experience, the backend-vs-
cloud-UI frame misses it completely. The audit's two worst findings live here and are
**on-LAN device communication**, not BSN.cloud:

- **C1** — `getPlayerCommands(ip)` runs **3 ports × 7 endpoints = 21 attempts,
  strictly sequential**, each with 5 s connect / 10 s read and **no aggregate
  deadline** → worst case ≈ **100-210 s** when a player is offline/slow/on the wrong
  port. Fires whenever a user opens a player's Actions.
  (`libs/mobile-services/src/lib/udp/player-command.service.ts:175-232`)
- **C2** — the device list **blocks ~7 s on discovery** before showing anything
  (`useUnifiedDevices` awaits a scan that runs a hard 5 s Zeroconf wait then a 2 s UDP
  wait, sequentially).

None of document 2's three sources can see this: Grafana watches BSN.cloud; the
bscontrol harness hits BSN.cloud, not players on the LAN; Playwright drives the cloud
UI, not the Capacitor app talking to a local device. **The worst latency in the
monorepo is outside the entire measurement plan.** Document 4 adds this track.

### 4.2 The plan measures page *load*; much of the pain is *interaction*

Document 1 opens with "slow page loads," and `T_total` is a per-*page* decomposition.
That does not capture:

- **R1** — `PlayerTable` rebuilds its entire TanStack column array on **every search
  keystroke and every checkbox toggle**, with an undebounced search
  (`PlayerTable.tsx:181-333`).
- **R2** — `PlayerJobContext` re-renders the whole wizard subtree **every 700 ms**
  during a job (`PlayerJobContext.tsx:240,291-318`).
- **R3** — the shared `common-ui` Table passes a fresh context value every render,
  re-rendering every row/cell (`libs/common-ui/.../table/index.tsx:89`).

Document 2 gestures at this with a single `PerformanceObserver longtask` line (B.4)
but does not treat render/interaction profiling as a first-class track. "Typing in the
bulk-action filter is janky" is an INP/long-task story, not a `T_total` story.
Document 4 adds this track too.

### 4.3 control-panel-ui's worst path is both unlisted and unmeasurable here

Document 2 correctly notes (B.5) that server actions hide upstream latency from the
browser — but it does not enumerate the audit's control-panel-ui findings as
hypotheses:

- **B1** — middleware on the Node runtime re-runs LaunchDarkly flags + `auth()`
  **~4× sequentially on every navigation** (`middleware.ts`;
  `services/featureFlags/featureFlags.ts:31-41`).
- **B2/B3** — the server component repeats the same flag + i18n work; a cold LD client
  can block render up to **10 s** (`ldClient.ts:22-31`).

Grafana (document 1) sees BSN.cloud, not the Next layer; browser Resource Timing
(document 2 §B) sees nothing behind the server action. Surfacing B1-B3 needs
**Next-layer server-side OTel**, which document 2 mentions only in passing (B.5). This
is a real plan gap, not just an audit extra.

### 4.4 The client config is never connected to backend load

Document 1 counts "the routes a page calls *on load*." But
`refetchOnWindowFocus:true` + `staleTime:0` (D1) means true request volume is far
higher than on-load — every tab-focus re-hammers every query. Document 1's volume
query (4.4) would *see* the inflated rate, but neither document ties it back to the
client config that causes it. When reading the PromQL volume numbers, subtract the
refetch-storm inflation or you will mis-attribute load to the backend.

### 4.5 A latent synergy both halves miss

The audit's **D5** (client discards pagination/filter and fetches the **whole
fleet** — `BsnPlayersApiClient.getPlayers` ignores its args) feeding document 1's
suspect **`GetDevicesSecure`** query is very likely **one combined defect**: an
unpaginated client call landing on an expensive legacy SQL path. Neither document
connects the two halves. It is a concrete, high-value joint target: fix the client to
send pagination *and* confirm the server route's SQL, together.

---

## 5. Decision matrix: which document answers which question

| Question | Use |
|---|---|
| Is a BSN.cloud route itself slow? | Doc 1 (Grafana p95) |
| Does the mc-ui bootstrap waterfall dominate? | Doc 2 §A (Go harness parallel vs serial) |
| Is it network / serialization / render on a cloud page? | Doc 2 §B (interceptor + Resource Timing + Playwright) |
| Why is opening a player's Actions slow (mobile)? | **Doc 4 §A (device-transport timing)** — not covered by 1-3 |
| Why is typing/sorting/selecting janky? | **Doc 4 §B (interaction/render profiling)** — not covered by 1-3 |
| Why is every control-panel-ui navigation slow? | Next-layer OTel (doc 2 B.5, under-specified) + the `Server-Timing` header (doc 2 §C) |
| Which specific code to change once measured | `PERFORMANCE-ANALYSIS.md` finding IDs |

---

## 6. Net assessment and recommended workflow

Documents 1-2 are **methodologically stronger and backend-aware**; the front-end audit
is **broader and interaction-aware**. They are complementary by construction, but the
measurement plan is scoped to the cloud-UI-load story and, as written, would miss the
mobile device-action latency (§4.1), the interaction jank (§4.2), and the
control-panel-ui middleware cost (§4.3).

Recommended sequence:

1. **Audit as the hypothesis list** — `PERFORMANCE-ANALYSIS.md`, all three apps.
2. **Document 1** — rule the BSN.cloud backend in or out cheaply, per route.
3. **Document 2 §A/§B** — quantify the mc-ui waterfall and split load/network/render.
4. **Document 4 (new)** — the two tracks the plan is missing: on-device/LAN transport
   timing for player-cloud-control, and render/interaction profiling for the
   bulk-action screens.
5. **Document 2 §C** — the `Server-Timing` header, once 1-4 confirm where to look; it
   is also the only route to control-panel-ui's server-action time.

See [`04-device-transport-and-interaction-latency-tests.md`](./04-device-transport-and-interaction-latency-tests.md)
for the two missing tracks.
