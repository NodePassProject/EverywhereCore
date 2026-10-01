// swift-tools-version:5.9
//
// `main` ships with a local `binaryTarget(path:)` so the package
// resolves directly against the on-disk xcframework — run
// `Scripts/build.sh` once, then `.package(path: "../EverywhereCore")`
// works in both consuming apps without a network round-trip.
//
// `Scripts/release.sh vX.Y.Z` rewrites this file to use
// `binaryTarget(url:, checksum:)` on the tagged commit only, then
// restores this version on `main`. Tagged consumers resolve against
// the GitHub Release asset; `main` consumers stay on the local path.

import PackageDescription

let package = Package(
    name: "EverywhereCore",
    platforms: [
        .iOS(.v15),
        .macOS(.v13),
    ],
    products: [
        .library(name: "EverywhereCore", targets: ["EverywhereCore"]),
    ],
    targets: [
        .binaryTarget(name: "EverywhereCore", path: "EverywhereCore.xcframework"),
    ]
)