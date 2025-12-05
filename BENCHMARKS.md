# Proof Generation Benchmarks

## Results (2025-12-01)

### Summary
| Batch Size | Time per Batch | Time per Tx | Notes |
|------------|----------------|-------------|-------|
| 1 tx       | ~20 ms         | ~20 ms      | Baseline overhead |
| 128 tx     | ~1.99 s        | ~15.5 ms    | Full batch (BatchSize=128) |

### Detailed Phase Breakdown (128 tx)
| Phase | Time | Description |
|-------|------|-------------|
| **Setup** | ~21.59 s | Circuit compilation + Trusted Setup (Key Generation). Run once. |
| **Witness** | ~0.49 ms | Generating witness from transaction inputs. |
| **Prove** | ~1.99 s | Generating the ZK proof (Groth16). |
| **Verify** | ~0.60 ms | Verifying the proof (off-chain). |

**Total Proving Time (Witness + Prove):** ~1.99 s

## Performance Analysis
## Latest Results (After Optimization)

- **Proof Generation Time**: ~1.70s (was 1.99s)
- **Throughput**: ~75 tx/s (was 64 tx/s)
- **Setup Cost**: ~21.6s (One-time)
- **Verification Time**: ~600µs

### Optimization Details
1. **State Transition Logic**: Refactored to use `TxHash` (leaf) for state transition instead of re-hashing all transaction fields. This reduced the number of MiMC permutations significantly.
2. **Field Packing**: Packed `Nonce`, `GasLimit`, and `GasPrice` into a single field element before hashing, further reducing the number of constraints.

![Benchmark Results](/docs/images/benchmark_results.png)

## Methodology
Benchmarks were run using `go test -bench` on the `rollup_circuit` package.
- **Hardware**: 12th Gen Intel(R) Core(TM) i7-12700H
- **Command**: `cd prover && go test -bench=Benchmark -benchtime=1x -run=^$ -v ./circuits/`
