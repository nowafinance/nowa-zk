# Nowa-ZK Architecture Documentation

This directory contains architecture documentation for the Nowa-ZK Sequencer/Prover ZK
Validium.

## Documents

- **[Overview](./overview.md)** — Components (Sequencer, Prover, L1 contracts), their
  responsibilities, and the known gaps between design intent and current code.
- **[Data Flow](./data-flow.md)** — Order → match → batch → proof → L1 settlement, step
  by step, with code references.
- **[Storage](./storage.md)** — What each component persists, where, and why.

## Legacy / Archived

An earlier design ran execution on a Cosmos-chain Indexer instead of the off-chain
Sequencer. That subsystem still builds (`indexer/`) but isn't wired into the live
pipeline. Its docs are preserved for reference:
- [Architecture Evolution: Cosmos → Sequencer](../archived-files/architecture-evolution-cosmos-to-sequencer.md) — why the migration happened
- [Indexer Batch Flow (legacy)](../archived-files/indexer-batch-flow-legacy.md)
- [Indexer Cleanup System (legacy)](../archived-files/cleanup-system-legacy-indexer.md)

## Quick Links

- [Deployment Guide](../deployment/cloud.md)
- [Local Development Setup](../deployment/local.md)
- [Testing Guide](../testing.md)
- [Prover README](../../prover/README.md) · [Contracts README](../../contracts/README.md)
- Sequencer has no standalone README yet — see [Overview](./overview.md#sequencer-sequencer--the-execution-layer) for its responsibilities.
