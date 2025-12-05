# L8MyFamily

A family location tracking system that allows you to monitor the real-time location of family members across multiple devices.

## Overview

L8MyFamily provides a complete solution for family location tracking with:

- **Web Dashboard** - View all family members' locations on an interactive map
- **Laptop Agent** - Track Linux/desktop devices using GeoClue or IP-based geolocation
- **Android Agent** - Track mobile devices using GPS with background location support
- **RESTful API** - Secure backend services for device management and location updates

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Android Agent  │     │  Laptop Agent   │     │   Web Browser   │
│     (GPS)       │     │   (GeoClue/IP)  │     │   (Dashboard)   │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         │      HTTPS + Auth     │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │      Web Server         │
                    │   (REST API + WebUI)    │
                    └────────────┬────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
     ┌────────▼────────┐ ┌───────▼───────┐ ┌───────▼───────┐
     │ Device Service  │ │Location Service│ │ Health Service│
     └─────────────────┘ └───────────────┘ └───────────────┘
```

## Features

- Real-time location tracking
- Multi-device support per family member
- Secure authentication with bearer tokens
- Encrypted credential storage on agents
- TLS/HTTPS communication
- Background location tracking on mobile
- Automatic device registration
- Activity tracking support

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- For Android agent:
  - Android SDK
  - Android NDK
  - gomobile

## Project Structure

```
l8myfamiliy/
├── go/
│   ├── myf/
│   │   ├── agent/
│   │   │   ├── android/     # Android location agent
│   │   │   │   └── mfagent/ # Go library for Android (gomobile)
│   │   │   └── laptop/      # Linux laptop location agent
│   │   ├── device_service/  # Device management service
│   │   ├── location_service/# Location update service
│   │   └── webui/           # Web server and dashboard
│   │       └── web/         # Static web files (HTML/CSS/JS)
│   ├── types/
│   │   └── l8myfamily/      # Protocol buffer generated types
│   └── tests/               # Test files
└── README.md
```

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/saichler/l8myfamiliy.git
cd l8myfamiliy/go
```

### 2. Install dependencies

```bash
go mod download
go mod vendor
```

### 3. Build the Web Server

```bash
cd myf/webui
go build -o l8myfamily-server
```

### 4. Build the Laptop Agent

```bash
cd myf/agent/laptop
go build -o l8myfamily-laptop
```

### 5. Build the Android Agent

See [Android Agent README](go/myf/agent/android/README.md) for detailed build instructions.

```bash
cd myf/agent/android
./build-apk.sh
```

## Configuration

### Web Server

The web server runs on port 9093 by default with HTTPS enabled. Configure the certificate path and other settings in the main.go file.

### Laptop Agent

On first run, the agent will prompt for:
- Device name
- Server URL (default: `https://www.probler.dev:9092`)
- TLS certificate validation preference
- Username and password

Configuration is stored in `~/.config/l8myfamily/laptop-agent.json` with encrypted credentials.

### Android Agent

Configure through the app UI:
- Device name
- Server endpoint URL
- Username and password

## Usage

### Starting the Web Server

```bash
./l8myfamily-server
```

Access the dashboard at `https://your-server:9093`

### Running the Laptop Agent

```bash
./l8myfamily-laptop
```

The agent will:
1. Authenticate with the server
2. Register the device
3. Post location updates every 10 seconds

Location sources (in order of preference):
1. GeoClue (Linux system location service)
2. IP-based geolocation fallback

### Running the Android Agent

1. Install the APK on your device
2. Open the app and configure settings
3. Grant location permissions
4. Tap "Start" to begin tracking

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth` | POST | Authenticate and receive bearer token |
| `/my-family/53/Family` | GET | List all devices |
| `/my-family/53/Family` | POST | Register a device |
| `/my-family/53/Location` | POST | Update device location |

### Location Payload

```json
{
  "device_id": "uuid-string",
  "latitude": 37.7749,
  "longitude": -122.4194
}
```

### Device Registration Payload

```json
{
  "id": "uuid-string",
  "name": "My Device",
  "familyId": "username"
}
```

## Data Model

| Entity | Description |
|--------|-------------|
| **Family** | A group of members with shared access |
| **Member** | A person in a family with one or more devices |
| **Device** | A trackable device (phone, laptop, etc.) |
| **Location** | GPS coordinates for a device |
| **Activity** | Scheduled activities for members |

## Security

- All communication uses HTTPS/TLS
- Authentication via username/password with bearer tokens
- Credentials stored encrypted on agents using AES-GCM
- Device-specific encryption keys derived from device ID

## Dependencies

- [l8bus](https://github.com/saichler/l8bus) - Virtual network overlay
- [l8services](https://github.com/saichler/l8services) - Service framework
- [l8web](https://github.com/saichler/l8web) - Web server utilities
- [l8types](https://github.com/saichler/l8types) - Common type definitions
- [Protocol Buffers](https://protobuf.dev/) - Data serialization

## License

This project is proprietary software.

## Contributing

Contributions are welcome. Please open an issue to discuss proposed changes before submitting a pull request.
