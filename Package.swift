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
            url: "https://github.com/NodePassProject/EverywhereCore/releases/download/v2026.11.27/EverywhereCore-v2026.11.27.xcframework.zip",
            checksum: "5e2b8b8ce357c79a5227688de0b71217608678f2c5a403b71abab1ac4944916c"
        ),
    ]
)
