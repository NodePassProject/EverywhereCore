// swift-tools-version:5.9
//
// Auto-generated for the v2026.08.04 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.08.04/EverywhereCore-v2026.08.04.xcframework.zip",
            checksum: "f0e1e6f88029aa9e7a06821abe6526a41830f726555ee4263c9432529215404f"
        ),
    ]
)
