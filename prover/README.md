# **ZK-Sequencer — Prover Module**

⚠️ *Under Production*

This directory contains the **ZK Prover** used by the `Tan-ZK` Sequencer.
It is a standalone Go module (`go 1.21+`) using **Gnark** for circuit definition, compilation, and proof generation.

The prover is responsible for:

* Defining ZK circuits
* Building witnesses
* Running (future) proving pipelines for batched state transitions
* Exposing a lightweight CLI interface

---

## **Structure**

```
prover/
├─ go.mod
├─ Makefile
├─ cmd/
│  └─ prover/         # CLI entrypoint
├─ prover/
│  ├─ prover.go       # core prover API
│  └─ prover_test.go  # gnark compile test
├─ circuits/
│  └─ simple_circuit.go   # example circuit
└─ internal/
   └─ witness/
      └─ witness.go       # witness helpers
```

---

## **Install Dependencies**

```bash
make deps
```

This fetches Gnark and updates `go.mod`.

---

## **Run Tests**

Compiles prover logic + gnark dependencies:

```bash
make test
```

or directly:

```bash
go test ./prover/...
```

---

## **Build Prover CLI**

```bash
make build
```

Binary output:

```
bin/prover
```

---

## **Run Proof (placeholder)**

```bash
make prove
```

or manually:

```bash
./bin/prover --circuit circuits/simple
```

## **Development Notes**

* This module follows the `Tan-ZK` architecture where the **sequencer** delegates batch proof generation to this prover.
* Future commits will add real circuits, state transition proving, and proof serialization.