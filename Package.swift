// swift-tools-version:5.9
//
// Auto-generated for the v2026.10.24 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.10.24/EverywhereCore-v2026.10.24.xcframework.zip",
            checksum: "af465f77a4c93b16c9a35eeeff5bf3f90bf06790af34b5911954bca9cf8c4fd2"
        ),
    ]
)
