// swift-tools-version:5.9
//
// Auto-generated for the v2026.10.01 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.10.01/EverywhereCore-v2026.10.01.xcframework.zip",
            checksum: "00a23b308be83506addbafcc23129f9958e6087f874c2d9e9136dcb22279ad14"
        ),
    ]
)
