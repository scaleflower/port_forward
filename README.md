# Port Forward Manager (PFM)

A cross-platform port forwarding manager with GUI, built on [gost](https://github.com/go-gost/x) core library.

![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-blue)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- **Port Forwarding**: TCP/UDP port mapping with load balancing support
- **Reverse Proxy**: HTTP/HTTPS reverse proxy with backend routing
- **Proxy Chain**: Multi-hop proxy chains (SOCKS5, HTTP, Shadowsocks)
- **Hot Reload**: Dynamic rule management without restart
- **Environment Tags**: Organize rules by environment (TRUNK, PRE-PROD, PRODUCTION, CUSTOM)
- **System Service**: Run as background service with auto-start
- **Cross-Platform**: macOS, Windows, Linux support

## Screenshots

The application provides a clean, intuitive interface for managing port forwarding rules:

- Environment-based organization with color-coded tags
- One-click rule enable/disable
- Real-time status monitoring

## Tech Stack

| Layer | Technology | Description |
|-------|------------|-------------|
| Core Engine | go-gost/x v0.8.1 | Full protocol support |
| GUI Framework | Wails v2 | Go + Web hybrid |
| Frontend | Vue 3 + TypeScript | Component-based UI |
| UI Components | Element Plus | Mature Vue 3 component library |
| State Management | Pinia | Official Vue 3 state management |
| Build Tool | Vite | Fast development experience |

## Installation

### Prerequisites

- Go 1.21+
- Node.js 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/scaleflower/port_forward.git
cd port_forward

# Install frontend dependencies
cd frontend && npm install && cd ..

# Development mode
wails dev

# Production build
wails build
```

### Cross-Platform Build

```bash
# macOS (Apple Silicon)
wails build -platform darwin/arm64

# macOS (Intel)
wails build -platform darwin/amd64

# Windows
wails build -platform windows/amd64

# Linux
wails build -platform linux/amd64
```

## Usage

### Port Forwarding

Create a port forwarding rule to map a local port to a remote target:

| Field | Description | Example |
|-------|-------------|---------|
| Environment | Rule category | PRODUCTION |
| Purpose | Rule name/description | MySQL Database |
| Target Host | Remote IP or hostname | 192.168.1.100 |
| Target Port | Remote port | 3306 |
| Local Port | Local listening port | 13306 |
| Protocol | TCP or UDP | TCP |

Access the forwarded service via `localhost:LOCAL_PORT`.

### Reverse Proxy

Set up HTTP/HTTPS reverse proxy to route traffic to backend servers.

### Proxy Chain

Create multi-hop proxy chains for complex routing scenarios:

1. Create a chain with multiple hops (SOCKS5, HTTP, Shadowsocks)
2. Create a rule that uses the chain
3. Traffic will be routed through all hops in sequence

## Configuration

Configuration is stored in:
- **macOS**: `~/Library/Application Support/pfm/data.json`
- **Windows**: `%APPDATA%\pfm\data.json`
- **Linux**: `~/.config/pfm/data.json`

### Export/Import

Rules can be exported and imported as JSON for backup or migration.

## System Service

The application can run as a system service for background operation:

1. Go to **Settings** > **Service Management**
2. Click **Install Service**
3. The service will start automatically on system boot

Service locations:
- **macOS**: launchd (`~/Library/LaunchAgents/`)
- **Windows**: Windows Service (SCM)
- **Linux**: systemd (`~/.config/systemd/user/`)

## Project Structure

```
pfm/
├── app.go                 # Wails application bindings
├── main.go                # Application entry point
├── wails.json             # Wails configuration
├── frontend/              # Vue 3 frontend
│   ├── src/
│   │   ├── views/         # Page components
│   │   ├── stores/        # Pinia stores
│   │   ├── types/         # TypeScript types
│   │   └── App.vue
│   └── package.json
├── internal/
│   ├── engine/            # gost engine wrapper
│   ├── models/            # Data models
│   ├── storage/           # JSON file persistence
│   ├── daemon/            # System service support
│   └── ipc/               # GUI-Service communication
├── cmd/
│   └── service/           # Background service entry
└── scripts/               # Utility scripts
```

## Development

### Live Development

```bash
wails dev
```

This starts a Vite dev server with hot reload for frontend changes.

### Generate Bindings

After modifying Go structs exposed to frontend:

```bash
wails generate module
```

## Upgrading gost

To upgrade the gost core library:

```bash
# Check current version
go list -m github.com/go-gost/x

# Update to latest
go get -u github.com/go-gost/x@latest
go mod tidy

# Rebuild
wails build -clean
```

## License

MIT License

## Acknowledgments

- [gost](https://github.com/go-gost/x) - The powerful proxy library
- [Wails](https://wails.io) - Go + Web application framework
- [Element Plus](https://element-plus.org) - Vue 3 component library
