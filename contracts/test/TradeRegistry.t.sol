// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/TradeRegistry.sol";
import "./mocks/MockTradeVerifier.sol";
import "../src/MiMC.sol";

contract TradeRegistryTest is Test {
    TradeRegistry public registry;
    MockTradeVerifier public mockVerifier;

    function setUp() public {
        mockVerifier = new MockTradeVerifier();
        registry = new TradeRegistry(address(mockVerifier));
    }

    function createValidInputs() internal pure returns (
        uint256[1] memory publicInputs,
        TradeRegistry.Trade[] memory trades
    ) {
        trades = new TradeRegistry.Trade[](25);
        // Fill arrays with dummy data
        for (uint i = 0; i < 25; i++) {
            trades[i] = TradeRegistry.Trade({
                messageHash: bytes32(uint256(i + 1)),
                pubKeyX: i + 100,
                pubKeyY: i + 200
            });
        }

        // Compute Expected Hash
        uint256[] memory hashData = new uint256[](150);
        for (uint i = 0; i < 25; i++) {
            uint256 hashInt = uint256(trades[i].messageHash);
            hashData[i*6] = hashInt & ((1 << 128) - 1);
            hashData[i*6 + 1] = hashInt >> 128;
            hashData[i*6 + 2] = trades[i].pubKeyX & ((1 << 128) - 1);
            hashData[i*6 + 3] = trades[i].pubKeyX >> 128;
            hashData[i*6 + 4] = trades[i].pubKeyY & ((1 << 128) - 1);
            hashData[i*6 + 5] = trades[i].pubKeyY >> 128;
        }
        
        publicInputs[0] = MiMC.hash(hashData);
    }

    function test_registerTrades_Success() public {
        uint256 batchNumber = 1;
        uint256 chunkIndex = 0;
        
        uint256[8] memory proof;
        uint256[2] memory commitments;
        uint256[2] memory commitmentPok;
        
        (uint256[1] memory publicInputs, TradeRegistry.Trade[] memory trades) = createValidInputs();
        
        // This should succeed as mockVerifier doesn't revert
        registry.registerTrades(
            batchNumber,
            chunkIndex,
            proof,
            commitments,
            commitmentPok,
            publicInputs,
            trades
        );

        // Verify state changes
        assertTrue(registry.isChunkVerified(batchNumber, chunkIndex));
        assertEq(registry.chunkBatchRoot(batchNumber, chunkIndex), bytes32(publicInputs[0]));
    }

    function test_registerTrades_RevertIfAlreadyVerified() public {
        uint256 batchNumber = 1;
        uint256 chunkIndex = 0;
        
        uint256[8] memory proof;
        uint256[2] memory commitments;
        uint256[2] memory commitmentPok;
        
        (uint256[1] memory publicInputs, TradeRegistry.Trade[] memory trades) = createValidInputs();
        
        // First call should succeed
        registry.registerTrades(
            batchNumber,
            chunkIndex,
            proof,
            commitments,
            commitmentPok,
            publicInputs,
            trades
        );

        // Second call should revert
        vm.expectRevert("Chunk already verified");
        registry.registerTrades(
            batchNumber,
            chunkIndex,
            proof,
            commitments,
            commitmentPok,
            publicInputs,
            trades
        );
    }

    function test_registerTrades_RevertIfInvalidProof() public {
        uint256 batchNumber = 1;
        uint256 chunkIndex = 0;
        
        uint256[8] memory proof;
        uint256[2] memory commitments;
        uint256[2] memory commitmentPok;
        
        (uint256[1] memory publicInputs, TradeRegistry.Trade[] memory trades) = createValidInputs();
        
        // Configure mock verifier to revert
        mockVerifier.setShouldRevert(true);

        // Expect the verifier to revert
        vm.expectRevert("MockTradeVerifier: proof invalid");
        registry.registerTrades(
            batchNumber,
            chunkIndex,
            proof,
            commitments,
            commitmentPok,
            publicInputs,
            trades
        );

        // Ensure state wasn't changed
        assertFalse(registry.isChunkVerified(batchNumber, chunkIndex));
    }
}
