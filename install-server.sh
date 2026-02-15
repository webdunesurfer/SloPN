#!/bin/bash

# SloPN One-Click Server Installer
# Author: webdunesurfer
# License: GNU GPLv3
# Version: 0.9.2
# Updated: 2026-02-15 20:00:00

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}====================================================${NC}"
echo -e "${BLUE}          SloPN Server Installation Script          ${NC}"
echo -e "${BLUE}                Version: 0.9.2                      ${NC}"
echo -e "${BLUE}====================================================${NC}"

# 1. Dependency Check
echo -e "\n${BLUE}[1/5] Checking dependencies...${NC}"
for cmd in git docker openssl curl; do
    if ! command -v $cmd &> /dev/null; then
        echo -e "${RED}Error: $cmd is not installed. Please install it and try again.${NC}"
        exit 1
    fi
done
echo -e "${GREEN}Dependencies OK.${NC}"

# 2. Download/Update SloPN
echo -e "\n${BLUE}[2/5] Downloading latest SloPN...${NC}"
if [ -d "SloPN" ]; then
    echo "Existing SloPN directory found. Updating..."
    cd SloPN
    git fetch --all
    git reset --hard origin/main
else
    git clone https://github.com/webdunesurfer/SloPN.git
    cd SloPN
fi

# 3. Generate Secure Configuration
echo -e "\n${BLUE}[3/5] Generating secure configuration...${NC}"
TOKEN=$(openssl rand -hex 16)
VERSION=$(grep "const ServerVersion =" cmd/server/main.go | cut -d'"' -f2 || echo "0.7.3")
# Force IPv4
PUBLIC_IP=$(curl -4s https://ifconfig.me || echo "your-server-ip")

# Use /dev/tty to ensure 'read' works when script is piped from curl
if [ -t 0 ]; then
    read -p "Enter mimic target (SNI) [default: www.google.com:443]: " INPUT_MIMIC
else
    read -p "Enter mimic target (SNI) [default: www.google.com:443]: " INPUT_MIMIC < /dev/tty
fi

USER_MIMIC=${INPUT_MIMIC:-"www.google.com:443"}
MIMIC_HOST=$(echo "$USER_MIMIC" | cut -d: -f1)

# 4. Build and Run Docker Containers
echo -e "\n${BLUE}[4/5] Building and starting Docker containers...${NC}"

# A) Build VPN Server using best available engine
if docker buildx version &>/dev/null; then
    docker buildx build -t slopn-server .
else
    # Fallback to standard legacy builder, explicitly disabling BuildKit to avoid shell errors
    DOCKER_BUILDKIT=0 docker build -t slopn-server .
fi

# B) Start VPN Server
docker stop slopn-server &>/dev/null || true
docker rm slopn-server &>/dev/null || true
docker run -d --name slopn-server --restart unless-stopped --cap-add=NET_ADMIN --device=/dev/net/tun:/dev/net/tun -p 4242:4242/udp -e SLOPN_TOKEN="$TOKEN" -e SLOPN_NAT=true -e SLOPN_MAX_ATTEMPTS=5 -e SLOPN_WINDOW=5 -e SLOPN_BAN_DURATION=60 -e SLOPN_MIMIC="$USER_MIMIC" slopn-server -nat

# C) Start CoreDNS
docker stop slopn-dns &>/dev/null || true
docker rm slopn-dns &>/dev/null || true
# Ensure config is readable by container user
chmod 644 coredns.conf
# Run in standard bridge mode, map port 53 to host (it won't conflict because we'll bind to the Docker Bridge IP)
DOCKER_BRIDGE_IP=$(ip addr show docker0 | grep "inet " | awk '{print $2}' | cut -d/ -f1 || echo "172.17.0.1")
docker run -d --name slopn-dns --restart unless-stopped -p $DOCKER_BRIDGE_IP:53:53/udp -p $DOCKER_BRIDGE_IP:53:53/tcp -v $(pwd)/coredns.conf:/etc/coredns/Corefile coredns/coredns:latest -conf /etc/coredns/Corefile

# 5. Final Report
echo -e "\n${BLUE}[5/5] Installation Complete!${NC}"
echo -e "${BLUE}====================================================${NC}"
echo -e "${GREEN}SloPN Infrastructure v$VERSION is now running!${NC}"
echo -e "\n${BLUE}Client Configuration Details:${NC}"
echo -e "  ${BLUE}Server Address:${NC} $PUBLIC_IP:4242"
echo -e "  ${BLUE}Auth Token:    ${NC} $TOKEN"
echo -e "  ${BLUE}SNI Value:     ${NC} $MIMIC_HOST"
echo -e "\n${BLUE}Management Commands:${NC}"
echo -e "  View Server Logs: ${GREEN}docker logs -f slopn-server${NC}"
echo -e "  View DNS Logs:    ${GREEN}docker logs -f slopn-dns${NC}"
echo -e "  Stop All:         ${GREEN}docker stop slopn-server slopn-dns${NC}"
echo -e "${BLUE}====================================================${NC}"
