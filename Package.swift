// swift-tools-version:5.9
//
// Auto-generated for the v2026.07.19 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.07.19/EverywhereCore-v2026.07.19.xcframework.zip",
            checksum: "946c5a61876d9c112f176f50c577c7efb9b880d6efaa2bd623740b4dcdd8351d"
        ),
    ]
)
