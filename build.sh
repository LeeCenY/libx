#!/bin/bash

rm -fr *.xcframework

go mod init libxray
go mod tidy
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
go get -d golang.org/x/mobile/cmd/gomobile
gomobile bind -target ios
mv Libxray.xcframework ios.xcframework
gomobile bind -target macos
mv Libxray.xcframework macos.xcframework
xcodebuild -create-xcframework -framework ios.xcframework/ios-arm64/Libxray.framework -framework ios.xcframework/ios-arm64_x86_64-simulator/Libxray.framework -framework macos.xcframework/macos-arm64_x86_64/Libxray.framework -output Libxray.xcframework
