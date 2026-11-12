// swift-tools-version:5.9
//
// Auto-generated for the v2026.11.12 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.11.12/EverywhereCore-v2026.11.12.xcframework.zip",
            checksum: "c16eb4d3e0815085034e01aee50f85a81a747290a018b1bfe42d4f3190712163"
        ),
    ]
)
