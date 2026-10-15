# Device-transport and interaction latency tests

**Goal:** Measure the two latency domains that documents 1-3 do not reach:

- **§A — on-device / LAN transport** for `player-cloud-control` (the Capacitor app
  talking to BrightSign players over HTTP/UDP). This is where the audit found the
  single worst latency in the monorepo (a 21-attempt sequential command sweep), and it
  is invisible to Grafana, the bscontrol harness, and cloud-UI Playwright.
- **§B — interaction / render** for the heavy React screens. Documents 1-2 decompose
  page *load*; this track measures *interaction* latency (INP, long tasks, React
  commits) on the bulk-action screens where the audit found per-keystroke table
  rebuilds and a 700 ms re-render loop.

This is a build plan, in the same shape as document 2: new, self-contained test
tooling plus small, flag-gated instrumentation. It does not change application
behavior in production.

Prerequisite reading:
[`02-latency-instrumentation-and-isolation-tests.md`](./02-latency-instrumentation-and-isolation-tests.md),
[`03-comparison-and-coverage-gaps.md`](./03-comparison-and-coverage-gaps.md).
Finding IDs (C*, R*) are from `bs-client-ui/PERFORMANCE-ANALYSIS.md`.

---

## 0. The measurement model, extended

Document 2 decomposed a page load into `T_total = T_backend + T_network +
T_serialization + T_client`. Two actions in this product do not fit that model:

