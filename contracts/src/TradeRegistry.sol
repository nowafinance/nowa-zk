// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface ITradeVerifier {
    function verifyProof(
        uint256[8] calldata proof,
        uint256[2] calldata commitments,
        uint256[2] calldata commitmentPok,
        uint256[301] calldata input
    ) external view;
}

contract TradeRegistry {
    ITradeVerifier public verifier;

    mapping(uint256 => mapping(uint256 => bool)) public isChunkVerified;
    mapping(uint256 => mapping(uint256 => bytes32)) public chunkBatchRoot;

    event TradesSettled(
        uint256 indexed batchNumber,
        uint256 indexed chunkIndex,
        bytes32 batchRoot,
        bytes32[25] messageHashes,
        address[25] signers
    );

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
        uint256[301] calldata publicInputs,
        bytes32[25] calldata messageHashes,
        address[25] calldata signers
    ) external {
        require(!isChunkVerified[batchNumber][chunkIndex], "Chunk already verified");

        // The verifier reverts if the proof is invalid.
        verifier.verifyProof(proof, commitments, commitmentPok, publicInputs);

        isChunkVerified[batchNumber][chunkIndex] = true;
        
        bytes32 batchRoot = bytes32(publicInputs[0]);
        chunkBatchRoot[batchNumber][chunkIndex] = batchRoot;

        emit TradesVerified(batchNumber, chunkIndex);
        emit TradesSettled(batchNumber, chunkIndex, batchRoot, messageHashes, signers);
    }
}
