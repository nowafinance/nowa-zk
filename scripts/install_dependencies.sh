#!/bin/bash

# Install dependencies for Tan-ZK on a fresh Linux (Ubuntu) VM
# Usage: ./install_dependencies.sh

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}>>> Starting Tan-ZK Dependency Installation...${NC}"

# 1. System Update & Essential Tools
echo -e "${GREEN}[1/4] Updating system and installing base tools...${NC}"
sudo apt update && sudo apt upgrade -y
sudo apt install -y make git build-essential curl wget

# 2. Install Go 1.23.2
GO_VER="1.23.2"
echo -e "${GREEN}[2/4] Installing Go ${GO_VER}...${NC}"

if command -v go &> /dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}')
    echo -e "${YELLOW}Go is already installed: $CURRENT_GO${NC}"
    echo "Skipping Go installation to avoid conflicts. Verify version is >= 1.21 manually."
else
    echo "Downloading Go ${GO_VER}..."
    curl -OL "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz"
    
    echo "Installing to /usr/local/go..."
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "go${GO_VER}.linux-amd64.tar.gz"
    rm "go${GO_VER}.linux-amd64.tar.gz"
    
    # Setup Path for current session
    export PATH=$PATH:/usr/local/go/bin
    
    # Persist Path
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo "Added Go to PATH in ~/.bashrc"
    fi
fi

# 3. Install Foundry
echo -e "${GREEN}[3/4] Installing Foundry...${NC}"
if command -v forge &> /dev/null; then
    echo -e "${YELLOW}Foundry is already installed.${NC}"
else
    curl -L https://foundry.paradigm.xyz | bash
    
    # Source foundry env if it was just added
    if [ -f ~/.foundry/bin/foundryup ]; then
        export PATH="$HOME/.foundry/bin:$PATH"
    fi
    
    # Run foundryup to install binaries
    echo "Running foundryup..."
    bash -c "source ~/.bashrc && foundryup" || ~/.foundry/bin/foundryup
fi

# 4. Final verification
echo -e "${BLUE}===============================================${NC}"
echo -e "${GREEN}Installation Complete!${NC}"
echo -e "${BLUE}===============================================${NC}"
echo ""
echo "Please run the following command to update your current shell:"
echo -e "${YELLOW}source ~/.bashrc${NC}"
echo ""
echo "Versions installed:"
echo -n "Go: "
/usr/local/go/bin/go version || echo "Not found"
echo -n "Forge: "
~/.foundry/bin/forge --version || echo "Not found"
echo ""