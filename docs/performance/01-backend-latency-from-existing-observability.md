# Backend latency from existing observability (zero code change)

**Goal:** Before building or instrumenting anything, find out whether the backend is
even the cause of slow page loads. The new BSN.cloud stack already emits
per-request latency metrics and distributed traces. This document is a runbook for
pulling backend-only latency per endpoint from what already exists, so we can decide
whether the multi-second page loads come from the server or from the front-end.

If the backend p95 for the routes a page calls is small relative to the page's
total load time, the problem is in the front-end and you should move to
[`02-latency-instrumentation-and-isolation-tests.md`](./02-latency-instrumentation-and-isolation-tests.md).

---

## 1. What this can and cannot measure

There are **two coexisting "Main API" stacks**. A given route lands on exactly one:

| Stack | Projects | Observability | This runbook |
|---|---|---|---|
| **New** (.NET 9) | `bsn-cloud/` — Ntrada gateway, gRPC Authorizer, `devices-service` | Full OpenTelemetry: request-duration histograms, authorizer metrics, SQL spans | Fully covered |
| **Legacy** | `bsn-cloud-inherited/` — `BSN.WebServices` (Main/Admin/Self/Upload), raw ADO.NET | **None** | Blind spot — see section 6 |

So this document gives real numbers **only for routes already migrated to the new
stack** (device/player routes are). For legacy routes, the only path to a number is
the `Server-Timing` header in document 2, section C.

**What you get (new stack):** server-side wall-clock per route (p50/p90/p95/p99),
split across gateway vs backend service, the authorization cost, and DB time per
request from traces.

**What you do not get here:** network transit, front-end serialization, or browser
render time. Those are document 2.

---

## 2. Prerequisites

- Access to the Grafana instance backed by Mimir/Prometheus (OTLP metrics are
  exported via grafana-alloy). Ask the platform team for the org/folder if you do
  not already have it.
- The ability to run PromQL in Grafana Explore.
- Knowing which routes the target page calls. Use the call graphs in section 5, or
  the browser Network panel against a live session.

---

## 3. The metrics that already exist

All emitted from `bsn-cloud/src/BSN.Cloud.Bootstrap/Telemetry/` (`Metrics.cs`,
`Tracing.cs`), exported OTLP to grafana-alloy to Mimir.

| Signal | Name | Emitted by | Tells you |
|---|---|---|---|
| Request duration histogram | `http_server_request_duration_seconds` | **Every** new-stack service (gateway and backend) | Server wall-clock per route, tagged `http_route`, `http_response_status_code`, `http_request_method`, `service` |
| Authorizer duration | `bs_authorizer_duration` | gRPC Authorizer (`Metrics/AuthorizerMetrics.cs`) | Cost of the authz decision |
| Authorizer cache | cache hit/miss counters | gRPC Authorizer | Whether authz is being served from cache (it should be) |
| DB spans | `AddSqlClientInstrumentation()` on `Back.Device` | `devices-service` | Per-request SQL time, visible in traces |

Because the histogram is emitted by **both** the gateway service and the backend
service, subtracting one from the other isolates the gateway + auth overhead from
the backend's own processing.

---

## 4. The queries

Run these in Grafana Explore. Replace `devices-service` and the route label with the
target from section 5. Metric label names can vary slightly by exporter version
(`http_route` vs `http_server_route`) — confirm by typing the metric name in Explore
and inspecting the label dropdown.

### 4.1 Backend p95 for one route

```promql
histogram_quantile(
  0.95,
  sum by (le, http_route) (
    rate(http_server_request_duration_seconds_bucket{service="devices-service"}[5m])
  )
)
```

Add `, http_route="/2019/03/REST/Devices/{serial}/"` inside the selector to pin a
single route, or keep the `by (http_route)` grouping to rank all routes at once.

### 4.2 Gateway + auth overhead (gateway minus backend)

Run 4.1 against the **Front gateway** service as well:

```promql
histogram_quantile(
  0.95,
  sum by (le, http_route) (
    rate(http_server_request_duration_seconds_bucket{service="bsn-cloud-front"}[5m])
  )
)
```

The difference `gateway_p95 - backend_p95` for the same route is the time spent in
the gateway and the authorizer, not in the backend. (Confirm the exact `service`
label value for the Front host in Explore.)

### 4.3 Authorization cost

```promql
histogram_quantile(0.95, sum by (le) (rate(bs_authorizer_duration_bucket[5m])))
```

