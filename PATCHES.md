# Patches

Ledger of wiring quirks and gomobile mechanics that make the three
upstreams co-exist in one Go module. The cores themselves are unpatched
— if that ever changes, see "Future source patches" at the bottom.

Pinned versions live in `go/go.mod`. The daily upstream-watch workflow
keeps them current; see `README.md` for the release flow.

## go.mod overrides (`go/go.mod`)

### tools.go to anchor `golang.org/x/mobile/bind`

`gomobile bind` invokes `gobind` from a temporary directory, and `gobind`
imports `golang.org/x/mobile/bind`. In module mode, that import has to
appear somewhere in our module's import graph for `go list` to find it.
`go/tools.go` does so under `//go:build tools`, so the package never
compiles into the framework but `go.mod` keeps the require.

**On upstream bump.** No action — this is a gomobile mechanic.

### `-ldflags="-s -w"` so archive validation doesn't demand a dSYM

`Scripts/build.sh` strips Go-side debug info via `-ldflags="-s -w"`
(`-s` = strip symbol table, `-w` = strip DWARF). With no DWARF in the
binary the validator does not look for a dSYM and the warning goes
away. Side effects:

- Framework is much smaller.
- No Go-side symbolication in Apple crash reports.

**On upstream bump.** No action — this is a gomobile mechanic.

## Source patches

Xray-core, sing-box, and mihomo all build unmodified from their
published Go-module tags. The only sources we modify are two
transitive deps that fight over `expvar.Publish`.

### tailscale forks: rename six expvar names to coexist

Two tailscale forks land in the dep graph and each runs its own
init():

- `github.com/sagernet/tailscale` — pulled in by sing-box's DERP
  service when `with_tailscale` is enabled.
- `github.com/metacubex/tailscale` — pulled in by mihomo
  unconditionally.

Both forks ship a copy of `tsweb/varz/varz.go` whose init() calls
`expvar.Publish` six times with hardcoded names:
`process_start_unix_time`, `version`, `go_version`,
`counter_uptime_sec`, `gauge_goroutines`, and `node_boot_time_seconds`.
`expvar.Publish` panics on duplicate names — two distinct module paths
run two distinct inits against the same process-global map → boom on
the second:

```
panic: Reuse of exported var name: process_start_unix_time
    expvar.Publish(.../tsweb/varz/varz.go:40)
```

`Scripts/build.sh` rewrites both copies after `go mod tidy`:

1. Resolve each fork's version via `go list -m`.
2. Copy the module-cache source into `go/.patched/<vendor>-tailscale@<version>/`.
3. Sed-rewrite the six `expvar.Publish(...)` calls in
   `tsweb/varz/varz.go` to prefix the published name with the
   vendor — `sagernet_process_start_unix_time`,
   `metacubex_version`, etc.
4. `go mod edit -replace` so the build links the patched copies.

Patched directories are version-suffixed and cached across builds.
An `EXIT` trap drops the replace directives so `go.mod` stays clean
in version control. `.patched/` is gitignored.

Nothing in either fork looks these names up by string — they're
emitted via Prometheus walk in `varz.Handler`, which iterates the
global `expvar.Map`. Renaming is invisible unless the host app also
scrapes that handler and pins exact names, which the iOS NE never does.

`node_boot_time_seconds` is the newest of the six and is the only one
published conditionally — its value comes from the `btime` line of
Linux's `/proc/stat`, so the guard fails and the name is never
published on iOS or macOS. It is renamed anyway: the sed line is free,
and it keeps the invariant "no name either fork publishes is left
unprefixed" true by construction rather than by platform accident.

**On upstream bump.** The patch is keyed on the resolved module
versions, so a new sing-box or mihomo tag that bumps either fork
just regenerates `.patched/<vendor>-tailscale@<new-version>/`. The
sed targets have been stable in `tsweb/varz/varz.go` for years;
if a fork ever drops, renames, or *adds* one,
`grep -c 'expvar.Publish' \
.patched/<vendor>-tailscale@.../tsweb/varz/varz.go` will not equal the
six `-e` clauses in `Scripts/build.sh` and the mismatch is the signal
to re-derive the list.

**Upstream-correct fix.** A canonical tailscale fork that both
sing-box and mihomo agree to share would obviate this. Until then,
the duplication is an unavoidable side effect of running both cores
in one binary.

## Wiring quirks per core

These are not patches but call-site requirements that the wrappers in
`go/` already encode. Listed here so they survive a future rewrite.

### TUN inbound: each core consumes the utun fd directly

We don't ship a userland tun→socks bridge. Each core owns its own TUN
inbound, with the FD plumbed differently:

