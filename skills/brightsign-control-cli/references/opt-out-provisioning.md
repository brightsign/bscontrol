# Join a player to a BSN network by registry keys ("opt-out")

**Opt-out** joins an already-deployed, already-running player to a BrightSign Control
network **in place** — no card format, wipe, content re-sync, or site visit. It is an
*add*, not a move: the player keeps playing its current content and comes up registered
and manageable. It works because every player holds an idle outbound WebSocket to the
cloud whether or not it belongs to a network, so the registration path can fire as soon
as the keys are present.

The mechanism is **five registry keys plus a registration token**. Write the five keys
into the player's `networking` section; the player self-registers (creating its own
record in the cloud Main database) the next time its WebSocket comes up.

> Privileged, internal operation (tracking issue BCN-21293). The remote delivery path
> rides the admin RDWS scope family; see [Caveats](#caveats-and-requirements). This is not
> a general-purpose user feature.

## The five keys (section `networking`, all string values)

| Key | Value | Meaning |
|---|---|---|
| `bsnce` | `true` | Enable BSN.cloud access on the player. |
| `a` | `<network name>` | The network (account) to join. |
| `bsnrt` | `<registration token>` | Registration token minted for that network (below). Consumed on success — it disappears from the registry after registration. |
| `sut` | `Standalone` | Setup type. `Standalone` is the safe default (avoids a wrong default group). OS also accepts `bsn` and `lfn`. |
| `inp` | `yes` | Inherit network properties — register using the existing network config. This is what keeps opt-out **non-destructive**. |

These five are the minimum to register the player and make it appear in RDWS. `inp=yes`
drives the **registration** code path, not **reprovision** — so no content or storage is
deleted. (Reprovision, which *does* wipe storage, is a different command and opt-out never
invokes it.)

A reboot is **optional**: the registration service re-checks on every WebSocket-open,
network reconnect, and retry tick. Tooling still issues a reboot as a reliable trigger for
a clean re-read and fresh connection.

## Step 1 — mint the registration token (`bsn-regtoken-get`)

The token is two cloud calls, and the first (binding the session to the network) is
**mandatory** — the token POST carries no network of its own and inherits the session's
bound network, so skipping the bind mints a token in an unpredictable network:

```text
PUT  /2022/06/REST/Self/Session/Network   {"name":"<network name>"}   -> 204
POST /2022/06/REST/Provisioning/Setups/Tokens/                         -> 201
```

bscontrol wraps both as `Provisioning.GenerateDeviceToken`, exposed by **`bsn-regtoken-get`**.
The token has `cert` scope, is valid two years, and is **reusable across many devices** —
mint once, reuse across a batch.

```bash
# clean, scriptable token value (human-readable details go to stderr)
BSNRT=$(bsn-regtoken-get --network "My Network" --json | jq -r '.token')
```

There is no token-only flag; `--json` stdout is clean to pipe (the `bsnrt` key needs just
the raw `token` string). Use a **separate, network-scoped credential** for this step —
deliberately not the admin RDWS credential that delivers the keys.

## Step 2 — write the five keys, then reboot

> **bscontrol limitation — read this before reaching for `bsn-rdws-registry-set`.**
> The `bsn-rdws-*` tools assume a player **already registered on your network** (they do a
> network session + device-list lookup). An opt-out target is **not yet registered**, so
> `bsn-rdws-registry-set` / `bsn-rdws-reboot` cannot address it. The remote opt-out path
> uses the generic RDWS `sendWsMessage` envelope, which bscontrol does **not** implement
> today. So with this CLI suite, `bsn-regtoken-get` mints the token and `bsn-rdws-info`
> verifies (Step 3), but **key delivery is out-of-band** by one of the methods below.
> (For a player already on your network, `bsn-rdws-registry-set` is the right tool — but
> that is ordinary per-player control, not opt-out.)

**Shortcut — generate the script with `bsn-device-reg-script`.** For the local methods
(B and the BrightScript form), this tool mints the token *and* emits a ready-to-run script
with all five keys and the reboot already in it, to stdout for redirection:

```bash
bsn-device-reg-script --network "My Network" --type sh  --file register-player.sh    # registry-CLI script
bsn-device-reg-script --network "My Network" --type brs > register-player.brs         # or redirect stdout
# sh is for an INSECURE player, run from its shell, e.g.: bash /storage/sd/register-player.sh
```

Use `--file <path>` to write the script to a file, or let it write to stdout and redirect.

It does not deliver the keys for you — you still copy/run the script on the player — but it
removes the hand-assembly below. The manual forms remain useful for Local DWS over curl and
for understanding exactly what gets written.

**A. Local DWS over the LAN** (hands-on / same network; needs the DWS password):
```bash
PLAYER=192.0.2.50
PW=<dws-password>
BSNRT=$(bsn-regtoken-get --network "My Network" --json | jq -r '.token')

for kv in "bsnce=true" "a=My Network" "bsnrt=$BSNRT" "sut=Standalone" "inp=yes"; do
  key=${kv%%=*}; val=${kv#*=}
  curl --digest -u "admin:$PW" -H 'Content-Type: application/json' \
    -X PUT "http://$PLAYER/api/v1/registry/networking/$key" -d "{\"value\":\"$val\"}"
done
curl --digest -u "admin:$PW" -X PUT "http://$PLAYER/api/v1/control/reboot"
```

**B. `registry` CLI over SSH or telnet** (flushes immediately):
```sh
registry networking bsnce true
registry networking a "My Network"
registry networking bsnrt <token>
registry networking sut Standalone
registry networking inp yes
reboot
```

**C. Remote over the cloud (the true "opt-out" path)** — one `sendWsMessage` envelope per
key then one to reboot, against the player's idle WebSocket. Requires the serial-bound
admin RDWS scope and Supervisor >= 3.3.4.1. **Not available through bscontrol/`bsn-*`
today**; the reference implementation is `opt-out/register-player.js` (Node):
```text
PUT {WSS}/rest/v1/sendWsMessage?destinationType=player&destinationName=<serial>
{ "method":"PUT", "route":"/v1/registry/networking/bsnce", "data":{"value":"true"} }
```

**Order and failure handling:** mint the token first (Step 1) so a token failure never
leaves a player half-written. Write all five keys, then reboot. The RDWS gateway returns
HTTP 200 even when the player refuses a call — inspect the envelope body
(`data.error`, `data.result.success`), and **abort before the reboot on the first failed
write** so you do not reboot a half-configured player.

## Step 3 — verify (`bsn-rdws-info`, with backoff)

An RDWS call only succeeds against a registered, online player holding a live WebSocket, so
a successful response is itself proof the join worked — stronger than merely seeing the
device in the Main list (which proves a record exists, not that the player is reachable).

```bash
# poll with backoff: expect gateway timeouts (504 "not connected") for the first
# attempts, then success once the player registers.
until bsn-rdws-info --serial "$SERIAL" --json; do sleep 10; done
```

The returned info mirrors the `networking/bsnce` key you wrote, so `bsnce=true` in the
response is a clean, self-consistent confirmation the player came up cloud-enabled.

## Caveats and requirements

- **Supervisor >= 3.3.4.1**, present in 9.1 OS builds after 9.1.131. Below that the player
  does not understand the opt-out command — check first.
- **Remote path scope:** a client-id carrying the admin RDWS scope with **every target
  serial bound to it** (the blast radius is the serial list, not the fleet), plus a
  separate network-scoped credential to mint the token. Do not hard-assert an exact scope
  string — print the granted scope and let the gateway decide (production succeeded with
  `wss.api.admin.limited.rdws`, not the documented `wss.api.admin.rdws`).
- **Bounded is not harmless:** against players it can reach, the channel carries arbitrary
  Supervisor commands, including destructive ones. Never expose it as a general API.
- The device record is created by the player's **self-registration** — there is no
  Main-API "create device" call, and a B-Deploy association record is **not** required for
  opt-out.

Source: `brightsign-playground/bs-agent-docs` →
`docs/guides/bsn-cloud/provisioning-by-registry.md` (and `docs/architecture/flow-provisioning.md`).
