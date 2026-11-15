// swift-tools-version:5.9

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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.11.15/EverywhereCore-v2026.11.15.xcframework.zip",
            checksum: "9822390d91ecf9b728624462f8a2fd5d8a9ffeec00731cca2e00e5668cbb11fe"
        ),
    ]
)
