# Prover-RS (Rust ZK Prover)

> **Status**: 🚧 Planned for 2027

This directory is reserved for the future Rust implementation of the ZK prover for the ZK-Indexer project.

## Overview

The ZK-Indexer currently uses a **Go-based prover** built with [Gnark](https://github.com/Consensys/gnark) (see [`../prover/`](../prover/)). This Rust implementation will serve as an alternative, high-performance prover implementation planned for development in 2027.

## Motivation

### Why Rust?

- **Performance**: Rust's zero-cost abstractions and memory safety enable high-performance cryptographic operations
- **Ecosystem**: Growing ZK libraries in Rust (arkworks, bellman, etc.)
- **Safety**: Memory safety without garbage collection overhead
- **Concurrency**: Excellent support for parallel proof generation
- **Production Ready**: Widely used in blockchain infrastructure

### Goals

- **Performance Parity**: Match or exceed Go prover performance
- **API Compatibility**: Maintain compatibility with existing indexer interface
- **Modularity**: Allow switching between Go and Rust provers
- **Optimization**: Leverage Rust's performance characteristics for proof generation

## Timeline

- **2025**: Go prover (Gnark) - Current implementation
- **2027**: Rust prover - Planned implementation
- **Future**: Potential for both implementations to coexist

## Planned Technology Stack

### Potential Libraries

- **[arkworks](https://github.com/arkworks-rs)**: Suite of libraries for building and working with zero-knowledge proof systems
- **[bellman](https://github.com/zcash/bellman)**: ZK-SNARK library
- **[halo2](https://github.com/zcash/halo2)**: ZK-SNARK implementation
- **[plonky2](https://github.com/mir-protocol/plonky2)**: Fast ZK-SNARK implementation

### Architecture Considerations

- **Circuit Compatibility**: Ensure circuits can be shared or translated between Go and Rust
- **Witness Generation**: Efficient witness data processing
- **Proof Generation**: Optimized proof computation
- **Integration**: Seamless integration with existing indexer service

## Current Status

🚧 **Not Started** - This is a placeholder for future development.

The Go prover in [`../prover/`](../prover/) is the current active implementation.

## Future Development

When development begins in 2027, this directory will contain:

- Rust project structure (Cargo.toml, src/, etc.)
- Circuit implementations
- Prover service
- Integration tests
- Performance benchmarks
- Documentation

## Related Documentation

- **Go Prover**: See [`../prover/README.md`](../prover/README.md)
- **Indexer**: See [`../indexer/README.md`](../indexer/README.md)
- **Project Roadmap**: See [`../ROADMAP.md`](../ROADMAP.md)
- **Milestones**: See [`../docs/milestone.md`](../docs/milestone.md)

## Contributing

This project is not yet accepting contributions for the Rust prover. Once development begins in 2027, please refer to the main [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Questions?

For questions about the Rust prover plans or timeline, please:
- Check the [ROADMAP.md](../ROADMAP.md) for project timeline
- Open a GitHub Discussion for general questions
- Open a GitHub Issue for specific feature requests

---

**Last Updated**: November 2025  
**Planned Start**: 2027

