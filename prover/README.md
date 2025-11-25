# **ZK-Sequencer — Prover Module**

⚠️ *Under Production*

This directory contains the **ZK Prover** used by the `Tan-ZK` Sequencer.
It is a standalone Go module (`go 1.21+`) using **Gnark** for circuit definition, compilation, and proof generation.

The prover is responsible for:

* Defining ZK circuits
* Building witnesses
* Running (future) proving pipelines for batched state transitions
* Exposing a lightweight CLI interface


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

## **Usage**

The prover CLI has two main commands: `setup` and `start`.

### **Setup**

The `setup` command generates the proving and verifying keys for the circuit.

```bash
./bin/prover setup --output-dir <path_to_keys_dir>
```

By default, the keys are saved in the `./keys` directory.

### **Start**

The `start` command starts the prover, which generates a proof.

```bash
./bin/prover start --keys-dir <path_to_keys_dir>
```

By default, the prover looks for the keys in the `./keys` directory.

## **Development Notes**

* This module follows the `Tan-ZK` architecture where the **sequencer** delegates batch proof generation to this prover.
* Future commits will add real circuits, state transition proving, and proof serialization.