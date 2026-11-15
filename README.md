# EverywhereCore

Go core (mihomo + sing-box + xray-core, glued by gomobile) packaged as
a Swift binary `.xcframework`. Consumed by the Everywhere iOS app and
the Everywhere-macOS app via Swift Package Manager.

The three core dependencies are not vendored. `go/go.mod` pins them
by Go-module-compatible semver; the GitHub Actions workflow at
`.github/workflows/upstream-watch.yml` polls the Go proxy daily and
auto-cuts a new release whenever any upstream moves.

## Layout

```
Package.swift              local binaryTarget(path:) on main, rewritten
                           to url+checksum on tagged releases
go/                        gomobile entry package; *.go + go.mod
Scripts/build.sh           gomobile bind → EverywhereCore.xcframework
.github/workflows/
  upstream-watch.yml       daily upstream poll + auto-release
```

## Building locally

```sh
git clone https://github.com/NodePassProject/EverywhereCore
cd EverywhereCore
Scripts/build.sh           # gomobile bind, deps fetched from Go proxy
```

Produces `./EverywhereCore.xcframework` with `ios-arm64`,
`ios-arm64_x86_64-simulator`, and `macos-arm64_x86_64` slices.

To build fewer slices — each is a full compile of all three cores —
set `EVCORE_TARGETS`, e.g. `EVCORE_TARGETS=iossimulator/arm64
Scripts/build.sh` for simulator-only work.

## Consuming from an Xcode project

```swift
.package(url: "https://github.com/NodePassProject/EverywhereCore", from: "2026.05.14")
```

The tag's `Package.swift` declares `binaryTarget(url:, checksum:)`
pointing at the GitHub Release asset; SwiftPM downloads and verifies
by SHA256.

## How releases happen

`.github/workflows/upstream-watch.yml` runs daily at 08:00 UTC and on
manual `workflow_dispatch`. Each run:

1. Queries `proxy.golang.org/<module>/@latest` for mihomo, sing-box,
   and xray-core (stable tags only — no pre-releases).
2. Compares each against the version currently pinned in `go/go.mod`.
3. If at least one is newer (or if dispatched with `force_release: true`):
   - `go get` each to its latest, `go mod tidy`
   - `Scripts/build.sh` to build the xcframework
   - `zip` the xcframework, compute SHA256
   - Write a release-flavored `Package.swift` (url+checksum form),
     commit + tag `vYYYY.MM.DD`
   - Append `.1`, `.2`, … to the tag if multiple runs land same day
   - Restore the dev `Package.swift` on top so `main` HEAD remains
     resolvable for in-tree consumers
   - Push tag + main; `gh release create` with the zip attached
4. Otherwise: no-op (logged as a notice).

To bootstrap the first release after pushing this repo, run the
workflow from the Actions tab with `force_release: true`.

## Pinning a specific upstream version manually

Edit `go/go.mod`, push to main. The next cron run will detect the
manual pin is current (or stale) and act accordingly. If you want a
release for a manual bump immediately, dispatch the workflow with
`force_release: true`.
