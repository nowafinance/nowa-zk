// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./MiMC.sol";

interface ITradeVerifier {
    function verifyProof(
        uint256[8] calldata proof,
        uint256[2] calldata commitments,
        uint256[2] calldata commitmentPok,
        uint256[1] calldata input
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
        uint256[25] pubKeyX,
        uint256[25] pubKeyY
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
        uint256[1] calldata publicInputs,
        bytes32[25] calldata messageHashes,
        uint256[25] calldata pubKeyX,
        uint256[25] calldata pubKeyY
    ) external {
        require(!isChunkVerified[batchNumber][chunkIndex], "Chunk already verified");

        // Compute the Expected Batch Root using MiMC on L1
        uint256[] memory hashData = new uint256[](150);
        for(uint i = 0; i < 25; i++) {
            uint256 hashInt = uint256(messageHashes[i]);
            
            // Hash: part 1 and 2
            hashData[i*6] = hashInt & ((1 << 128) - 1);
            hashData[i*6 + 1] = hashInt >> 128;
            
            // PubKeyX: part 1 and 2
            hashData[i*6 + 2] = pubKeyX[i] & ((1 << 128) - 1);
            hashData[i*6 + 3] = pubKeyX[i] >> 128;
            
            // PubKeyY: part 1 and 2
            hashData[i*6 + 4] = pubKeyY[i] & ((1 << 128) - 1);
            hashData[i*6 + 5] = pubKeyY[i] >> 128;
        }


        uint256 expectedBatchRoot = MiMC.hash(hashData);
        require(expectedBatchRoot == publicInputs[0], "Invalid BatchRoot");

        // The verifier reverts if the proof is invalid.
        verifier.verifyProof(proof, commitments, commitmentPok, publicInputs);

        isChunkVerified[batchNumber][chunkIndex] = true;
        
        bytes32 batchRoot = bytes32(publicInputs[0]);
        chunkBatchRoot[batchNumber][chunkIndex] = batchRoot;

        emit TradesVerified(batchNumber, chunkIndex);
        emit TradesSettled(batchNumber, chunkIndex, batchRoot, messageHashes, pubKeyX, pubKeyY);
    }
}

