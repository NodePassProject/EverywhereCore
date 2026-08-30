// swift-tools-version:5.9
//
// Auto-generated for the v2026.08.30 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.08.30/EverywhereCore-v2026.08.30.xcframework.zip",
            checksum: "e2d56334a0b34f50a256e4df46056a6f923ffa0d90ab39d7bdb750ddb63a88ff"
        ),
    ]
)
