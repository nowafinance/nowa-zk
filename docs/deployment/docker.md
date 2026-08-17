# Docker Setup Guide

> [!WARNING]
> **`docker-compose.yml` only defines `indexer` and `prover` services today** — there is
> no `sequencer` service in it. Since the live pipeline is Sequencer → Prover (the
> Indexer is optional/legacy, see [architecture/overview.md](../architecture/overview.md#indexer-indexer--legacy-optional)),
> this compose file doesn't yet describe a complete current deployment. Either run the
> Sequencer natively alongside `docker compose up` (§4 below has a snippet to add a
> `sequencer` service), or use [Local](./local.md)/[Cloud](./cloud.md) setup instead
> until that's added.

## Prerequisites

- Docker (includes `docker compose`)
- Go 1.24.10+ (for generating keys on the host)
- Foundry (for deploying contracts)

---

## 1. Generate Prover Keys (on the host)

Keys are generated on the host and mounted into the container — they aren't produced
inside Docker.

```bash
cd prover
go build -o ../build/prover-bin ./cmd/prover
../build/prover-bin setup --output-dir ../.nowa-zk/keys --contract-output ../contracts/src/generated
cd ..
```
(Matches what `docker-compose.yml` mounts: `./.nowa-zk:/app/.nowa-zk`, and the prover
service is started with `--keys-dir /app/.nowa-zk/keys`.)

---

## 2. Configure Environment

```bash
cp .env.example .env
nano .env
```
At minimum:
```bash
L1_RPC_URL=https://ethereum-sepolia-rpc.publicnode.com
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE
```

---

## 3. Deploy Contracts (on the host)

```bash
set -a && source .env && set +a
cd contracts && mkdir -p deployments
forge script script/Deploy.s.sol --rpc-url $L1_RPC_URL --broadcast
cd ..
mkdir -p .nowa-zk
cp contracts/deployments/deployments.json .nowa-zk/deployments.json
```

---

## 4. Start Docker Services

```bash
docker compose up -d
```
This builds and starts:
- **`indexer`** on `:8080` (legacy — optional; the current pipeline doesn't need it)
- **`prover`**, pointed at `http://indexer:8080` by default via `command:` in
  `docker-compose.yml` — **update this** to point at wherever your Sequencer is
  reachable from inside the container (e.g. `http://host.docker.internal:8080` if
  you're running the Sequencer natively on the host; the compose file already sets
  `extra_hosts: host.docker.internal:host-gateway` for exactly this).

There's no `sequencer` service to add by editing YAML alone — checked the `Dockerfile`:
its build stage only compiles `indexer` and `prover` (`go build -o /app/bin/indexer
./cmd/indexer` / `.../prover ./cmd/prover`) and copies just those two binaries into the
runtime image. Containerizing the Sequencer needs a `Dockerfile` build stage for
`sequencer/cmd/sequencer` first. Until that exists, run the Sequencer natively
(`make run-sequencer`, per [local.md](./local.md)) alongside the Dockerized Prover, and
point the Prover container at it via `host.docker.internal:8080` as described above.

---

## Verifying the System

```bash
docker compose ps
docker compose logs -f prover
```

If you're running the Sequencer natively:
```bash
curl http://localhost:8080/orderbook?token_id=1
curl http://localhost:8080/batch/latest
```

---

## Managing Services

```bash
docker compose down
docker compose restart
docker compose up -d --build   # rebuild after code changes
docker compose stats
```

---

## Troubleshooting

**"Connection refused" / RPC errors**
- Verify `L1_RPC_URL` is reachable from inside the container.
- Check your RPC provider's rate limits.

**"Contract address required"**
- Confirm `forge script` succeeded and `.nowa-zk/deployments.json` has a `NowaRollup`
  entry.

**"Keys not found" / "invalid witness size"**
- Confirm `.nowa-zk/keys/` has `state.ccs`/`state.pk`/`state.vk` (or the `rollup.*`
  names) and that they were generated from the **same** circuit version as what's
  running — see [operations/troubleshooting.md](../operations/troubleshooting.md).

**Reset everything**
```bash
docker compose down -v
rm -rf dir_data/
docker compose up -d
```
This only resets the containerized Indexer/Prover. If you're running the Sequencer
natively alongside (per the warning at the top), it also needs a fresh contract +
`stateRoot` bootstrap after a reset — see
[local.md §5 "Restart After Clearing All Data"](./local.md#5-restart-after-clearing-all-data)
for that part.

---

## Advanced Configuration

### Custom state persistence path
```yaml
volumes:
  - ./your-custom-path:/app/data
```

### Resource limits
```yaml
services:
  prover:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
```
