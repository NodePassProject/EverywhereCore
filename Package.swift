// swift-tools-version:5.9
//
// Auto-generated for the v2026.07.08 release by
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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.07.08/EverywhereCore-v2026.07.08.xcframework.zip",
            checksum: "cec71ec8ffedc05842c92b77707408559b93fea73fffccf5c4497ada424af7c3"
        ),
    ]
)
