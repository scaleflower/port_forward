# Port Forward Manager (PFM)

A cross-platform port forwarding manager with GUI and CLI support, built on [gost](https://github.com/go-gost/x) core library.

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
- **Global Hotkey**: Quick window toggle with customizable hotkey (default: Cmd+Shift+P)
- **System Tray**: Menu bar/tray icon for quick access
- **Cross-Platform**: macOS GUI, Windows GUI, Linux CLI

## Platform Support

| Platform | Mode | Features |
|----------|------|----------|
| macOS | GUI + Service | Full GUI, system tray, global hotkey, LaunchDaemon |
| Windows | GUI + Service | Full GUI, system tray, global hotkey, Windows Service |
| Linux | CLI + Service | Command-line interface, systemd service |
| Docker | CLI + Container | Containerized deployment, volume persistence |

## Tech Stack

| Layer | Technology | Description |
|-------|------------|-------------|
| Core Engine | go-gost/x | Full protocol support |
| GUI Framework | Wails v2 | Go + Web hybrid |
| Frontend | Vue 3 + TypeScript | Component-based UI |
| UI Components | Element Plus | Vue 3 component library |
| State Management | Pinia | Vue 3 state management |
| Build Tool | Vite | Fast development |

---

## Installation

### Prerequisites

- Go 1.21+
- Node.js 18+ (for GUI builds)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) (for GUI builds)

```bash
# Install Wails CLI (for GUI builds only)
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/scaleflower/port_forward.git
cd port_forward/pfm

# Install frontend dependencies
cd frontend && npm install && cd ..
```

### Build Options

| Platform | Command | Output |
|----------|---------|--------|
| macOS GUI (Apple Silicon) | `wails build -platform darwin/arm64` | `build/bin/pfm.app` |
| macOS GUI (Intel) | `wails build -platform darwin/amd64` | `build/bin/pfm.app` |
| Windows GUI | `wails build -platform windows/amd64` | `build/bin/pfm.exe` |
| Linux CLI | `GOOS=linux GOARCH=amd64 go build -tags nogui -o pfm .` | `pfm` |

---

## Deployment

### macOS (GUI Mode)

1. **Build the application**:
   ```bash
   wails build -platform darwin/arm64
   ```

2. **Run the app**:
   - Double-click `build/bin/pfm.app`
   - Or copy to `/Applications/` for permanent installation

3. **Install as system service** (optional):
   - Open the app → Settings → Service Management → Install Service
   - Service auto-starts on boot and keeps running in background
   - Only authorization needed during installation (not every startup)

4. **Global Hotkey**:
   - Default: `Cmd + Shift + P` to show/hide window
   - Requires Accessibility permission: System Settings → Privacy & Security → Accessibility

### Windows (GUI Mode)

1. **Build the application**:
   ```bash
   wails build -platform windows/amd64
   ```

2. **Run the app**:
   - Double-click `build/bin/pfm.exe`

3. **Install as system service**:
   - Open the app → Settings → Service Management → Install Service

### Linux (CLI Mode)

Linux uses a headless CLI-only version, perfect for servers without X Window.

#### Quick Install

```bash
# 1. Build on your dev machine
GOOS=linux GOARCH=amd64 go build -tags nogui -o pfm .

# 2. Copy to Linux server
scp pfm scripts/install-linux.sh user@server:/tmp/

# 3. Install on server
ssh user@server
cd /tmp
chmod +x install-linux.sh
sudo ./install-linux.sh
```

#### Manual Install

```bash
# Copy binary
sudo cp pfm /usr/local/bin/
sudo chmod +x /usr/local/bin/pfm

# Create systemd service
sudo tee /etc/systemd/system/pfm.service << 'EOF'
[Unit]
Description=Port Forward Manager Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/pfm service run
Restart=always
RestartSec=5
User=root
Environment=HOME=/var/lib/pfm

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo mkdir -p /var/lib/pfm
sudo systemctl daemon-reload
sudo systemctl enable pfm
sudo systemctl start pfm
```

#### Service Management (Linux)

```bash
sudo systemctl status pfm      # Check status
sudo systemctl start pfm       # Start service
sudo systemctl stop pfm        # Stop service
sudo systemctl restart pfm     # Restart service
journalctl -u pfm -f           # View logs
```

### Docker Deployment

Docker provides the easiest deployment method with automatic container management.

