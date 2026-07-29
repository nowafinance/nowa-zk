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

    struct Trade {
        bytes32 messageHash;
        uint256 pubKeyX;
        uint256 pubKeyY;
    }

    mapping(uint256 => mapping(uint256 => bool)) public isChunkVerified;
    mapping(uint256 => mapping(uint256 => bytes32)) public chunkBatchRoot;

    event TradesSettled(
        uint256 indexed batchNumber,
        uint256 indexed chunkIndex,
        bytes32 batchRoot,
        Trade[] trades
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
        Trade[] calldata trades
    ) external {
        require(!isChunkVerified[batchNumber][chunkIndex], "Chunk already verified");
        require(trades.length == 25, "Invalid trades length");

        // Compute the Expected Batch Root using MiMC on L1
        uint256[] memory hashData = new uint256[](150);
        for(uint i = 0; i < 25; i++) {
            uint256 hashInt = uint256(trades[i].messageHash);
            
            // Hash: part 1 and 2
            hashData[i*6] = hashInt & ((1 << 128) - 1);
            hashData[i*6 + 1] = hashInt >> 128;
            
            // PubKeyX: part 1 and 2
            hashData[i*6 + 2] = trades[i].pubKeyX & ((1 << 128) - 1);
            hashData[i*6 + 3] = trades[i].pubKeyX >> 128;
            
            // PubKeyY: part 1 and 2
            hashData[i*6 + 4] = trades[i].pubKeyY & ((1 << 128) - 1);
            hashData[i*6 + 5] = trades[i].pubKeyY >> 128;
        }


        uint256 expectedBatchRoot = MiMC.hash(hashData);
        require(expectedBatchRoot == publicInputs[0], "Invalid BatchRoot");

        // The verifier reverts if the proof is invalid.
        verifier.verifyProof(proof, commitments, commitmentPok, publicInputs);

        isChunkVerified[batchNumber][chunkIndex] = true;
        
        bytes32 batchRoot = bytes32(publicInputs[0]);
        chunkBatchRoot[batchNumber][chunkIndex] = batchRoot;

        emit TradesVerified(batchNumber, chunkIndex);
        emit TradesSettled(batchNumber, chunkIndex, batchRoot, trades);
    }
}