- **Xray-core**: read from the `xray.tun.fd` env var (see
  `proxy/tun/tun_darwin.go`). `go/xray.go` sets it before
  `core.StartInstance`.
- **sing-box**: read via an `adapter.PlatformInterface` whose
  `OpenInterface` is invoked by the tun inbound. The interface is
  registered on the `box.Options.Context` via
  `service.ContextWith[adapter.PlatformInterface]`. See
  `go/singbox.go` for the minimal implementation; only
  `OpenInterface`, `UnderNetworkExtension` and the
  `CreateDefaultInterfaceMonitor` no-op stub do meaningful work.
- **mihomo**: written into `cfg.General.Tun.FileDescriptor` between
  `executor.ParseWithBytes` and `hub.ApplyConfig`. The wire-level
  YAML key is `tun.file-descriptor`, but we keep the FD out of the
  config string so users can't accidentally pin a stale value.

For sing-box and mihomo we `syscall.Dup` the FD before handing it to
sing-tun — its `Close()` always closes the wrapped `os.File`, so a
non-dup'd path would tear down NEPacketTunnelFlow's underlying utun
out from under the Network Extension. Xray's darwin tun path checks
`ownsFd` and skips `Close()` when the FD came in externally, so we
don't dup there.

### Xray-core: needs `_ "main/distro/all"` and `_ "main/json"`

`distro/all` registers every inbound/outbound/transport via init().
`main/json` registers the JSON config loader and transitively pulls
in `infra/conf` and `proxy/tun`, which is what registers the `tun`
inbound type in the JSON loader's protocol map.
`core.StartInstance("json", …)` fails without both. See
`go/xray.go`.

### sing-box: gomobile bind needs `-tags=with_*`

sing-box keeps optional subsystems behind Go build tags. With no tag
set, the corresponding `*_stub.go` files compile in and the feature
returns "not included in this build" errors at runtime.

`Scripts/build.sh` enables the subset of sing-box's tag matrix that
matters to an iOS Network Extension client. The full list is
reproducible — re-derive it any time against the module cache:

```bash
( cd go && go mod download github.com/sagernet/sing-box )
grep -rh '^//go:build' "$(go env GOMODCACHE)/github.com/sagernet/sing-box@$(awk '/sing-box v/{print $2; exit}' go/go.mod)/" \
  | grep -oE 'with_[a-zA-Z0-9_]+' | sort -u
```

Currently shipped (7):

| Tag                   | Unlocks                                        |
| --------------------- | ---------------------------------------------- |
| `with_clash_api`      | clash REST/WebSocket API (yacd talks to this)  |
| `with_grpc`           | full gRPC transport (vs. the lite HTTP/2 fallback) |
| `with_gvisor`         | gVisor + mixed TUN stack (sing-tun ships an iOS-tuned TCP buffer); also enables gVisor for wireguard endpoints |
| `with_quic`           | QUIC transports — Hysteria/Hysteria2/TUIC, QUIC/HTTP3 DNS |
| `with_tailscale`      | tailscale endpoint (joins a tailnet from inside the NE) |
| `with_utls`           | uTLS client fingerprinting and client-side REALITY |
| `with_wireguard`      | wireguard outbound endpoint                    |

Excluded:

| Tag                   | Why                                                                |
| --------------------- | ------------------------------------------------------------------ |
| `with_acme`           | ACME issuance is for *inbound* TLS servers. iOS NE is client-only. Drops `caddyserver/certmagic`, `caddyserver/zerossl`, `mholt/acmez`, `libdns/*`. |
| `with_ccm`            | "CCM" service registry runs an HTTP service that proxies the Anthropic Claude API — server-side only. Drops `anthropics/anthropic-sdk-go`. |
| `with_cloudflared`    | New in 1.14. Registers a Cloudflare Tunnel **inbound** (`protocol/cloudflare`) — server-side. Drops `sagernet/sing-cloudflared`. |
| `with_dhcp`           | `dhcp://auto` DNS transport probes DHCP via a raw socket bound to a named system interface — unreliable from inside the iOS NE sandbox, and iOS configs don't use it. Drops `insomniacslk/dhcp`. |
| `with_ech`            | Deprecated in 1.13 — ECH moved to Go stdlib; tag's `_stub.go` now intentionally fails the build with that explanation. |
| `with_external_windivert` | New in 1.14. Selects an out-of-tree WinDivert driver asset; every file under `common/windivert` is `_windows`. Irrelevant on Apple platforms. |
| `with_naive_outbound` | Pulls in `sagernet/cronet-go/all`, which has no Go files for iOS.  |
| `with_ocm`            | "OCM" service registry runs an HTTP service that proxies the OpenAI API — server-side only. Drops `openai/openai-go/v3`. |
| `with_openconnect`    | New in 1.14. Registers an OpenConnect **endpoint** + DNS transport — genuinely client-side, so this is a deliberate hold rather than an automatic exclusion: no Everywhere config uses OpenConnect today, and enabling it pulls `sagernet/sing-openconnect`. Revisit if the app adds OpenConnect profiles. |
| `with_openvpn`        | New in 1.14. Same story as `with_openconnect` — OpenVPN endpoint + DNS transport, client-side, held pending a config that needs it. Drops `sagernet/sing-openvpn`. |
| `with_reality_server` | Deprecated in 1.13 — folded into `with_utls`; same intentional-build-error pattern. |
| `with_usbip`          | New in 1.14. USB/IP device-sharing *service* — a desktop-daemon feature, and its `include` registration is gated on `darwin && cgo`/`linux`/`windows` anyway. Drops `sagernet/sing-usbip`. |
| `with_v2ray_api`      | gRPC stats *server* — iOS dashboards talk to the clash API instead. Combined with `with_grpc` retention, the only `google.golang.org/grpc` *server* consumer is gone, but the client transport stays. |