#### Quick Start with Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/scaleflower/port_forward.git
cd port_forward/pfm

# Start with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

#### Build and Run with Docker

```bash
# Build image
docker build -t pfm:latest .

# Run container (host network mode for port forwarding)
docker run -d \
  --name pfm \
  --network host \
  --restart unless-stopped \
  -v pfm-data:/data \
  -e TZ=Asia/Shanghai \
  pfm:latest

# View logs
docker logs -f pfm

# Execute CLI commands
docker exec pfm pfm status
docker exec pfm pfm rule list
```

#### Docker Compose Configuration

```yaml
version: '3.8'

services:
  pfm:
    image: pfm:latest
    container_name: pfm
    restart: unless-stopped

    # Host network mode (recommended for port forwarding)
    network_mode: host

    # Alternative: Bridge mode with port mappings
    # ports:
    #   - "10000-10100:10000-10100"

    volumes:
      # Named volume for persistent data
      - pfm-data:/data

      # Or bind mount to local directory
      # - ./data:/data

    environment:
      - TZ=Asia/Shanghai
      - PFM_DATA_DIR=/data

volumes:
  pfm-data:
```

#### Network Modes

| Mode | Command | Use Case |
|------|---------|----------|
| **Host** | `network_mode: host` | Port forwarding (recommended) - container shares host network |
| **Bridge** | `ports: ["8080:8080"]` | Isolated network with explicit port mappings |

#### Data Persistence

Data is stored in `/data` inside the container:

```bash
# Using named volume (recommended)
-v pfm-data:/data

# Using bind mount (for easy backup/editing)
-v /path/on/host:/data
```

Data files:
- `/data/data.json` - Rules, chains, and configuration

#### Managing Rules via CLI

```bash
# Check status
docker exec pfm pfm status

# List rules
docker exec pfm pfm rule list

# Start/stop a rule
docker exec pfm pfm rule start <rule-id>
docker exec pfm pfm rule stop <rule-id>

# Create a new rule
docker exec pfm pfm rule create '{"name":"MySQL","type":"forward","protocol":"tcp","localPort":13306,"targetHost":"192.168.1.100","targetPort":3306}'
```

#### Pre-configured Rules

You can mount a pre-configured `data.json` file:

```bash
# Create config directory
mkdir -p ./data

# Create initial configuration
cat > ./data/data.json << 'EOF'
{
  "config": {
    "logLevel": "info",
    "autoStart": true
  },
  "rules": [
    {
      "id": "rule-001",
      "name": "MySQL Forward",
      "type": "forward",
      "protocol": "tcp",
      "localPort": 13306,
      "targetHost": "192.168.1.100",
      "targetPort": 3306,
      "enabled": true,
      "status": "stopped"
    }
  ],
  "chains": []
}
EOF

# Run with bind mount
docker run -d --name pfm --network host -v ./data:/data pfm:latest
```

---

## CLI Commands

The CLI is available on all platforms (run `pfm help` for full list):

```
Usage:
  pfm <command> [arguments]

Commands:
  service     Manage the background service
  rule        Manage port forwarding rules
  chain       Manage proxy chains
  status      Show service and rules status
  version     Show version information
  help        Show help message
```

### Service Commands

```bash
pfm service run         # Run as foreground service (for systemd/init)
pfm service install     # Install as system service
pfm service uninstall   # Uninstall system service
pfm service status      # Show service status
```

### Rule Commands

```bash
pfm rule list                  # List all rules
pfm rule show <id>             # Show rule details
pfm rule start <id>            # Start a rule
pfm rule stop <id>             # Stop a rule
pfm rule delete <id>           # Delete a rule
pfm rule create '<json>'       # Create rule from JSON
```

### Chain Commands

```bash
pfm chain list                 # List all chains
pfm chain show <id>            # Show chain details
pfm chain delete <id>          # Delete a chain
```

### Status Command

```bash
pfm status                     # Show overall status and rule list
```

### Examples

```bash
# Check service and rules status
pfm status

# List all forwarding rules
pfm rule list

# Start a specific rule
pfm rule start 41cdc69b

# Create a new rule
pfm rule create '{"name":"MySQL","type":"forward","protocol":"tcp","localPort":13306,"targetHost":"192.168.1.100","targetPort":3306}'
```

---

## Configuration

