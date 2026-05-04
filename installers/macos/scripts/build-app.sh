#!/bin/bash
# Build AgentFS macOS .app bundle
# Usage: ./build-app.sh [--sign IDENTITY] [--notarize]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/build/macos"
APP_NAME="AgentFS"
APP_BUNDLE="$BUILD_DIR/$APP_NAME.app"
VERSION=$(grep 'version = ' "$PROJECT_ROOT/cmd/agentfs/main.go" | head -1 | cut -d'"' -f2)

SIGN_IDENTITY=""
NOTARIZE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --sign)
            SIGN_IDENTITY="$2"
            shift 2
            ;;
        --notarize)
            NOTARIZE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "Building AgentFS.app v$VERSION..."

# Clean previous build
rm -rf "$APP_BUNDLE"
mkdir -p "$APP_BUNDLE/Contents/MacOS"
mkdir -p "$APP_BUNDLE/Contents/Resources"
mkdir -p "$APP_BUNDLE/Contents/Library/LaunchAgents"
mkdir -p "$APP_BUNDLE/Contents/Library/Spotlight"

# Build Go binaries for both architectures
echo "Building binaries..."
cd "$PROJECT_ROOT"

# Build for arm64
GOOS=darwin GOARCH=arm64 go build -tags "fts5" -o "$BUILD_DIR/agentfs-arm64" ./cmd/agentfs

# Build for amd64
GOOS=darwin GOARCH=amd64 go build -tags "fts5" -o "$BUILD_DIR/agentfs-amd64" ./cmd/agentfs

# Create universal binary
echo "Creating universal binary..."
lipo -create -output "$APP_BUNDLE/Contents/MacOS/agentfs" \
    "$BUILD_DIR/agentfs-arm64" \
    "$BUILD_DIR/agentfs-amd64"

# Build Wails UI if available
if [ -d "$PROJECT_ROOT/desktop/agentfs-ui" ]; then
    echo "Building Wails UI..."
    cd "$PROJECT_ROOT/desktop/agentfs-ui"

    # Build for arm64
    GOOS=darwin GOARCH=arm64 wails build -platform darwin/arm64 -o agentfs-ui-arm64

    # Build for amd64
    GOOS=darwin GOARCH=amd64 wails build -platform darwin/amd64 -o agentfs-ui-amd64

    # Create universal binary
    lipo -create -output "$APP_BUNDLE/Contents/MacOS/agentfs-ui" \
        build/bin/agentfs-ui-arm64 \
        build/bin/agentfs-ui-amd64
fi

# Copy Info.plist
cat > "$APP_BUNDLE/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>agentfs-ui</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>org.agentfs.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>LSUIElement</key>
    <false/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright © 2024 AgentFS. All rights reserved.</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.developer-tools</string>
    <key>CFBundleURLTypes</key>
    <array>
        <dict>
            <key>CFBundleURLName</key>
            <string>AgentFS URL</string>
            <key>CFBundleURLSchemes</key>
            <array>
                <string>agentfs</string>
            </array>
        </dict>
    </array>
</dict>
</plist>
EOF

# Copy LaunchAgent
cp "$SCRIPT_DIR/../launchd/org.agentfs.daemon.plist" \
    "$APP_BUNDLE/Contents/Library/LaunchAgents/"

# Copy Spotlight importer if built
if [ -d "$PROJECT_ROOT/installers/spotlight/AgentFSImporter.mdimporter" ]; then
    cp -R "$PROJECT_ROOT/installers/spotlight/AgentFSImporter.mdimporter" \
        "$APP_BUNDLE/Contents/Library/Spotlight/"
fi

# Copy resources
if [ -f "$SCRIPT_DIR/../resources/AppIcon.icns" ]; then
    cp "$SCRIPT_DIR/../resources/AppIcon.icns" "$APP_BUNDLE/Contents/Resources/"
fi

# Create PkgInfo
echo -n "APPL????" > "$APP_BUNDLE/Contents/PkgInfo"

# Sign the app if identity provided
if [ -n "$SIGN_IDENTITY" ]; then
    echo "Signing app with identity: $SIGN_IDENTITY"

    # Sign all executables
    codesign --force --options runtime --sign "$SIGN_IDENTITY" \
        --entitlements "$SCRIPT_DIR/../resources/entitlements.plist" \
        "$APP_BUNDLE/Contents/MacOS/agentfs"

    if [ -f "$APP_BUNDLE/Contents/MacOS/agentfs-ui" ]; then
        codesign --force --options runtime --sign "$SIGN_IDENTITY" \
            --entitlements "$SCRIPT_DIR/../resources/entitlements.plist" \
            "$APP_BUNDLE/Contents/MacOS/agentfs-ui"
    fi

    # Sign the bundle
    codesign --force --options runtime --sign "$SIGN_IDENTITY" \
        --entitlements "$SCRIPT_DIR/../resources/entitlements.plist" \
        "$APP_BUNDLE"

    # Verify signature
    codesign --verify --verbose "$APP_BUNDLE"
fi

# Notarize if requested
if [ "$NOTARIZE" = true ] && [ -n "$SIGN_IDENTITY" ]; then
    echo "Notarizing app..."

    # Create zip for notarization
    ditto -c -k --keepParent "$APP_BUNDLE" "$BUILD_DIR/AgentFS.zip"

    # Submit for notarization (requires APPLE_ID and APP_PASSWORD env vars)
    xcrun notarytool submit "$BUILD_DIR/AgentFS.zip" \
        --apple-id "$APPLE_ID" \
        --password "$APP_PASSWORD" \
        --team-id "$TEAM_ID" \
        --wait

    # Staple the ticket
    xcrun stapler staple "$APP_BUNDLE"
fi

echo "✅ Built: $APP_BUNDLE"

# Create DMG
echo "Creating DMG..."
DMG_PATH="$BUILD_DIR/AgentFS-$VERSION.dmg"

# Create temporary DMG directory
DMG_DIR="$BUILD_DIR/dmg"
rm -rf "$DMG_DIR"
mkdir -p "$DMG_DIR"

cp -R "$APP_BUNDLE" "$DMG_DIR/"
ln -s /Applications "$DMG_DIR/Applications"

# Create DMG
hdiutil create -volname "AgentFS $VERSION" -srcfolder "$DMG_DIR" \
    -ov -format UDZO "$DMG_PATH"

# Sign DMG if identity provided
if [ -n "$SIGN_IDENTITY" ]; then
    codesign --force --sign "$SIGN_IDENTITY" "$DMG_PATH"
fi

rm -rf "$DMG_DIR"
rm -f "$BUILD_DIR/agentfs-arm64" "$BUILD_DIR/agentfs-amd64"

echo "✅ Created: $DMG_PATH"
