# Testing Guide: Nowa-ZK App-Specific ZK Validium

Quick reference for testing all components of the Nowa-ZK system with terminal commands.

---

## Quick Setup & Validation

```bash
# Full clean setup
make clean
make setup
make build
make test

# Verify binaries
ls -lh build/
# Should show: indexer-bin, prover-bin
```

---

## 1. Contract Testing

### Unit Tests
```bash
cd contracts
forge test -vv

# Specific test
forge test --match-test testVerifyProof -vvv

# Gas report
forge test --gas-report
```

### Deploy & Verify
```bash
# Deploy to local anvil
make anvil                    # Terminal 1
make deploy                   # Terminal 2

# Check deployment
cat ~/.nowa-zk/deployments.json
```

**Common Errors:**
```bash
# Error: "RollupVerifier.sol not found"
make setup  # Regenerate verifier

# Error: "insufficient funds"
# Check .env has funded private key for anvil
grep PRIVATE_KEY .env
```

---

## 2. Indexer Testing

### Unit Tests
```bash
cd indexer
go test ./... -v

# Specific package
go test ./internal/indexer -v

# Coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Test
```bash
# Start indexer
make run-indexer

# Test API
curl http://localhost:8080/health
curl http://localhost:8080/api/batches/latest
curl http://localhost:8080/prover/batch/latest

# Check metrics
curl http://localhost:8080/metrics
```

### Traffic Generation
```bash
# Generate test transactions
make run-traffic-gen COUNT=1000

# Check batch creation
curl http://localhost:8080/api/batches/latest | jq
```

**Common Errors:**
```bash
# Error: "bind: address already in use"
lsof -i :8080
kill -9 <PID>

# Error: "failed to connect to state db"
rm -rf ~/.nowa-zk/indexer/data
make run-indexer

# Error: "RPC connection failed"
# Check L2_RPC_URL in .env
curl -X POST $L2_RPC_URL -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

---

## 3. Prover Testing

### Unit Tests
```bash
cd prover
go test ./... -v

# Circuit benchmarks
cd circuits
go test -bench=. -benchtime=3s
```

### Setup Validation
```bash
# Check keys exist
ls -lh ~/.nowa-zk/keys/
# Should show: rollup.r1cs, rollup.pk, rollup.vk

# Verify verifier contract generated
ls -lh contracts/src/generated/RollupVerifier.sol
```

### Integration Test
```bash
# Normal operation
make run-prover

# Watch logs
tail -f ~/.nowa-zk/prover/data/*.log  # if logging to file
```

### Error Handling Tests

**Test 1: Paranoid Mode (Proof Rebuild)**
```bash
# Simulate verification failure
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --test-failure

# Expected: 3 retries → rebuild → halt
# Check failure data
ls -lh ~/.nowa-zk/prover/failures/
cat ~/.nowa-zk/prover/failures/batch_*_error.log
```

**Test 2: Recovery from Halt**
```bash
# Try restart (should refuse)
./build/prover-bin start --keys-dir ~/.nowa-zk/keys

# Clear halt and resume
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --clear-halt
```

**Test 3: Disable Paranoid Mode**
```bash
./build/prover-bin start --keys-dir ~/.nowa-zk/keys \
  --test-failure --paranoid-mode=false
# Expected: retries only, no rebuild, continues to next batch
```

**Common Errors:**
```bash
# Error: "failed to load circuit/keys"
make setup  # Regenerate keys
ls -lh ~/.nowa-zk/keys/

# Error: "failed to fetch contract state root"
# Check contract deployed
cat ~/.nowa-zk/deployments.json

# Error: "local verification failed"
# Circuit/key mismatch - regenerate both
cd contracts && forge clean
make setup

# Error: "transaction reverted"
# Check contract address correct
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --contract <ADDR>

# Error: "prover is halted"
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --clear-halt
```

---

## 4. End-to-End Testing

### Complete Flow Test
```bash
# Terminal 1: Blockchain (if using anvil)
make anvil

# Terminal 2: Deploy contracts
make deploy
cat ~/.nowa-zk/deployments.json

# Terminal 3: Indexer
make run-indexer

# Terminal 4: Prover  
make run-prover

# Terminal 5: Generate traffic
make run-traffic-gen COUNT=500

# Verify
curl http://localhost:8080/api/batches/latest | jq '.number'
# Should increment as prover processes batches
```

### State Synchronization Test
```bash
# Stop prover mid-processing
pkill prover-bin

# Check last processed batch
# (stored in ~/.nowa-zk/prover/data/)

# Restart prover
make run-prover
# Should resume from last batch
```

### Performance Test
```bash
# High volume traffic
make run-traffic-gen COUNT=10000

# Monitor prover throughput
watch -n 1 'curl -s http://localhost:8081/health | jq'

# Check batch processing time
# (look for "took=" in prover logs)
```

---

## 5. Common System Errors & Debugging

### Setup Issues

**Error: `make setup` fails**
```bash
# Check Go installed
go version  # Need 1.24.10+

# Check Foundry installed
forge --version

# Clean and retry
make clean
make setup
```

