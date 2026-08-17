# Project Update & Full Reset Guide

Procedures for when the ZK circuit changes — this almost always means new keys **and**
redeploying contracts.

## When Do You Need This?

- The circuit changed (`prover/circuits/state_circuit.go` — check the diff/release notes)
- You're seeing `invalid witness size` or `constraint #N is not satisfied` from the Prover
- Keys need regenerating for any other reason

> [!CAUTION]
> **Circuit changes require redeploying both `Verifier.sol` and `NowaRollup.sol`.**
> `Verifier.sol` is generated straight from the new verifying key — a fresh deploy of
> just `Verifier` doesn't help on its own, because `NowaRollup`'s constructor pins one
> specific `Verifier` address permanently (no setter). You need a new `NowaRollup`
> pointed at the new `Verifier`, which means the on-chain `stateRoot`/`batchCount`
> history resets too.

---

## Steps

```bash
# 1. Stop services
sudo systemctl stop nowa-sequencer nowa-prover   # or Ctrl+C if running manually

# 2. Get the new circuit code
cd ~/nowa-zk
git pull origin main

# 3. Regenerate keys AND Verifier.sol from the new circuit
make setup
# writes ~/.nowa-zk/keys/{state.ccs,state.pk,state.vk} (or rollup.* names)
# and contracts/src/generated/Verifier.sol

# 4. Rebuild everything
make build

# 5. Redeploy contracts (fresh Verifier + fresh NowaRollup)
set -a
source .env
set +a
make deploy
# → writes NEW addresses to contracts/deployments/deployments.json
#   AND copies them to ~/.nowa-zk/deployments.json automatically

cat ~/.nowa-zk/deployments.json   # confirm both addresses changed

# 6. Re-register any ERC20 tokens (registerToken is per-deployment, not carried over)
ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
cast send $ROLLUP "registerToken(address)" <TOKEN_ADDRESS> --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY

# 7. Reset the Prover's checkpoint (it's tied to the old contract's batch numbering)
make clean-data

# 8. Bootstrap the new contract's stateRoot — REQUIRED, not optional.
#    A fresh NowaRollup always starts at stateRoot = 0, but no real Sequencer tree
#    (even a genuinely empty one) ever roots to 0 — a depth-28 SMT's empty root is
#    the MiMC hash of 28 levels of zero-nodes, a different specific constant. Skip
#    this and every submitBatch() reverts with "Invalid old state root", spending
#    real gas each attempt.
#
#    If keeping existing Sequencer state (circuit change is balance/logic-compatible):
OLDROOT_DEC=$(curl -s http://localhost:8080/batch/1 | jq -r '.old_root')   # batch #1, not /batch/latest — the Prover always starts there on a fresh checkpoint
OLDROOT_HEX=$(python3 -c "print('0x' + format(int('$OLDROOT_DEC'), '064x'))")
cast send $ROLLUP "setStateRoot(bytes32)" $OLDROOT_HEX --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY
#    (owner-only, only callable while batchCount == 0 on the NEW contract)
#
#    If starting the Sequencer from scratch instead: `make clean-sequencer-state`
#    only deletes the on-disk DB — it has no effect while the Sequencer process is
#    still running (it'll keep serving its already-open, unchanged state). Actually
#    STOP the Sequencer first, run clean-sequencer-state, then restart it — and note
#    its first real batch still won't have old_root == 0, so you'll run the same
#    setStateRoot command above once it has at least one account/deposit in it.

# 9. Restart
sudo systemctl start nowa-sequencer nowa-prover
sudo journalctl -u nowa-prover -f
```

> [!WARNING]
> Redeploying loses all previous on-chain batch history on the new `NowaRollup`
> instance — the old contract still exists and still verifies its own old proofs, but
> nothing new gets submitted to it. There is currently no migration path for L2
> balances between the old and new deployments other than `setStateRoot()` if the
> circuit's account/leaf layout didn't change.

---

## Quick Reference

```bash
sudo systemctl stop nowa-sequencer nowa-prover
cd ~/nowa-zk && git pull origin main
make setup && make build
set -a && source .env && set +a && make deploy
make clean-data
sudo systemctl start nowa-sequencer nowa-prover
```

## Related
- [Upgrade Guide](./upgrade.md) — for code changes that *don't* touch the circuit
- [Troubleshooting](./troubleshooting.md) — for diagnosing which situation you're in