**A device action** (open a player's Actions) resolves against a player on the LAN, not
BSN.cloud. Decompose it as:

```
T_action = T_discovery_wait + T_endpoint_search + T_device_response + T_parse
```

where `T_endpoint_search` is the cost of *finding* the right port/endpoint (the audit's
C1), distinct from the device's own response time.

**An interaction** (type a character, toggle a row) decomposes as the browser's INP
model:

```
T_interaction = T_input_delay + T_processing + T_presentation
```

where `T_processing` is the React commit — the term the audit's R1/R2/R3 inflate.

The point of both is the same as document 2: separate a *transport/search* cost from a
*device/backend* cost, and a *render* cost from an *input* cost, so a single "it's
slow" splits into causes with different owners.

---

## A. Device-transport timing (player-cloud-control)

### A.1 What is being measured, and why it is separate

The audit's worst finding, **C1**, is a **3 ports × 7 endpoints = 21-attempt,
strictly sequential** HTTP sweep with no aggregate deadline
(`libs/mobile-services/src/lib/udp/player-command.service.ts:175-232`; timeout
constants `libs/mobile-services/src/lib/config/constants.ts:34-46`). The dependent
cost, **C2**, is a **~7 s discovery wait** on mount
(`apps/player-cloud-control/src/hooks/useUnifiedDevices.ts:73` →
`player-network-detection.service.ts:534,551`;
`cordova-zeroconf.service.ts:700` hard 5 s + `:710` cache clear).

The device's *own* response is usually fast; the latency is the **search** and the
**serial retries against unreachable candidates**. So the test must vary device
reachability deliberately, not just time a happy path.

### A.2 Two instruments

**(1) A standalone LAN-floor probe (Go).** A small program that reproduces the sweep's
call set against a known player IP and reports per-attempt timing — the pure
device+network floor, with no app or webview. Go is the right tool (matches the house
language and the bscontrol harness in document 2); build it through `make` only, with
the required `build`/`run-tests`/`clean` targets.

What it does, for a given IP:

- Issue the same candidate requests the service tries — ports `[8008, 8080, 80]` ×
  the 7 command endpoints — but **time each candidate independently and in parallel**,
  so you learn *which* port/endpoint actually answers and how fast, versus the app's
  serial 21-deep walk.
- Record, per candidate: `port, endpoint, connect_ms, response_ms, status, answered`.
- Derive two headline numbers: **`T_device_response`** (the fastest answering
  candidate) and **`T_endpoint_search_serial`** (the sum the app pays walking in its
  fixed order until that candidate — the cost C1 adds over the floor).

**(2) In-app transport instrumentation (flag-gated).** The probe cannot see the real
app's scheduling (discovery awaited before render, caching, concurrent scan loops).
Add timing inside the service layer, behind a debug flag so it is off in production:

- Wrap `getPlayerCommands` and the discovery entry points
  (`PlayerSyncService.syncPlayers`, `scanForPlayers`, `verifyDeviceConnectivity`) with
  `performance.now()` stamps at entry/exit, pushing records to a `window.__transportTimings`
  buffer (shape: `op, ip, t_start, t_end, duration_ms, attempts, outcome`).
- Stamp **per attempt** inside `tryFetchFromMultipleEndpoints` so the buffer shows the
  21-deep walk explicitly — attempt index, port, endpoint, elapsed, result.
- Stamp discovery: scan start, Zeroconf-resolved, UDP-resolved, scan complete, and the
  moment `useUnifiedDevices` actually renders devices — this quantifies C2's
  "await before first paint."

The attach points are the service methods the audit cites; no new architecture, just
timing hooks under an `if (DEBUG_TRANSPORT)` guard.

### A.3 The reachability matrix (the actual test)

Run both instruments against a controlled set, because the cost is reachability-
dependent:

| Scenario | Expected signal |
|---|---|
| Player **online, answers on the first candidate** (port 8008, first endpoint) | Near-floor; `T_endpoint_search ≈ 0` |
| Player **online, answers on the last candidate** (port 80, 7th endpoint) | Several stacked connect timeouts before success — C1's common case |
| Player **offline** (IP present, nothing listening) | Worst case: full 21 × timeout ≈ 100-210 s — confirms the C1 ceiling |
| Player **present but slow** (reachable, delayed response) | `T_device_response` large; isolates a genuine device problem from the search |
| **N players, mixed** (e.g. 5 online + 15 offline) | Discovery + the serial per-device `verifyAll` (C5, `player-connectivity.service.ts:59-96`) backlog; measures whether the 60 s verify loop drains within its interval |

Offline scenarios are produced by pointing at an IP on the LAN with no player (or a
firewalled port), not by unplugging hardware — repeatable and safe.

### A.4 Output and the decision rule

Emit JSON/CSV, one row per attempt and a per-scenario summary, diffable against the
Go floor:

- If `T_action` in the app is **approximately `T_device_response`** → transport is
  fine; the device itself is the cost (rare).
- If `T_action` is **dominated by `T_endpoint_search`** → **C1 confirmed**: the fix is
  to race candidates (`Promise.any`), impose one overall deadline, and persist the
  known-good port/endpoint per IP. The matrix row that answers on the last candidate
  tells you exactly how much the serial order costs.
- If **time-to-first-rendered-device** ≫ time-to-first-resolved-device → **C2
  confirmed**: render from cache immediately and scan in the background.
- If the `verifyAll` backlog exceeds its 60 s interval in the mixed-N scenario → **C5
  confirmed**: bound concurrency.

---

## B. Interaction / render profiling (bulk-action screens)

### B.1 What is being measured, and why document 2 misses it

Document 2's `T_client` is "last-byte-to-interactive" for a page *load*. The audit's
render findings are about *interaction after load*:

- **R1** — `PlayerTable` rebuilds its entire column array on every search keystroke and
  every checkbox toggle; search is undebounced
  (`apps/control-panel-ui/src/player-actions/components/PlayerTable.tsx:181-333`;
  `PlayersFilterSidebar.tsx:83`).
- **R2** — `PlayerJobContext` passes a new value object and `setProgress` every 700 ms,
  re-rendering the whole wizard subtree
  (`player-actions/context/PlayerJobContext.tsx:240,291-318`).
- **R3/R4** — the shared `common-ui` Table (`libs/common-ui/.../table/index.tsx:89`)
  and `OrganizationMembersTable` (`.../members/OrganizationMembersTable.tsx:60`)
  re-render every row/cell on unrelated state changes.

These show up as **long tasks and poor INP**, never in a load waterfall.

### B.2 Three instruments

**(1) React Profiler API, flag-gated.** Wrap the suspect subtrees in `<Profiler>` and
log `onRender(id, phase, actualDuration, baseDuration, commitTime)` to a
`window.__renderTimings` buffer. Wrap at least: the player-actions `PlayerTable`, the
`PlayerJobProvider` subtree, and one `common-ui` Table instance. The ratio of commit
*count* to user actions is the headline — one keystroke should cause one commit of a
bounded subtree, not a full-table re-init.

**(2) `PerformanceObserver` for `longtask` and `event` (INP).** In the page or the
Playwright script, observe:

- `entryType: 'longtask'` — any task > 50 ms on the main thread during interaction.
- `entryType: 'event'` with `durationThreshold` — the Event Timing API, which yields
  INP (input delay + processing + presentation) per interaction.

This is the browser-native confirmation of the Profiler numbers and needs no React
internals.

**(3) `why-did-you-render` in dev (optional).** Drop-in for development only; it logs
the exact prop/state identity change that caused each avoidable re-render — it will
name the new-object-every-render context values (R2/R3) directly. Never shipped.

### B.3 Playwright interaction scripts

Repeatable, logged-in scripts that drive the *actions*, not just load the page:

- **Type-in-filter:** load the player-actions screen with a seeded fleet, type a
  10-character query into the search, record INP per keystroke and long-task count.
  Expect R1 to show one long task per keystroke that scales with fleet size.
- **Toggle-selection:** check/uncheck rows, record commit count and INP. Expect R1/R3
  to re-render the whole table per toggle.
- **Job-progress:** start a (simulated) bulk job and sit on the screen for 10 s;
  count commits. Expect R2 to show ~14 full-subtree commits (700 ms cadence) with no
  user input.
- Aggregate to `p50/p90/p99` INP and `long_tasks_per_interaction`, the same
  distribution shape as documents 1-2.

Seed the fleet at several sizes (e.g. 25 / 200 / 1000) — the audit's findings are
fleet-size-sensitive, and the slope is the proof.

### B.4 Decision rule

- **INP processing term large and commit count ≫ 1 per action** → a render problem:
  apply the audit's R1/R2/R3 fixes (decouple columns from selection, debounce search,
  `useMemo`/split the context values) and re-measure the same scripts.
