// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface ITradeVerifier {
    function verifyProof(
        uint256[8] calldata proof,
        uint256[2] calldata commitments,
        uint256[2] calldata commitmentPok,
        uint256[120] calldata input
    ) external view;
}

contract TradeRegistry {
    ITradeVerifier public verifier;

    // Optional tracking of submitted proofs
    mapping(uint256 => mapping(uint256 => bool)) public isChunkVerified;

    event TradesVerified(uint256 indexed batchNumber, uint256 indexed chunkIndex);

    constructor(address _verifier) {
        verifier = ITradeVerifier(_verifier);
    }

    function registerTrades(
        uint256 batchNumber,
        uint256 chunkIndex,
        uint256[8] calldata proof,
        uint256[2] calldata commitments,
        uint256[2] calldata commitmentPok,
        uint256[120] calldata publicInputs
    ) external {
        require(!isChunkVerified[batchNumber][chunkIndex], "Chunk already verified");

        // The verifier reverts if the proof is invalid.
        verifier.verifyProof(proof, commitments, commitmentPok, publicInputs);

        isChunkVerified[batchNumber][chunkIndex] = true;
        emit TradesVerified(batchNumber, chunkIndex);
    }
}
