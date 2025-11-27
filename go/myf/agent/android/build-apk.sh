#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_MODULE_DIR="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

echo "=== MyFamily Agent APK Builder ==="
echo ""

# Check for required tools
check_tool() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is not installed"
        echo "$2"
        exit 1
    fi
}

check_tool "go" "Install Go from https://golang.org/dl/"
check_tool "gomobile" "Install with: go install golang.org/x/mobile/cmd/gomobile@latest"

if [ -z "$ANDROID_HOME" ] && [ -z "$ANDROID_SDK_ROOT" ]; then
    echo "Error: ANDROID_HOME or ANDROID_SDK_ROOT must be set"
    echo "Install Android SDK and set the environment variable"
    exit 1
fi

ANDROID_SDK="${ANDROID_HOME:-$ANDROID_SDK_ROOT}"
echo "Android SDK: $ANDROID_SDK"

# Check for NDK
NDK_DIR=""
if [ -d "$ANDROID_SDK/ndk" ]; then
    NDK_DIR=$(ls -d "$ANDROID_SDK/ndk"/*/ 2>/dev/null | head -n1)
fi
if [ -z "$NDK_DIR" ] && [ -d "$ANDROID_SDK/ndk-bundle" ]; then
    NDK_DIR="$ANDROID_SDK/ndk-bundle"
fi
if [ -z "$NDK_DIR" ]; then
    echo "Error: Android NDK not found"
    echo "Install NDK via Android SDK Manager"
    exit 1
fi
echo "Android NDK: $NDK_DIR"

# Initialize gomobile if needed
echo ""
echo "=== Initializing gomobile ==="
gomobile init

# Build Go library as AAR
echo ""
echo "=== Building Go library (mfagent.aar) ==="
cd "$SCRIPT_DIR/mfagent"

gomobile bind -v -target=android -androidapi 24 -o "$SCRIPT_DIR/app/libs/mfagent.aar" .

if [ ! -f "$SCRIPT_DIR/app/libs/mfagent.aar" ]; then
    echo "Error: Failed to build mfagent.aar"
    exit 1
fi
echo "Built: $SCRIPT_DIR/app/libs/mfagent.aar"

# Build APK with Gradle
echo ""
echo "=== Building APK with Gradle ==="
cd "$SCRIPT_DIR"

# Use gradle wrapper if available, otherwise use system gradle
if [ -f "./gradlew" ]; then
    chmod +x ./gradlew
    ./gradlew assembleDebug
else
    if command -v gradle &> /dev/null; then
        gradle assembleDebug
    else
        echo "Error: Gradle not found"
        echo "Install Gradle or use the gradle wrapper"
        echo ""
        echo "To generate gradle wrapper, run:"
        echo "  gradle wrapper"
        exit 1
    fi
fi

# Find and report APK location
APK_PATH="$SCRIPT_DIR/app/build/outputs/apk/debug/app-debug.apk"
if [ -f "$APK_PATH" ]; then
    echo ""
    echo "=== Build Successful ==="
    echo "APK location: $APK_PATH"
    echo ""
    echo "To install on device:"
    echo "  adb install $APK_PATH"
    echo ""
    echo "To install on emulator:"
    echo "  adb -e install $APK_PATH"
else
    echo "Error: APK not found at expected location"
    find "$SCRIPT_DIR/app/build" -name "*.apk" 2>/dev/null || true
    exit 1
fi
