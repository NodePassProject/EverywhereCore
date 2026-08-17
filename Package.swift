// swift-tools-version:5.9
//
// Auto-generated for the v2026.08.17 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.08.17/EverywhereCore-v2026.08.17.xcframework.zip",
            checksum: "ec094c524e5b0dd9a5825ce494cebd23137af335797cd65e78a6ed53d77b4d92"
        ),
    ]
)
