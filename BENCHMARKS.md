# Proof Generation Benchmarks

## Results (2025-12-01)

### Summary
| Batch Size | Time per Batch | Time per Tx | Notes |
|------------|----------------|-------------|-------|
| 1 tx       | ~20 ms         | ~20 ms      | Baseline overhead |
| 128 tx     | ~1.99 s        | ~15.5 ms    | Full batch (BatchSize=128) |

![Benchmark Results](docs/images/benchmark_results.png)

### Detailed Phase Breakdown (128 tx)
| Phase | Time | Description |
|-------|------|-------------|
| **Setup** | ~21.59 s | Circuit compilation + Trusted Setup (Key Generation). Run once. |
| **Witness** | ~0.49 ms | Generating witness from transaction inputs. |
| **Prove** | ~1.99 s | Generating the ZK proof (Groth16). |
| **Verify** | ~0.60 ms | Verifying the proof (off-chain). |

**Total Proving Time (Witness + Prove):** ~1.99 s

## Performance Analysis
- **Scalability**: The proving time scales linearly with the number of transactions.
- **Throughput**: At ~1.99s for 128 txs, the prover can handle **~64 tx/s** on a single core.
- **Setup Cost**: Setup is expensive (~21.6s) but only needs to be run once per circuit update.
- **Verification**: Extremely fast (~0.6ms), ensuring low latency for verifiers.

## SLA / Performance Targets
Based on these results, we define the following Service Level Agreements (SLA):

1.  **Proof Generation Time**: < 3 seconds for a 128-tx batch.
2.  **Max Latency**: < 5 seconds from batch submission to proof generation.

## Methodology
Benchmarks were run using `go test -bench` on the `rollup_circuit` package.
- **Hardware**: 12th Gen Intel(R) Core(TM) i7-12700H
- **Command**: `cd prover && go test -bench=Benchmark -benchtime=1x -run=^$ -v ./circuits/`