When sing-box adds a new `with_*` stub, the grep above will surface
it; evaluate it against the "client-only inside a Network Extension"
filter before appending to `BUILD_TAGS` in `Scripts/build.sh`. If a
new tag's stub fails the build the way `with_ech` does, that's
sing-box telling you the feature has been merged elsewhere.

### sing-box: must pass `include.Context(ctx)` into `box.New`

sing-box 1.10+ requires the inbound/outbound/endpoint/DNS-transport/
service registries to be attached to the context that `box.New` is
called with. The `github.com/sagernet/sing-box/include` package's
`Context(ctx)` helper bundles them in one call.

If you only pass `context.Background()`, `box.New` parses the JSON but
cannot instantiate `socks`, `direct`, `vmess`, etc., and returns an
error. From iOS's perspective the Network Extension dies the instant
the tunnel comes up. See `EverywhereCore/singbox.go`.

**On upstream bump.** Verify `include.Context` is still the canonical
entry point — the registry surface has been refactored a couple of
times in 1.x.

### sing-box: `singBoxPlatform` must track `adapter.PlatformInterface`

`go/singbox.go`'s `singBoxPlatform` is a hand-written implementation of
`adapter.PlatformInterface`, and Go interface satisfaction is all-or-
nothing: sing-box adding *one* method to that interface breaks our build
with `cannot use pi (variable of type *singBoxPlatform) as
adapter.PlatformInterface value`. This is the single most likely way an
upstream bump breaks the build, because the interface is sing-box's
mobile-embedder seam and it grows every minor release.

1.14 changed it in three ways at once:

- **Added** `ProcessPlatformOptions`, `CancelNotification`, the neighbor
  trio (`UsePlatformNeighborResolver`, `StartNeighborMonitor`,
  `CloseNeighborMonitor`), the platform-shell set (`UsePlatformShell`,
  `CheckPlatformShell`, `OpenShellSession`, `LookupUser`,
  `LookupSFTPServer`, `ReadSystemSSHHostKey`, `TailscaleHostname`), and
  the bridge pair (`UsePlatformBridge`, `CreateBridge`).
- **Changed** `ReadWIFIState()` → `ReadWIFIState(ctx context.Context)`,
  so the call from `NetworkManager.UpdateWIFIState(ctx)` is cancellable.
- **Removed** `SystemCertificates()`. Extra methods don't break
  satisfaction, so a stale one lingers silently — ours was deleted.

Every added method declines the capability, matching what libbox's own
`platformInterfaceStub` returns for an embedder that can't offer it. The
two returns that are *not* errors are load-bearing:
`ProcessPlatformOptions` must return nil (the tun inbound closes the
interface and fails startup if it errors — it is only meaningful for
desktop `platform.http_proxy` wiring), and `CheckPlatformShell` must
return nil because it is a capability probe, not a failure path.

**On upstream bump.** Diff the interface against our implementation
rather than reading the compiler error alone — a bump can silently
*remove* a method too:

```bash
( cd go && go mod download github.com/sagernet/sing-box )
sed -n '/type PlatformInterface interface {/,/^}/p' \
  "$(go env GOMODCACHE)/github.com/sagernet/sing-box@$(awk '/sing-box v/{print $2; exit}' go/go.mod)/adapter/platform.go"
grep -c 'func (p \*singBoxPlatform)' go/singbox.go
```

