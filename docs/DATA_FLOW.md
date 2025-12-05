# System Data Flow

This document outlines the data flow between the User, Sequencer, Prover, and the L1 Smart Contracts.

```mermaid
graph TD
    User[User] -->|Submit Tx (RPC/WS)| Sequencer
    subgraph Sequencer Node
        Sequencer -->|Batch Txs| Batching[Batching Logic]
        Batching -->|Save State| DB[(State DB)]
        Batching -->|Expose API| API[API Server]
    end
    
    subgraph Prover Node
        Prover -->|Poll/Fetch Batch| API
        Prover -->|Get State Root| L1[L1 Smart Contract]
        Prover -->|Generate Proof| ProofGen[Proof Generation]
        ProofGen -->|Submit Proof| L1
    end
    
    subgraph L1 Ethereum
        L1 -->|Verify| Verifier[Verifier Contract]
    end
```

## Description

1.  **User Submission**: Users submit transactions to the Sequencer via RPC or WebSocket.
2.  **Sequencer Processing**:
    *   The Sequencer batches transactions.
    *   Updates its local State DB.
    *   Exposes the batch data via an HTTP API.
3.  **Prover Actions**:
    *   Polls the Sequencer API for new batches.
    *   Fetches the current state root from the L1 Contract to ensure consistency.
    *   Generates a ZK Proof (Groth16) for the batch.
    *   Submits the proof and batch commitment to the L1 Contract.
4.  **L1 Verification**:
    *   The Rollup Contract receives the proof.
    *   Calls the Verifier Contract to validate the proof.
    *   If valid, the batch is registered and the state root is updated.
