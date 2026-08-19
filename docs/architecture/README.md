# Nowa-ZK Architecture Documentation

> [!IMPORTANT]
> This directory describes `main`'s Sequencer/Prover architecture — built groundwork,
> **on hold** pending prioritization, not the current operated flow. Nowa-ZK's current
> flow is [`v0.3.0`](../project/testnet-v0.3.0-flow.md).

This directory contains architecture documentation for `main`'s Sequencer/Prover
design.

## Documents

- **[Overview](./overview.md)** — Components (Sequencer, Prover, L1 contracts), their
  responsibilities, and the known gaps between design intent and current code.
- **[Data Flow](./data-flow.md)** — Order → match → batch → proof → L1 settlement, step
  by step, with code references.
- **[Storage](./storage.md)** — What each component persists, where, and why.

## Quick Links

- [Deployment Guide](../deployment/cloud.md)
- [Local Development Setup](../deployment/local.md)
- [Testing Guide](../testing.md)
- [Prover README](../../prover/README.md) · [Contracts README](../../contracts/README.md)
- Sequencer has no standalone README yet — see [Overview](./overview.md#sequencer-sequencer--the-execution-layer) for its responsibilities.