The counts should match; libbox's `platformInterfaceStub` in
`experimental/libbox/config.go` is the reference for what each new
method should return.

### mihomo: must call `hub.ApplyConfig`, not `executor.ApplyConfig`

mihomo has two `ApplyConfig` functions:

- `executor.ApplyConfig(cfg, force)` — sets up DNS, proxies, rules,
  inbound listeners (socks-port, http-port, mixed-port…). Does **not**
  start the external-controller HTTP/WS API server.
- `hub.ApplyConfig(cfg)` — wraps `applyRoute(cfg)` (which calls
  `route.ReCreateServer` and *that* boots the API server) followed by
  `executor.ApplyConfig(cfg, true)`.

If you call only `executor.ApplyConfig`, the SOCKS inbound and the
tunnel work fine, but the clash REST API never starts and yacd shows
"cannot connect to 127.0.0.1:9090". `go/mihomo.go` calls
`hub.ApplyConfig`.

`hub.ApplyConfig` returns no error — failures inside it are logged via
mihomo's own logger.

### Xray-core: dashboard data is served in-process (`Control`)

mihomo and sing-box bring a clash API the app's dashboard can talk to.
Xray-core doesn't: its gRPC API would need an HTTP/2 client on the
Swift side, and the `metrics` expvar endpoint is read-only. So
`go/xray_control.go` answers the dashboard through the bound
`Control(method, paramsJSON)` function, using the same feature
interfaces Xray's own gRPC services sit on:

| Method    | Backed by                                                        |
| --------- | ---------------------------------------------------------------- |
| `info`    | the JSON config the instance was started with (outbounds, rules) |
| `proxies` | `routing.BalancerOverrider` / `BalancerPrincipleTarget`, `outbound.HandlerSelector`, `extension.Observatory` |
| `select`  | `routing.BalancerOverrider.SetOverrideTarget` (what `RoutingService.OverrideBalancerTarget` calls) |
| `delay`   | `tagged.Dialer` with the outbound tag forced — the technique `app/observatory` uses |
| `stats`   | `VisitCounters` on the stats manager (`outbound>>>tag>>>traffic>>>uplink` …) |
| `logs`    | a tee on `common/log`'s process-global handler                   |

Things that are easy to break on an upstream bump:

- **Instance context.** `tagged.Dialer` refuses a context without the
  running instance in it, and `core.toContext` is unexported. The
  bridge builds one with `context.WithValue(ctx, core.XrayKey(1), inst)`
  — `XrayKey` is exported "for test" and its value is the unexported
  `xrayKey` constant. If upstream changes either, `delay` starts
  failing with "Instance context variable is not in context".
- **Balancer and rule names come from the config, not the router.**
  `Router.balancers` and `Router.rules` are unexported, so the bridge
  re-parses the config for tags and only asks the router about them by
  tag. A balancer or rule added at runtime through Xray's own API
  would not show up.
- **`VisitCounters` is asserted, not assumed.** It is on the concrete
  `app/stats.Manager` in every version and only joined the
  `features/stats.Manager` interface later; the bridge uses a local
  one-method interface so it builds against both.
- **Log tee.** `app/log.Instance` registers itself as the global
  handler in its constructor; the bridge wraps it after start and puts
  it back on stop. Capture is off until someone polls `logs` and stops
  15 s after the last poll, so an idle dashboard costs one atomic load
  per record.

The stats counters only exist when the config has a `stats` object and
`policy.system.statsOutboundUplink/Downlink`; the app's ConfigNormalizer
adds them while its dashboard is enabled.

## Future source patches

The Go module cache is read-only by design, so patching upstream
sources in place is not an option. Two paths exist depending on the
size of the change:

**Build-time sed (preferred for tiny, line-level fixes).** Copy the
module-cache source into `go/.patched/<vendor>-<repo>@<version>/`,
apply a `sed` rewrite, and add a transient `replace` directive that
an `EXIT` trap drops afterwards. See the tailscale-forks entry under
"Source patches" for the template. No external repo to maintain;
patch follows the upstream version automatically.

**GitHub fork (for anything bigger).** When a change is too complex
for in-place sed:

1. Fork the upstream to `github.com/NodePassProject/<repo>` and apply
   the change on a branch.
2. Add a `replace` directive to `go/go.mod`:
   `replace github.com/x/y => github.com/NodePassProject/y vX.Y.Z`.
3. Append a section under "Source patches" describing **why**, **what
   file**, and **what the upstream-correct fix would be** so we can
   drop the fork when upstream catches up.
4. Update `.github/workflows/upstream-watch.yml` to watch the fork's
   `@latest` instead, or to skip that core's auto-bump entirely.
