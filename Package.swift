// swift-tools-version:5.9
//
// Auto-generated for the v2026.08.09 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.08.09/EverywhereCore-v2026.08.09.xcframework.zip",
            checksum: "c5cc5d4d2c62393c7e65f1be90c451017a33765058a4a62f38034285a6bf8372"
        ),
    ]
)
