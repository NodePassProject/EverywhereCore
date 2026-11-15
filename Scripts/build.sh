#!/usr/bin/env bash
# Builds EverywhereCore.xcframework via gomobile bind.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_DIR="$ROOT/go"
OUT="$ROOT/EverywhereCore.xcframework"

GOPATH="$(go env GOPATH)"
GOBIN="$GOPATH/bin"
PATH="$GOBIN:$PATH"
export PATH

if ! command -v gomobile >/dev/null 2>&1; then
    echo "→ installing gomobile + gobind"
    go install golang.org/x/mobile/cmd/gomobile@latest
    go install golang.org/x/mobile/cmd/gobind@latest
fi

cd "$CORE_DIR"

# --- Symmetric varz patch -------------------------------------------------
PATCHED_ROOT="$CORE_DIR/.patched"

cleanup_replaces() {
    go mod edit -dropreplace=github.com/sagernet/tailscale  2>/dev/null || true
    go mod edit -dropreplace=github.com/metacubex/tailscale 2>/dev/null || true
}
trap cleanup_replaces EXIT
cleanup_replaces

echo "→ go mod tidy"
go mod tidy

apply_varz_patch() {
    local module="$1" prefix="$2" dirname="$3"
    local version
    version="$(go list -m -f '{{.Version}}' "$module" 2>/dev/null || true)"
    if [[ -z "$version" ]]; then
        echo "→ $module not in deps; skipping varz patch"
        return
    fi
    local gomodcache src dest
    gomodcache="$(go env GOMODCACHE)"
    src="$gomodcache/$module@$version"
    dest="$PATCHED_ROOT/$dirname@$version"
    if [[ ! -d "$src" ]]; then
        go mod download "$module@$version"
    fi
    if [[ ! -f "$dest/tsweb/varz/varz.go" ]]; then
        rm -rf "$dest"
        mkdir -p "$dest"
        cp -R "$src/" "$dest/"
        chmod -R u+w "$dest"
        sed -i.bak \
            -e "s/expvar\.Publish(\"process_start_unix_time\"/expvar.Publish(\"${prefix}_process_start_unix_time\"/" \
            -e "s/expvar\.Publish(\"version\"/expvar.Publish(\"${prefix}_version\"/" \
            -e "s/expvar\.Publish(\"go_version\"/expvar.Publish(\"${prefix}_go_version\"/" \
            -e "s/expvar\.Publish(\"counter_uptime_sec\"/expvar.Publish(\"${prefix}_counter_uptime_sec\"/" \
            -e "s/expvar\.Publish(\"gauge_goroutines\"/expvar.Publish(\"${prefix}_gauge_goroutines\"/" \
            -e "s/expvar\.Publish(\"node_boot_time_seconds\"/expvar.Publish(\"${prefix}_node_boot_time_seconds\"/" \
            "$dest/tsweb/varz/varz.go"
        rm -f "$dest/tsweb/varz/varz.go.bak"
    fi
    go mod edit -replace "$module=$dest"
    echo "→ patched $module@$version (prefix=${prefix}_)"
}

apply_varz_patch github.com/sagernet/tailscale  sagernet  sagernet-tailscale
apply_varz_patch github.com/metacubex/tailscale metacubex metacubex-tailscale

BUILD_TAGS="\
with_clash_api \
with_grpc \
with_gvisor \
with_quic \
with_tailscale \
with_utls \
with_wireguard"

LDFLAGS="-s -w"

TARGETS="${EVCORE_TARGETS:-ios,iossimulator,macos}"

echo "→ gomobile bind target=$TARGETS tags=$BUILD_TAGS"
rm -rf "$OUT"
gomobile bind \
    -target="$TARGETS" \
    -tags="$BUILD_TAGS" \
    -ldflags="$LDFLAGS" \
    -o "$OUT" .

echo "✓ built $OUT"
du -sh "$OUT"