### Data Storage

| Platform | Location |
|----------|----------|
| macOS | `~/Library/Application Support/pfm/data.json` |
| Windows | `%APPDATA%\pfm\data.json` |
| Linux | `/var/lib/pfm/data.json` (service) or `~/.config/pfm/data.json` (user) |

### Export/Import

Rules can be exported and imported as JSON via:
- **GUI**: Settings → Data Management → Export/Import
- **CLI**: Manual JSON file editing

### Settings

| Setting | Description | Default |
|---------|-------------|---------|
| Log Level | debug, info, warn, error | info |
| Auto Start | Start enabled rules on service startup | true |
| System Tray | Show tray icon (GUI only) | true |
| Global Hotkey | Enable hotkey (GUI only) | true |
| Hotkey Combo | Modifier + Key | Cmd+Shift+P (macOS) |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PFM Architecture                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐     IPC (Unix Socket)    ┌─────────────┐  │
│  │   GUI App   │ ◄─────────────────────► │   Service   │  │
│  │  (Wails)    │                          │  (Daemon)   │  │
│  └─────────────┘                          └──────┬──────┘  │
│                                                  │         │
│  ┌─────────────┐                                 │         │
│  │   CLI App   │ ◄───────────────────────────────┤         │
│  │  (nogui)    │                                 │         │
│  └─────────────┘                                 ▼         │
│                                           ┌─────────────┐  │
│                                           │ gost Engine │  │
│                                           │  (go-gost)  │  │
│                                           └─────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Components

| Component | Description |
|-----------|-------------|
| GUI App | Wails-based desktop application with Vue 3 frontend |
| CLI App | Headless command-line interface (Linux servers) |
| Service | Background daemon managing port forwarding |
| IPC | JSON-RPC over Unix socket for GUI/CLI ↔ Service communication |
| gost Engine | Core proxy/forwarding engine from go-gost/x |

---

## Project Structure

```
pfm/
├── main.go                    # GUI app entry point
├── main_nogui.go              # CLI-only entry point (build tag: nogui)
├── app.go                     # Wails application bindings
├── wails.json                 # Wails configuration
├── frontend/                  # Vue 3 frontend
│   ├── src/
│   │   ├── views/             # Page components
│   │   ├── stores/            # Pinia stores
│   │   ├── types/             # TypeScript types
│   │   └── App.vue
│   └── package.json
├── internal/
│   ├── engine/                # gost engine wrapper
│   ├── models/                # Data models
│   ├── storage/               # JSON file persistence
│   ├── daemon/                # System service support
│   ├── ipc/                   # GUI/CLI-Service communication
│   ├── cli/                   # CLI command handlers
│   ├── tray/                  # System tray support
│   └── hotkey/                # Global hotkey support
├── scripts/
│   ├── install-linux.sh       # Linux installation script
│   └── pfm.service            # systemd service template
└── build/
    └── bin/                   # Build output
```

---

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

### Building for Different Platforms

```bash
# macOS (current architecture)
wails build

# macOS (specific architecture)
wails build -platform darwin/arm64
wails build -platform darwin/amd64

# Windows
wails build -platform windows/amd64

# Linux CLI (no GUI dependencies)
GOOS=linux GOARCH=amd64 go build -tags nogui -o pfm .
```

---

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

---

## Troubleshooting

### macOS: Global hotkey not working

Grant Accessibility permission:
1. System Settings → Privacy & Security → Accessibility
2. Enable permission for `pfm.app`

### macOS: Service requires authorization

The service is installed as a LaunchDaemon in `/Library/LaunchDaemons/`, which requires admin privileges. Authorization is only needed during:
- Service installation
- Service uninstallation

Once installed, the service auto-manages itself (auto-start, auto-restart).

### Linux: Service not starting

Check logs:
```bash
journalctl -u pfm -n 50
```

Verify binary permissions:
```bash
ls -la /usr/local/bin/pfm
sudo chmod +x /usr/local/bin/pfm
```

### Connection refused when using CLI

Ensure the service is running:
```bash
pfm service status
# or
sudo systemctl status pfm
```

---

## License

MIT License

## Acknowledgments

- [gost](https://github.com/go-gost/x) - The powerful proxy library
- [Wails](https://wails.io) - Go + Web application framework
- [Element Plus](https://element-plus.org) - Vue 3 component library
