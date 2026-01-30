#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}===============================================${NC}"
echo -e "${BLUE}       Nowa-ZK Storage Usage Measurement        ${NC}"
echo -e "${BLUE}===============================================${NC}"
echo ""

# Defined Paths based on Cloud Deployment Guide
SEQ_STATE_DIR="/var/lib/tan-zk/sequencer"
PROVER_DIR="/var/lib/tan-zk/prover"
USER_CONFIG_DIR="$HOME/.tan-zk"
REPO_BUILD_DIR="$HOME/tan-zk/build"

# Function to measure and print size
measure_dir() {
    local name=$1
    local path=$2
    
    if [ -d "$path" ]; then
        # Get size in human readable format and raw bytes for precision if needed
        local size_human=$(du -sh "$path" 2>/dev/null | cut -f1)
        # Detailed breakdown of top-level items
        echo -e "${GREEN}▶ $name${NC} (Path: $path)"
        echo -e "  Total Size: ${YELLOW}$size_human${NC}"
        echo "  Breakdown:"
        du -h --max-depth=1 "$path" 2>/dev/null | sed 's/^/    /'
    else
        echo -e "${YELLOW}▶ $name${NC}: Directory not found at $path (Is the service running?)"
    fi
    echo ""
}

# 1. Sequencer Storage
# This contains the Chain State / DB. This is the main directory that grows "per batch".
measure_dir "Sequencer Storage (State & DB)" "$SEQ_STATE_DIR"

# 2. Prover Storage
# This contains the Proving Keys (large, static) and any generated proof data.
measure_dir "Prover Storage (Keys & Data)" "$PROVER_DIR"

# 3. Configuration & Deployments
# Contains deployment.json and other small config files.
measure_dir "Config & Deployments" "$USER_CONFIG_DIR"

# 4. Build Artifacts (Optional)
# Binaries and logs in the build folder
if [ -d "$REPO_BUILD_DIR" ]; then
    measure_dir "Build Artifacts" "$REPO_BUILD_DIR"
fi

echo -e "${BLUE}===============================================${NC}"
echo -e "To measure growth per batch:"
echo "1. Run this script and note the 'Sequencer Storage' size."
echo "2. Process a batch."
echo "3. Run this script again and calculate the difference."
echo -e "${BLUE}===============================================${NC}"