The JWT is validated by a **local, cached JWKS signature check** (no per-request
Keycloak round-trip — `Authorizer.gRPC/Services/Users/KeycloakJwtValidator.cs`) and
authorization decisions are cached (`CachingAuthorizationCoordinator`), so this
should be small. If it is not, cache misses are the story — check the hit/miss
counters.

### 4.4 Request volume and error rate (context for the percentiles)

```promql
sum by (http_route, http_response_status_code) (
  rate(http_server_request_duration_seconds_count{service="devices-service"}[5m])
)
```

A p99 computed over three requests is noise. Confirm the route actually has traffic
during your window before trusting its percentiles.

### 4.5 DB time per request — use traces, not a metric

OTel traces already break out the span tree **Front gateway → authorizer →
devices-service → SqlClient**. In Grafana, open Tempo/traces, filter by `http.route`
and a high `duration`, and read the SqlClient child span. This gives DB time per
request **for free** on the new stack — no query needed.

---

## 5. Target routes and the pages that call them

These are the heavy pages and the Main API routes they hit (from the front-end
analysis). Start with these.

| UI page (bs-management-cloud-ui) | Main API calls on load | Backend route to query |
|---|---|---|
| **Dashboard** | `self/networks`, `self/session/network`, `playersStatus`, `subscriptions/count` | devices-service player-status route + authorizer |
| **Device List / Network** | single `devices` GET | `GET /…/Devices` |
| **Groups** | `Groups/Regular` | groups route |
| **Subscriptions** | `Subscriptions`, `/Count` | subscriptions route(s) |

For each, record p50/p90/p95/p99 and request volume. A simple table of
`route -> p95 -> volume` across these four is enough to answer the first question.

---

## 6. The legacy blind spot

`bsn-cloud-inherited` (`BSN.WebServices`, raw ADO.NET) emits **no** OTel and has no
per-request duration log (`UseSerilogRequestLogging()` is explicitly disabled). If a
target route is served here, these queries return nothing for it.

**How to tell which stack a route is on:** check the gateway route table at
`bsn-cloud/src/BSN.Cloud.Hosts.Web.API.Front/ntrada.yml`. Routes mapped to
`devices-service` (and the other new backends) are instrumented; anything proxied to
the legacy web services is not.

**Suspect legacy DB queries** (raw T-SQL in
`bsn-cloud-inherited/BusinessLayer/BNM.BusinessLayer/Database/DbDevices.cs`), worth
knowing about when a legacy route is slow:

- `GetDevices()` — `CROSS APPLY [dbo].[Device_LocationsHierarchy](DeviceId)`, a
  table-valued function evaluated per row.
- `GetGroupStatus()` — scalar UDF `[dbo].[Device_Health](DeviceID)` per row inside an
  aggregate.
- `GetDevicesSecure()` — `ROW_NUMBER() OVER (<dynamic sort>)` pagination with
  `DeviceSubscriptions` joins and correlated `ActivationDate` subqueries.
- `GetDevicesCountSecure()` — re-runs the full filter as a separate COUNT.

You cannot time these from metrics today. To get a number, either add the
`Server-Timing` header (document 2, section C) or capture the SQL with SQL Server's
own tooling (Query Store / Extended Events) — a DBA task outside this runbook.

---

## 7. How to read the result (decision rule)

For each target page, compare the sum of backend p95 across the routes it calls
against the page's observed total load time (what the user feels).

- **Backend p95 is a small fraction of page time** → the backend is not the
  bottleneck. The latency is network, front-end serialization (the bootstrap
  waterfall), or browser render. Go to document 2 and measure the front-end.
- **Backend p95 is itself multiple seconds for a route** → you have a genuine slow
  endpoint. Open its trace (4.5) and read the SqlClient span to confirm it is the
  database, then hand the query to the data team.
- **Gateway minus backend is large** → the overhead is in the gateway or
  authorizer, not the backend service. Check `bs_authorizer_duration` and the cache
  counters (4.3).

Record the numbers. They are the baseline that document 2's active tests are
compared against.

---

## 8. Limitations

- New stack only; legacy routes are invisible (section 6).
- Server wall-clock only — excludes everything outside the server process (network,
  browser).
- Percentiles are only meaningful with enough traffic in the window (4.4).
- `service`/`http_route` label spellings depend on exporter version — verify in
  Explore before trusting a dashboard built blind.