- **INP input-delay term large but processing small** → the main thread is busy with
  something else (a concurrent poll or parse), not the component — look at the polling
  loops (D7) and XML parse (C7), not the table.
- **Long tasks only at the largest fleet size** → virtualization (R5) is the lever,
  not memoization.

---

## C. Sequencing and how this joins documents 1-2

Run after the cloud-UI tracks have localized the cloud pages, or in parallel if the
reported slowness is clearly the mobile app or a specific interaction.

1. **§A device-transport** — only track that can see the audit's worst finding (C1).
   Start here if "slow action" means the mobile/player experience.
2. **§B interaction/render** — start here if "slow action" means typing, sorting, or
   selecting in the bulk-action screens.
3. Feed confirmed items back to `bs-client-ui/PERFORMANCE-ANALYSIS.md` §8 (the
   prioritized roadmap) and fix in that order.
4. Where an interaction turns out to be main-thread contention rather than render
   (§B.4, second branch), cross-reference the polling/parse findings (D7, C6, C7)
   rather than the component.

Combined with documents 1-2, the four documents now cover every term of both models:
backend (1, 2§A), network/serialization/page-render (2§B), **device transport (4§A)**,
and **interaction render (4§B)**.

---

## Appendix: attach points and file reference index

**player-cloud-control / mobile-services (transport)**
- `libs/mobile-services/src/lib/udp/player-command.service.ts:175-232` — the 21-attempt sweep (C1); per-attempt stamp point.
- `libs/mobile-services/src/lib/config/constants.ts:34-46` — connect/read timeout constants.
- `apps/player-cloud-control/src/hooks/useUnifiedDevices.ts:73` — awaits `initialize()` before render (C2).
- `libs/mobile-services/.../player-network-detection/player-network-detection.service.ts:422,534,551` — discovery scan (serial Zeroconf then UDP).
- `libs/mobile-services/.../zeroconf/cordova-zeroconf.service.ts:700,710` — hard 5 s wait + cache clear (C3).
- `libs/mobile-services/.../player-connectivity/player-connectivity.service.ts:59-96` — serial `verifyAll` (C5).
- `libs/mobile-services/.../brightsign-xml-parser.ts:16,65` — synchronous XML parse, 1 MB cap (C7).

**control-panel-ui (interaction/render)**
- `apps/control-panel-ui/src/player-actions/components/PlayerTable.tsx:181-333` — per-keystroke/per-toggle column rebuild (R1).
- `apps/control-panel-ui/src/player-actions/components/PlayersFilterSidebar.tsx:83` — undebounced search.
- `apps/control-panel-ui/src/player-actions/context/PlayerJobContext.tsx:240,291-318` — 700 ms re-render loop, unmemoized value (R2).
- `apps/control-panel-ui/src/app/[locale]/user/organizations/[id]/_components/members/OrganizationMembersTable.tsx:60` — inline columns (R4).

**common-ui (shared render amplifier)**
- `libs/common-ui/src/lib/components/table/index.tsx:89` — fresh context value every render (R3).
- `libs/common-ui/src/lib/components/user-interaction/accordion/index.tsx:71` — same pattern.

**Build/run:** both the Go LAN probe and any Node aggregation go through `make`
(house rule — never `go build`/`go test`/`vitest` directly). Add targets alongside the
existing `build`/`run-tests`/`clean`.
