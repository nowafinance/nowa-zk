# Nowa-ZK Documentation

Complete documentation for setting up, deploying, and operating the Nowa-ZK ZK Validium.

---

## 📖 Quick Navigation

### 🚀 Getting Started

- **[Root README](../README.md)** — quick-start with the essential `make` commands
- **[FAQ](../FAQ-ZK.md)** — architecture decisions (Validium vs Rollup, DA, public input hashing, Groth16 vs PLONK, etc.)

### 🌐 Deployment

- **[Local Development Setup](deployment/local.md)** — run the full stack (Sequencer + Prover + contracts) on your machine
- **[Cloud Server Setup](deployment/cloud.md)** — deploy on a Linux VPS with systemd services
- **[Docker Setup](deployment/docker.md)** — containerized Indexer + Prover (Sequencer service not yet in `docker-compose.yml` — see the doc for the gap)
- **[Cleanup / Full Reset](deployment/cloud-clear.md)** — tear down a cloud deployment

### ⚙️ Operations

- **[Upgrade Guide](operations/upgrade.md)** — regular code updates
- **[Project Update & Reset](operations/project-update.md)** — key regeneration and contract redeployment when the circuit changes
- **[Troubleshooting](operations/troubleshooting.md)** — common issues and solutions

### 📐 Architecture

- **[Overview](architecture/overview.md)** — components, responsibilities, known gaps
- **[Data Flow](architecture/data-flow.md)** — order → match → batch → proof → L1
- **[Storage](architecture/storage.md)** — what's persisted where

### 🧪 Testing

- **[Testing Guide](testing.md)** — unit tests per component, and how to drive a real end-to-end run

### 📊 Project

- **[Release Status](project/release-status.md)** — what's actually released vs. built-but-unreleased vs. genuinely incomplete
- **[Technical Roadmap](project/roadmap-technical.md)** — internal engineering phase tracking
- **[Marketing Roadmap](project/roadmap-marketing.md)** — public-facing phase/milestone status
- **[Frontend EdDSA Guide](project/frontend-eddsa-guide.md)** — the "hidden key" signing flow for wallet integrations

---

## 🎯 Common Tasks

### First Time Setup

1. Read the [root README](../README.md) for the `make setup` / `make deploy` quick start.
2. Follow [Local Development Setup](deployment/local.md) to run everything on one machine, or [Cloud Setup](deployment/cloud.md) for a server.
3. Bookmark the [Troubleshooting Guide](operations/troubleshooting.md).

### Regular Maintenance

- **Code update only**: [Upgrade Guide](operations/upgrade.md)
- **Circuit changed**: [Project Update Guide](operations/project-update.md) — this always requires new keys **and** redeploying `NowaRollup`/`Verifier` together.

### When Things Go Wrong

1. Check the [Troubleshooting Guide](operations/troubleshooting.md) first.
2. Review service logs (systemd: `sudo journalctl -u nowa-sequencer -f` / `-u nowa-prover -f`; local: whatever terminal you started them in).
3. Verify `.env` is actually loaded in the shell you're running from — `godotenv.Load()` (used by the Prover) only looks in the **current working directory**, not the repo root, so running `go run ./cmd/prover ...` from inside `prover/` needs its own env export. See Troubleshooting.

---

## 📁 Documentation Structure

```
docs/
├── README.md                        # This file
│
├── deployment/
│   ├── local.md                     # Local dev, all services on one machine
│   ├── cloud.md                     # Cloud server (systemd)
│   ├── docker.md                    # Docker Compose (Indexer + Prover only, today)
│   └── cloud-clear.md               # Full teardown/reset
│
├── operations/
│   ├── upgrade.md                   # Regular upgrades
│   ├── project-update.md            # Circuit changes & key/contract resets
│   └── troubleshooting.md           # Problem solving
│
├── architecture/
│   ├── README.md                    # Architecture doc index
│   ├── overview.md                  # Components & known gaps
│   ├── data-flow.md                 # Sequencer → Prover → L1
│   └── storage.md                   # Data retention per component
│
├── project/                         # Roadmap & project tracking
│
├── testing.md                       # Test commands per component + E2E
│
└── archived-files/                  # Retired docs, kept for historical reference
```

---

## ⚡ Critical Notes

### Circuit Changes Require Contract Redeployment

> [!CAUTION]
> When `prover/circuits/state_circuit.go` changes:
> - You MUST regenerate keys: `make setup` (writes to `~/.nowa-zk/keys/` and
>   regenerates `contracts/src/generated/Verifier.sol`)
> - You MUST redeploy **both** `Verifier.sol` and `NowaRollup.sol` — the verifying key
>   is baked into the deployed `Verifier` bytecode, and `NowaRollup`'s constructor
>   pins a specific `Verifier` address with no setter to change it later.
> - The old contract cannot verify proofs from new keys — you'll see
>   `invalid witness size` or `constraint #N is not satisfied` from the Prover if you
>   skip this.
>
> See: [Project Update Guide](operations/project-update.md)

### Environment Variables

`.env` lives at the **repo root**. Commands that `cd` into a component directory first
(e.g. running the Prover directly from `prover/`) won't find it automatically —
`godotenv.Load()` only checks the current working directory:

```bash
# From inside prover/, load the root .env explicitly:
export $(grep -v '^\s*#' ../.env | xargs)
```

Prefer the `make run-*` targets from the repo root — they load `.env` for you.

### Deployment Commands

```bash
# ✅ What `make deploy` actually runs (contracts/script/Deploy.s.sol, contract name Deploy)
cd contracts && forge script script/Deploy.s.sol --rpc-url $L1_RPC_URL --broadcast
```

### Correct Keys/Data Directories

| What | Correct path | Wrong / stale |
|---|---|---|
| Prover trusted-setup keys | `~/.nowa-zk/keys/` | `prover/keys/` (git-ignored, may hold stale local artifacts) |
| Sequencer LevelDB state | `~/.nowa-zk/sequencer/nowa_state_db` (via `make run-sequencer`) | `sequencer/nowa_state_db/` (accidentally tracked in git if you ran the binary manually from that directory) |
| Deployment addresses | `~/.nowa-zk/deployments.json` (Sequencer/Prover auto-load from here) | `contracts/deployments/deployments.json` is the source `forge` writes to — copy it, don't hand-edit the home-dir copy |

---

## 📝 Contributing to Docs

When updating documentation:
- Keep commands up-to-date with actual code — verify against the `Makefile` and the
  actual CLI flags/routes, not against what an older doc said.
- Include real error messages in troubleshooting entries.
- Test commands before documenting them.
- If a subsystem stops being on the live path (like the Indexer), move its doc to
  `archived-files/` with a note explaining what replaced it, rather than leaving it to
  rot in place looking current.
