#!/bin/bash
# Helper script to run RPC client tests
# Usage: ./test.sh [unit|integration]

set -e

cd "$(dirname "$0")"

case "${1:-unit}" in
    unit|short)
        echo "Running unit tests (mock server)..."
        go test ./pkg/rpc/... -short -v
        ;;
    integration|int)
        echo "Running integration tests (requires .env with TAN_ZK_RPC_URL)..."
        if [ ! -f .env ]; then
            echo "⚠️  Warning: .env file not found. Creating from .env.example..."
            if [ -f .env.example ]; then
                cp .env.example .env
                echo "✅ Created .env file. Please edit it with your RPC URL."
                echo "   Then run: ./test.sh integration"
                exit 1
            else
                echo "❌ Error: .env.example not found!"
                exit 1
            fi
        fi
        go test ./pkg/rpc/... -tags=integration -v
        ;;
    all)
        echo "Running all tests..."
        go test ./pkg/rpc/... -short -v
        echo ""
        echo "Running integration tests..."
        if [ -f .env ]; then
            go test ./pkg/rpc/... -tags=integration -v
        else
            echo "⚠️  Skipping integration tests: .env file not found"
        fi
        ;;
    *)
        echo "Usage: $0 [unit|integration|all]"
        echo ""
        echo "  unit         - Run unit tests only (mock server, default)"
        echo "  integration  - Run integration tests (requires .env)"
        echo "  all          - Run both unit and integration tests"
        exit 1
        ;;
esac

