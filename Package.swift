// swift-tools-version:5.9
//
// Auto-generated for the v2026.07.30 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.07.30/EverywhereCore-v2026.07.30.xcframework.zip",
            checksum: "4e00a705238492845f7f66a1eacb0e328459cbae685523b93b731cb6a308628f"
        ),
    ]
)
