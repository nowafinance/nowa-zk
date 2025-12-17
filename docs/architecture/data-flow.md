# System Data Flow

This document provides a comprehensive overview of how data flows through the Tan-ZK rollup system, from user transaction submission to final L1 verification.

---

## High-Level Architecture

```mermaid
graph TD
    User[👤 User/DApp] -->|1. Submit Tx| Sequencer
    
    subgraph Offchain["Off-Chain (L2)"]
        subgraph SequencerNode["Sequencer Node"]
            Sequencer[Sequencer Service] -->|2. Validate & Order| Mempool[Mempool]
            Mempool -->|3. Build Batch| BatchBuilder[Batch Builder]
            BatchBuilder -->|4. Update State| SMT[(Sparse Merkle Tree)]
            SMT -->|5. Compute Root| StateRoot[New State Root]
            BatchBuilder -->|6. Store| BadgerDB[(BadgerDB)]
            BatchBuilder -->|7. Expose| API[HTTP API :8080]
        end
        
        subgraph ProverNode["Prover Node"]
            ProverService[Prover Service] -->|8. Fetch Batch| API
            ProverService -->|9. Sync State| L1Contract
            ProverService -->|10. Build Witness| Witness[Circuit Witness]
            Witness -->|11. Generate| ZKProof[ZK Proof Generator]
            ZKProof -->|12. Groth16 Prove| Proof[π Proof]
            ProverService -->|13. Submit| L1Contract
            ProverService -->|14. Expose| ProverAPI[Status API :8081]
        end
    end
    
    subgraph Onchain["On-Chain (L1)"]
        L1Contract[BatchRegistry Contract] -->|15. Forward| Verifier[RollupVerifier Contract]
        Verifier -->|16. Verify π| Result{Valid?}
        Result -->|Yes| UpdateState[Update State Root]
        Result -->|No| Revert[Revert Transaction]
    end
    
    UpdateState -->|17. Emit Event| Event[BatchVerified Event]
```

---

## Detailed Data Flow Stages

### Stage 1: Transaction Submission

```mermaid
sequenceDiagram
    participant User as 👤 User/DApp
    participant RPC as Sequencer RPC
    participant WS as Sequencer WebSocket
    participant Seq as Sequencer Core
    
    User->>RPC: eth_sendTransaction / custom RPC
    RPC->>Seq: Validate signature & nonce
    Seq-->>RPC: Transaction hash
    RPC-->>User: tx_hash confirmation
    
    Note over User,Seq: Alternative: WebSocket for real-time
    User->>WS: Subscribe + Send TX
    WS->>Seq: Process TX
    Seq-->>WS: Confirmation
    WS-->>User: Real-time update
```

**What happens:**
1. User submits a transaction via JSON-RPC or WebSocket
2. Sequencer validates the signature and checks the nonce
3. Transaction is added to the mempool
4. User receives a transaction hash as confirmation

**Key data structures:**
```go
type Transaction struct {
    Hash     string      // Keccak256 hash of signed tx
    From     string      // Sender address (20 bytes)
    To       string      // Recipient address (20 bytes)  
    Value    *big.Int    // Amount in wei
    Nonce    uint64      // Sender's transaction count
    Input    []byte      // Calldata for contract calls
    GasLimit uint64      // Max gas units
    GasPrice uint64      // Price per gas unit
}
```

---

### Stage 2: Batch Building

```mermaid
sequenceDiagram
    participant Mempool as Mempool
    participant Builder as Batch Builder
    participant SMT as Sparse Merkle Tree
    participant Store as BadgerDB
    
    loop Every N seconds or M transactions
        Mempool->>Builder: Get pending txs
        Builder->>Builder: Sort by gas price (priority)
        
        loop For each transaction
            Builder->>SMT: Update sender balance
            Builder->>SMT: Update receiver balance
            Builder->>SMT: Store tx hash
            SMT-->>Builder: Updated root hash
        end
        
        Builder->>Builder: Compute batch hash
        Builder->>Builder: Generate execution traces
        Builder->>Store: Save batch to DB
        Builder-->>Mempool: Mark txs as processed
    end
```

**Batch creation process:**
1. **Transaction Collection**: Gather pending transactions from mempool
2. **Ordering**: Sort by gas price (highest first) for optimal fee extraction
3. **State Updates**: For each transaction:
   - Debit sender balance
   - Credit receiver balance
   - Record transaction in state
4. **Root Computation**: Calculate new Sparse Merkle Tree root
5. **Batch Finalization**: Create batch with metadata

