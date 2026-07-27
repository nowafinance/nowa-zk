// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/TradeRegistry.sol";
import "./mocks/MockTradeVerifier.sol";

contract TradeRegistryTest is Test {
    TradeRegistry public registry;
    MockTradeVerifier public mockVerifier;

    function setUp() public {
        mockVerifier = new MockTradeVerifier();
        registry = new TradeRegistry(address(mockVerifier));
    }

    function test_registerTrades_Success() public {
        uint256 batchNumber = 1;
        uint256 chunkIndex = 0;
        
        uint256[8] memory proof;
        uint256[2] memory commitments;
        uint256[2] memory commitmentPok;
        
        uint256[121] memory publicInputs;
        publicInputs[0] = uint256(keccak256("batchRoot"));
        
        bytes32[10] memory messageHashes;
        address[10] memory signers;
        
        // This should succeed as mockVerifier doesn't revert
        registry.registerTrades(
            batchNumber,
            chunkIndex,
            proof,
            commitments,
            commitmentPok,
            publicInputs,
            messageHashes,
            signers
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
        uint256[121] memory publicInputs;
        bytes32[10] memory messageHashes;
        address[10] memory signers;
        
        // First call should succeed
        registry.registerTrades(
            batchNumber,
            chunkIndex,
            proof,
            commitments,
            commitmentPok,
            publicInputs,
            messageHashes,
            signers
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
            messageHashes,
            signers
        );
    }

    function test_registerTrades_RevertIfInvalidProof() public {
        uint256 batchNumber = 1;
        uint256 chunkIndex = 0;
        
        uint256[8] memory proof;
        uint256[2] memory commitments;
        uint256[2] memory commitmentPok;
        uint256[121] memory publicInputs;
        bytes32[10] memory messageHashes;
        address[10] memory signers;
        
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
            messageHashes,
            signers
        );

        // Ensure state wasn't changed
        assertFalse(registry.isChunkVerified(batchNumber, chunkIndex));
    }
}
