#!/bin/bash
# Port Forward Manager - Linux Installation Script

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Port Forward Manager - Linux Installer${NC}"
echo "========================================"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root (sudo ./install-linux.sh)${NC}"
    exit 1
fi

# Variables
INSTALL_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/pfm.service"
DATA_DIR="/var/lib/pfm"
BINARY_NAME="pfm"

# Check if binary exists in current directory
if [ ! -f "./$BINARY_NAME" ]; then
    echo -e "${RED}Error: $BINARY_NAME binary not found in current directory${NC}"
    echo "Please build the binary first with: GOOS=linux GOARCH=amd64 go build -tags nogui -o pfm ."
    exit 1
fi

echo -e "${YELLOW}Installing $BINARY_NAME to $INSTALL_DIR...${NC}"
cp "./$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

echo -e "${YELLOW}Creating data directory...${NC}"
mkdir -p "$DATA_DIR"

echo -e "${YELLOW}Installing systemd service...${NC}"
cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=Port Forward Manager Service
Documentation=https://github.com/your-repo/pfm
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/pfm service run
Restart=always
RestartSec=5
User=root
Environment=HOME=/var/lib/pfm

StandardOutput=journal
StandardError=journal
SyslogIdentifier=pfm

[Install]
WantedBy=multi-user.target
EOF

echo -e "${YELLOW}Reloading systemd...${NC}"
systemctl daemon-reload

echo -e "${YELLOW}Enabling service to start on boot...${NC}"
systemctl enable pfm

echo -e "${YELLOW}Starting service...${NC}"
systemctl start pfm

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "Usage:"
echo "  pfm status          - Show service status"
echo "  pfm rule list       - List all rules"
echo "  pfm rule start <id> - Start a rule"
echo "  pfm help            - Show all commands"
echo ""
echo "Service management:"
echo "  systemctl status pfm   - Check service status"
echo "  systemctl restart pfm  - Restart service"
echo "  journalctl -u pfm -f   - View logs"
echo ""