**Batch data structure:**
```go
type Batch struct {
    Number       uint64         // Sequential batch number
    Hash         string         // Batch commitment hash
    OldStateRoot string         // State root before execution
    NewStateRoot string         // State root after execution
    Timestamp    int64          // Unix timestamp
    Transactions []*Transaction // Up to 128 transactions
    Traces       []*ExecutionTrace // Execution metadata
    Status       string         // "pending" | "proven" | "verified"
}
```

---

### Stage 3: Proof Generation

```mermaid
flowchart TB
    subgraph Input["Prover Inputs"]
        Batch[Batch Data]
        PrevRoot[Previous State Root]
        Keys[Proving Keys ~100MB]
    end
    
    subgraph Circuit["ZK Circuit Processing"]
        Witness[Build Witness] --> Hash[MiMC Hash Each Tx]
        Hash --> Merkle[Build Merkle Tree]
        Merkle --> Verify[Verify Root Match]
        Verify --> StateT[Compute State Transition]
        StateT --> Checks[Range Checks]
    end
    
    subgraph Output["Proof Output"]
        Proof["π = (A, B, C)"]
        Public[Public Inputs]
    end
    
    Batch --> Witness
    PrevRoot --> Witness
    Keys --> Groth16[Groth16 Prover]
    Checks --> Groth16
    Groth16 --> Proof
    Groth16 --> Public
```

**Circuit constraints verified:**
| Constraint | Description |
|------------|-------------|
| Batch Root | Merkle root of transactions matches public input |
| State Transition | NewStateRoot = Hash(OldStateRoot, tx_hashes...) |
| Nonce Check | Each tx nonce ≤ 1,000,000 |
| Gas Price Check | Each tx gas price ≤ 100 Gwei |
| Gas Limit Check | Each tx gas limit ≤ 30M |
| Sequencer Check | Sequencer address ≠ 0 |
| Timestamp Check | Timestamp ≤ 2,000,000,000 |
| Batch Number Check | Batch number ≤ 1,000,000 |

**Proof structure (Groth16):**
```
Proof π consists of:
├── A ∈ G₁ (BN254 curve point)
├── B ∈ G₂ (BN254 curve point in extension field)
└── C ∈ G₁ (BN254 curve point)

Public Inputs [6]:
├── BatchRoot     (Merkle root of transactions)
├── PrevStateRoot (State before batch)
├── NewStateRoot  (State after batch)
├── BatchNumber   (Sequential identifier)
├── Timestamp     (Block timestamp)
└── SequencerAddr (Authorized sequencer)
```

---

### Stage 4: On-Chain Verification

```mermaid
sequenceDiagram
    participant Prover as Prover Node
    participant Registry as BatchRegistry
    participant Verifier as RollupVerifier
    participant Chain as Blockchain State
    
    Prover->>Registry: registerBatch(batchHash, newStateRoot, batchData, proof, publicInputs)
    
    Registry->>Registry: Validate batch number = lastBatch + 1
    Registry->>Registry: Check caller is authorized sequencer
    Registry->>Registry: Verify batchHash = keccak256(batchData)
    Registry->>Registry: Verify publicInputs[1] = current stateRoot
    
    Registry->>Verifier: verifyProof(proof, publicInputs)
    
    Note over Verifier: Pairing check: e(A,B) = e(α,β) · e(L,γ) · e(C,δ)
    
    Verifier-->>Registry: true/false
    
    alt Proof Valid
        Registry->>Chain: stateRoot = newStateRoot
        Registry->>Chain: batches[batchNum] = BatchInfo{...}
        Registry->>Chain: totalBatches++
        Registry-->>Prover: Success + Event
    else Proof Invalid
        Registry-->>Prover: Revert("Invalid proof")
    end
```

**Contract functions:**
```solidity
// Main entry point
function registerBatch(
    bytes32 batchHash,
    bytes32 newStateRoot,
    bytes calldata batchData,
    uint256[2] calldata proofA,
    uint256[2][2] calldata proofB,
    uint256[2] calldata proofC,
    uint256[6] calldata publicInputs
) external onlySequencer

// State queries
function stateRoot() external view returns (bytes32)
function totalBatches() external view returns (uint256)
function getBatch(uint256 batchId) external view returns (BatchInfo)
```

---

## Data Storage Locations