**Error: Keys not generated**
```bash
# Manual key generation
./build/prover-bin setup --output-dir ~/.nowa-zk/keys \
  --contract-output contracts/src/generated

# Verify
ls -lh ~/.nowa-zk/keys/
ls -lh contracts/src/generated/RollupVerifier.sol
```

### Runtime Issues

**Indexer not creating batches**
```bash
# Check RPC connection
curl -X POST $L2_RPC_URL -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Check transactions in mempool
curl http://localhost:8080/api/mempool | jq

# Reset state
rm -rf ~/.nowa-zk/indexer/data
make run-indexer
```

**Prover not processing batches**
```bash
# Check indexer is running
curl http://localhost:8080/health

# Check batches available
curl http://localhost:8080/prover/batch/latest

# Check prover contract address
cat ~/.nowa-zk/deployments.json
# Should match what prover is using

# Manual contract check
cast call <CONTRACT_ADDR> "totalBatches()" --rpc-url $L1_RPC_URL
```

**State root mismatch**
```bash
# Get contract state root
cast call <CONTRACT_ADDR> "getCurrentStateRoot()" --rpc-url $L1_RPC_URL

# Reset prover state (WARNING: deletes progress)
rm -rf ~/.nowa-zk/prover/data
make run-prover
```

**Gas estimation failures**
```bash
# Check wallet has funds
cast balance <YOUR_ADDRESS> --rpc-url $L1_RPC_URL

# Increase gas limit in code if needed
# Or use faster RPC endpoint
```

### Network Issues

**Port conflicts**
```bash
# Find process using port
lsof -i :8080  # Indexer
lsof -i :8081  # Prover API
lsof -i :8545  # Anvil

# Kill process
kill -9 <PID>
```

**RPC timeouts**
```bash
# Test RPC latency
time curl -X POST $L1_RPC_URL -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Use local node or faster endpoint
# Update .env with better RPC
```

### Data Issues

**Corrupted database**
```bash
# Indexer
rm -rf ~/.nowa-zk/indexer/data
make run-indexer

# Prover (WARNING: loses progress)
rm -rf ~/.nowa-zk/prover/data
make run-prover
```

**Disk space full**
```bash
# Check space
df -h ~/.nowa-zk

# Clean old data
rm -rf ~/.nowa-zk/prover/failures/*  # Old test data
# Badger DB auto-compacts but can be manually cleaned
```

---

## 6. Debugging Tools

### Logs
```bash
# Enable debug logging in prover
# (already enabled by default - look for DEBUG: lines)

# Grep for errors
journalctl -u nowa-zk-prover | grep ERROR
# or if running in terminal:
make run-prover 2>&1 | tee prover.log
grep ERROR prover.log
```

### State Inspection
```bash
# Check BadgerDB (Indexer)
# Keys: last_batch_number, batch_<N>, tx_<hash>

# Check BadgerDB (Prover)  
# Keys: last_processed_batch, last_state_root, halt_state, failure_<N>

# Manual inspection (requires badger CLI or custom tool)
```

### Contract Debugging
```bash
# Get batch info from contract
cast call <CONTRACT_ADDR> "getBatch(uint256)" <BATCH_NUM> --rpc-url $L1_RPC_URL

# Get total batches
cast call <CONTRACT_ADDR> "totalBatches()" --rpc-url $L1_RPC_URL

# Check events
cast logs --address <CONTRACT_ADDR> --from-block <START> --to-block latest
```

### Performance Profiling
```bash
# Go profiling
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
go tool pprof cpu.prof

# Circuit benchmarks
cd prover/circuits
go test -bench=BenchmarkProve -benchmem -cpuprofile=cpu.prof
```

---

## 7. CI/CD Testing

```bash
# Full CI test suite
make clean
make deps
make build  
make test

# Expected: All tests pass
# contracts: forge test
# indexer: go test ./...
# prover: go test ./...
```

---

## Quick Reference: Test Commands

| Component | Command | Purpose |
|-----------|---------|---------|
| All | `make test` | Run all unit tests |
| Contracts | `cd contracts && forge test` | Contract unit tests |
| Indexer | `cd indexer && go test ./...` | Indexer unit tests |
| Prover | `cd prover && go test ./...` | Prover unit tests |
| E2E | See section 4 | Full system test |
| Error Handling | `--test-failure` flag | Test paranoid mode |
| Performance | `make run-traffic-gen COUNT=10000` | Load test |

---

## Test Checklist

Before deployment:
- [ ] `make clean && make setup && make build && make test` passes
- [ ] Contracts deploy successfully
- [ ] Indexer creates batches from transactions
- [ ] Prover processes batches and submits proofs
- [ ] Error handling tested (paranoid mode)
- [ ] State recovery tested (restart components)
- [ ] Performance meets requirements (batch time < target)

---

## Getting Help

- Check logs for ERROR/WARN messages
- Verify all environment variables in `.env`
- Ensure all binaries are latest version: `make build`
- Reset state if corrupted: `make clean && make setup`
- Join Discord/GitHub issues for support
