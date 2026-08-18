// swift-tools-version:5.9
//
// Auto-generated for the v2026.08.18 release by
// .github/workflows/upstream-watch.yml. The `main` branch
// keeps a local `binaryTarget(path:)` variant for in-tree
// development; this variant lives only on the tag.

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
        .binaryTarget(
            name: "EverywhereCore",
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.08.18/EverywhereCore-v2026.08.18.xcframework.zip",
            checksum: "63bd44a8c11c0ffad81683a5d299184ad7ee1f016837f88b6175ebb4d794ad95"
        ),
    ]
)
