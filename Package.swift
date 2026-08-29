// swift-tools-version:5.9
//
// Auto-generated for the v2026.08.29 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.08.29/EverywhereCore-v2026.08.29.xcframework.zip",
            checksum: "8efd98c00589724488911d1f1888405f19f899e608e0c812af90bbe4f8a8177b"
        ),
    ]
)
