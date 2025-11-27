# MyFamily Android Location Agent

Android app that collects GPS location every 10 seconds and posts it to a configured endpoint.

## Prerequisites

1. **Go** (1.21+): https://golang.org/dl/
2. **gomobile**: Install with:
   ```bash
   go install golang.org/x/mobile/cmd/gomobile@latest
   go install golang.org/x/mobile/cmd/gobind@latest
   ```
3. **Android SDK**: Install via Android Studio or command line tools
4. **Android NDK**: Install via Android SDK Manager
5. **Gradle** (optional): The build script will use system gradle or you can generate a wrapper

## Environment Setup

Set the Android SDK path:
```bash
export ANDROID_HOME=/path/to/android/sdk
# or
export ANDROID_SDK_ROOT=/path/to/android/sdk
```

## Configuration

Edit `mfagent/agent.go` to set default values:
```go
var (
    DeviceID = "android-001"           // Your device ID
    Endpoint = "http://server:port/api/location"  // Your server endpoint
)
```

These can also be changed in the app UI at runtime.

## Building the APK

```bash
./build-apk.sh
```

The APK will be generated at:
```
app/build/outputs/apk/debug/app-debug.apk
```

## Installing on Device

Enable USB debugging on your Android device, then:
```bash
adb install app/build/outputs/apk/debug/app-debug.apk
```

## App Permissions

The app requires the following permissions:
- **Location (Fine/Coarse)**: For GPS tracking
- **Background Location**: To track location when app is in background
- **Internet**: To post location data
- **Foreground Service**: To run location tracking as a service

## Usage

1. Open the app
2. Enter your Device ID
3. Enter your server endpoint URL
4. Tap "Start" to begin tracking
5. Grant location permissions when prompted
6. The app will post location every 10 seconds

## Payload Format

The app posts JSON to the endpoint:
```json
{
  "device_id": "android-001",
  "latitude": 37.7749,
  "longitude": -122.4194
}
```