```mermaid
graph LR
    subgraph Sequencer["Sequencer Storage"]
        SDB[(BadgerDB)]
        SDB --> Batches["batch:{N} → JSON"]
        SDB --> State["state:lastBlock"]
        SDB --> BlockHash["blockhash:{N}"]
    end
    
    subgraph Prover["Prover Storage"]
        PDB[(BoltDB)]
        PDB --> LastBatch["lastProcessedBatch"]
        PDB --> Proofs["proof:{N} → bytes"]
        PDB --> Status["status:{N}"]
        
        Keys[(File System)]
        Keys --> PK["rollup.pk (100MB)"]
        Keys --> VK["rollup.vk (524B)"]
        Keys --> R1CS["rollup.r1cs (48MB)"]
    end
    
    subgraph L1["On-Chain Storage"]
        Contract[(Smart Contract)]
        Contract --> SR["stateRoot: bytes32"]
        Contract --> TB["totalBatches: uint256"]
        Contract --> BM["batches: mapping"]
    end
```

---

## API Endpoints Summary

### Sequencer API (Port 8080)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/prover/batch/{n}` | GET | Get batch N with all transactions |
| `/prover/latest` | GET | Get the latest finalized batch |
| `/health` | GET | Service health check |
| `/metrics` | GET | Prometheus metrics |

**Example batch response:**
```json
{
  "number": 348,
  "hash": "0x7f3a91da2e7bc5b1...",
  "oldStateRoot": "0x123...",
  "newStateRoot": "0x456...",
  "timestamp": 1765460033,
  "transactions": [
    {
      "hash": "0xc3b997896815267b...",
      "from": "0x11c92eeea226a746...",
      "to": "0xd35544b9f8a9039c...",
      "value": 0,
      "nonce": 44419,
      "input": "0x79606d29..."
    }
  ]
}
```

### Prover API (Port 8081)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/status/{n}` | GET | Proof status for batch N |
| `/proof/{n}` | GET | Get proof bytes for batch N |
| `/batches/latest` | GET | Latest verified batch from L1 |
| `/batches/{n}` | GET | Batch details from L1 |

---

## Error Handling & Recovery

```mermaid
flowchart TD
    Start[Start Processing] --> Fetch{Fetch Batch}
    
    Fetch -->|Success| Prove[Generate Proof]
    Fetch -->|API Error| Retry1[Wait & Retry]
    Retry1 --> Fetch
    
    Prove -->|Success| Submit[Submit to L1]
    Prove -->|Constraint Error| Debug[Log Debug Info]
    Debug --> Skip[Skip Batch or Alert]
    
    Submit -->|Success| Save[Save State]
    Submit -->|Nonce Error| Resync[Resync Nonce]
    Submit -->|Gas Error| Bump[Bump Gas Price]
    Submit -->|Revert| Analyze[Analyze Failure]
    
    Resync --> Submit
    Bump --> Submit
    
    Save --> Next[Next Batch]
    Next --> Fetch
```

**Common error scenarios:**
| Error | Cause | Recovery |
|-------|-------|----------|
| Constraint not satisfied | Transaction data violates circuit rules | Check tx values, increase limits |
| State root mismatch | Local state out of sync with L1 | Re-sync from contract |
| Nonce too low | Transaction already submitted | Refresh nonce from chain |
| Insufficient gas | Gas estimation failed | Increase gas limit |
| Invalid proof | Keys don't match verifier | Regenerate keys & redeploy |

---

## Performance Characteristics

| Operation | Typical Duration | Notes |
|-----------|------------------|-------|
| Batch building | <100ms | Fast in-memory SMT updates |
| Key loading | 3-5 seconds | One-time at startup |
| Proof generation | 30-60 seconds | Depends on transaction count |
| Proof verification (local) | <100ms | Fast elliptic curve ops |
| L1 submission | 15-30 seconds | Depends on network congestion |
| L1 mining | 1-2 blocks | Chain-specific |

---

## Security Model

```mermaid
graph TB
    subgraph Guarantees["What ZK Proves"]
        G1[Transactions are valid]
        G2[State transition is correct]
        G3[Merkle root is authentic]
        G4[Sequencer is authorized]
    end
    
    subgraph Trust["Trust Assumptions"]
        T1[L1 chain is secure]
        T2[Trusted setup was honest]
        T3[Cryptographic primitives are sound]
    end
    
    subgraph NonGuarantees["What ZK Does NOT Prove"]
        N1[Transaction ordering fairness]
        N2[Censorship resistance]
        N3[Data availability]
    end
```

**Key security properties:**
- **Validity**: Only valid state transitions can produce a valid proof
- **Soundness**: It's computationally infeasible to forge a proof
- **Zero-Knowledge**: The proof reveals nothing about individual transactions
- **Succinctness**: Proof is constant size (~200 bytes) regardless of batch size
