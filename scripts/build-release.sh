#!/bin/sh
# Builds the downloads into dist/ (run on macOS: lipo, codesign, iconutil):
#   YouTube-Music-Discord-Setup.exe, YouTube-Music-Discord-Setup-ARM.exe  (icon + details)
#   YouTube-Music-Discord-Setup-Mac.zip  ("YouTube Music Discord Setup.app", universal)
set -eu
cd "$(dirname "$0")/.."
VERSION=$(sed -n 's/^const version = "\(.*\)"$/\1/p' version.go)
APP="YouTube Music Discord Setup.app"
rm -rf dist build
mkdir -p dist build
export CGO_ENABLED=0

# The icon: Windows resources (linked into the .exe) and a macOS iconset.
trap 'rm -f rsrc_windows_*.syso' EXIT
go run ./tools/genicons -version "$VERSION" -syso . -iconset build/AppIcon.iconset

build() { GOOS=$1 GOARCH=$2 go build -trimpath -ldflags='-s -w' -o "$3" .; }
build windows amd64 dist/YouTube-Music-Discord-Setup.exe
build windows arm64 dist/YouTube-Music-Discord-Setup-ARM.exe
build darwin arm64 build/ytm-discord-arm64
build darwin amd64 build/ytm-discord-amd64

# One Mac program for Apple Silicon and Intel, inside an app so it has an icon
# and talks through dialog boxes instead of a Terminal window.
mkdir -p "build/$APP/Contents/MacOS" "build/$APP/Contents/Resources"
lipo -create -output "build/$APP/Contents/MacOS/ytm-discord" build/ytm-discord-arm64 build/ytm-discord-amd64
iconutil -c icns build/AppIcon.iconset -o "build/$APP/Contents/Resources/AppIcon.icns"
cat > "build/$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key><string>io.github.tobsatop.ytm-discord-setup</string>
	<key>CFBundleName</key><string>YouTube Music Discord Setup</string>
	<key>CFBundleDisplayName</key><string>YouTube Music Discord Setup</string>
	<key>CFBundleExecutable</key><string>ytm-discord</string>
	<key>CFBundleIconFile</key><string>AppIcon</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>$VERSION</string>
	<key>CFBundleVersion</key><string>$VERSION</string>
	<key>LSMinimumSystemVersion</key><string>12.0</string>
	<key>LSUIElement</key><true/>
</dict>
</plist>
PLIST
# Apple Silicon refuses unsigned code: sign ad hoc after merging.
codesign --force --deep --sign - "build/$APP"
lipo -info "build/$APP/Contents/MacOS/ytm-discord"
"build/$APP/Contents/MacOS/ytm-discord" --version
# ditto keeps the app bundle intact (permissions, signature).
(cd build && ditto -c -k --norsrc --noextattr --keepParent "$APP" ../dist/YouTube-Music-Discord-Setup-Mac.zip)
(cd dist && shasum -a 256 * > SHA256SUMS.txt && cat SHA256SUMS.txt)
