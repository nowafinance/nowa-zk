// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {Test} from "forge-std/Test.sol";
import {StateManager} from "../src/StateManager.sol";
import {BatchRegistry} from "../src/BatchRegistry.sol";
import {MockVerifier} from "../src/mocks/MockVerifier.sol";

/**
 * @title BatchRegistryTest
 * @notice Comprehensive test suite for BatchRegistry with fuzzing
 */
contract BatchRegistryTest is Test {
    StateManager public stateManager;
    BatchRegistry public batchRegistry;
    MockVerifier public verifier;

    address public owner;
    address public sequencer;
    address public user;

    bytes32 public constant INITIAL_STATE_ROOT = bytes32(uint256(1));
    uint256 public constant FINALIZATION_DELAY = 1 hours;

    event BatchRegistered(
        uint256 indexed batchNumber,
        bytes32 indexed batchHash,
        bytes32 oldStateRoot,
        bytes32 newStateRoot,
        address indexed submitter,
        uint256 timestamp
    );

    event BatchFinalized(
        uint256 indexed batchNumber,
        bytes32 indexed newStateRoot
    );

    function setUp() public {
        owner = address(this);
        sequencer = makeAddr("sequencer");
        user = makeAddr("user");

        // Deploy contracts
        stateManager = new StateManager(INITIAL_STATE_ROOT);
        verifier = new MockVerifier();
        batchRegistry = new BatchRegistry(
            address(verifier),
            address(stateManager),
            sequencer,
            FINALIZATION_DELAY
        );

        // Transfer StateManager ownership to BatchRegistry
        stateManager.transferOwnership(address(batchRegistry));
    }

    // ============ Deployment Tests ============
    function test_Deployment() public view {
        assertEq(address(batchRegistry.STATE_MANAGER()), address(stateManager));
        assertEq(address(batchRegistry.verifier()), address(verifier));
        assertEq(batchRegistry.sequencer(), sequencer);
        assertEq(batchRegistry.owner(), owner);
        assertEq(batchRegistry.nextBatchNumber(), 1);
        assertFalse(batchRegistry.paused());
    }

    function testRevert_DeploymentWithZeroVerifier() public {
        vm.expectRevert("BatchRegistry: verifier cannot be zero address");
        new BatchRegistry(address(0), address(stateManager), sequencer, 0);
    }

    function testRevert_DeploymentWithZeroStateManager() public {
        vm.expectRevert("BatchRegistry: stateManager cannot be zero address");
        new BatchRegistry(address(verifier), address(0), sequencer, 0);
    }

    function testRevert_DeploymentWithZeroSequencer() public {
        vm.expectRevert("BatchRegistry: sequencer cannot be zero address");
        new BatchRegistry(
            address(verifier),
            address(stateManager),
            address(0),
            0
        );
    }

    // ============ Register Batch Tests ============

    function test_RegisterBatch() public {
        bytes memory batchData = abi.encode("test batch data");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        vm.expectEmit(true, true, true, true);
        emit BatchRegistered(
            1,
            batchHash,
            INITIAL_STATE_ROOT,
            newStateRoot,
            sequencer,
            block.timestamp
        );
        vm.expectEmit(true, true, false, false);
        emit BatchFinalized(1, newStateRoot);

        uint256 batchNumber = batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );

        assertEq(batchNumber, 1);

        BatchRegistry.Batch memory batch = batchRegistry.getBatch(1);
        assertEq(batch.batchHash, batchHash);
        assertEq(batch.oldStateRoot, INITIAL_STATE_ROOT);
        assertEq(batch.newStateRoot, newStateRoot);
        assertEq(batch.submitter, sequencer);
        assertEq(
            uint8(batch.status),
            uint8(BatchRegistry.BatchStatus.Finalized)
        ); // Should be Finalized immediately
        assertEq(batchRegistry.totalBatches(), 1);
        assertEq(stateManager.getCurrentStateRoot(), newStateRoot); // State root updated immediately
    }

    function testRevert_RegisterBatch_NotSequencer() public {
        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(user);
        vm.expectRevert("BatchRegistry: caller is not the sequencer");
        batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );
    }

    function testRevert_RegisterBatch_InvalidBatchHash() public {
        bytes memory batchData = abi.encode("test");
        bytes32 wrongHash = bytes32(uint256(999));
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        vm.expectRevert("BatchRegistry: batch hash mismatch");
        batchRegistry.registerBatch(
            wrongHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );
    }

    function testRevert_RegisterBatch_DuplicateHash() public {
        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );

        vm.prank(sequencer);
        vm.expectRevert("BatchRegistry: batch hash already exists");
        batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );
    }

    function testRevert_RegisterBatch_InvalidProof() public {
        verifier.setVerificationResult(false);

        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        vm.expectRevert("BatchRegistry: invalid proof");
        batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );
    }

    function testRevert_RegisterBatch_WhenPaused() public {
        batchRegistry.pause();

        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        vm.expectRevert("BatchRegistry: contract is paused");
        batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );
    }

    function test_SequentialFinalization() public {
        // Register batch 1
        bytes memory batchData1 = abi.encode("test", 1);
        bytes32 batchHash1 = keccak256(batchData1);
        bytes32 oldStateRoot1 = stateManager.getCurrentStateRoot();
        bytes32 newStateRoot1 = bytes32(
            uint256(keccak256(abi.encode("newStateRoot", 1)))
        );

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs1 = [
            uint256(0),
            uint256(oldStateRoot1),
            uint256(newStateRoot1),
            uint256(1)
        ];

        vm.prank(sequencer);
        batchRegistry.registerBatch(
            batchHash1,
            newStateRoot1,
            batchData1,
            a,
            b,
            c,
            publicInputs1
        );

        // Register batch 2 (chained to batch 1)
        bytes memory batchData2 = abi.encode("test", 2);
        bytes32 batchHash2 = keccak256(batchData2);
        bytes32 oldStateRoot2 = newStateRoot1; // Chain from batch 1
        bytes32 newStateRoot2 = bytes32(
            uint256(keccak256(abi.encode("newStateRoot", 2)))
        );

        uint256[4] memory publicInputs2 = [
            uint256(0),
            uint256(oldStateRoot2),
            uint256(newStateRoot2),
            uint256(2)
        ];

        vm.prank(sequencer);
        batchRegistry.registerBatch(
            batchHash2,
            newStateRoot2,
            batchData2,
            a,
            b,
            c,
            publicInputs2
        );

        assertEq(
            uint8(batchRegistry.getBatch(1).status),
            uint8(BatchRegistry.BatchStatus.Finalized)
        );
        assertEq(
            uint8(batchRegistry.getBatch(2).status),
            uint8(BatchRegistry.BatchStatus.Finalized)
        );
        assertEq(stateManager.getCurrentStateRoot(), newStateRoot2);
    }

    // ============ Fuzzing Tests ============

    function testFuzz_RegisterBatch(bytes32 seed, uint256 count) public {
        count = bound(count, 1, 5); // Reduced to avoid too many state transitions

        for (uint256 i = 0; i < count; i++) {
            bytes memory batchData = abi.encode(seed, i);
            bytes32 batchHash = keccak256(batchData);

            // Get actual current state root from StateManager
            bytes32 oldStateRoot = stateManager.getCurrentStateRoot();
            bytes32 newStateRoot = bytes32(
                uint256(keccak256(abi.encode(seed, i, block.timestamp)))
            );

            // Ensure unique state roots
            vm.assume(oldStateRoot != newStateRoot);
            vm.assume(newStateRoot != bytes32(0));

            uint256[2] memory a = [uint256(1), uint256(2)];
            uint256[2][2] memory b = [
                [uint256(3), uint256(4)],
                [uint256(5), uint256(6)]
            ];
            uint256[2] memory c = [uint256(7), uint256(8)];
            uint256[4] memory publicInputs = [
                uint256(0),
                uint256(oldStateRoot),
                uint256(newStateRoot),
                uint256(i + 1)
            ];

            vm.prank(sequencer);
            uint256 batchNumber = batchRegistry.registerBatch(
                batchHash,
                newStateRoot,
                batchData,
                a,
                b,
                c,
                publicInputs
            );
            assertEq(batchNumber, i + 1);

            // Should be finalized immediately
            assertEq(
                uint8(batchRegistry.getBatch(batchNumber).status),
                uint8(BatchRegistry.BatchStatus.Finalized)
            );
        }
    }

    // ============ Admin Function Tests ============

    function test_UpdateSequencer() public {
        address newSequencer = makeAddr("newSequencer");
        batchRegistry.updateSequencer(newSequencer);
        assertEq(batchRegistry.sequencer(), newSequencer);
    }

    function testRevert_UpdateSequencer_NotOwner() public {
        vm.prank(user);
        vm.expectRevert("BatchRegistry: caller is not the owner");
        batchRegistry.updateSequencer(user);
    }

    function test_PauseUnpause() public {
        batchRegistry.pause();
        assertTrue(batchRegistry.paused());

        batchRegistry.unpause();
        assertFalse(batchRegistry.paused());
    }

    function testRevert_Pause_NotOwner() public {
        vm.prank(user);
        vm.expectRevert("BatchRegistry: caller is not the owner");
        batchRegistry.pause();
    }

    function test_GetBatch() public {
        // Register batch
        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        uint256 batchNumber = batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );

        BatchRegistry.Batch memory batch = batchRegistry.getBatch(batchNumber);
        assertEq(batch.batchHash, batchHash);
        assertEq(batch.newStateRoot, newStateRoot);
        assertEq(batch.submitter, sequencer);
        assertEq(
            uint8(batch.status),
            uint8(BatchRegistry.BatchStatus.Finalized)
        );
    }

    function test_GetBatchNumber() public {
        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        uint256 batchNumber = batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );

        assertEq(batchRegistry.getBatchNumber(batchHash), batchNumber);
        assertEq(batchRegistry.getBatchNumber(bytes32(uint256(999))), 0); // Non-existent
    }

    function test_BatchExists() public {
        bytes memory batchData = abi.encode("test");
        bytes32 batchHash = keccak256(batchData);
        bytes32 newStateRoot = bytes32(uint256(2));

        assertFalse(batchRegistry.batchExists(batchHash));

        uint256[2] memory a = [uint256(1), uint256(2)];
        uint256[2][2] memory b = [
            [uint256(3), uint256(4)],
            [uint256(5), uint256(6)]
        ];
        uint256[2] memory c = [uint256(7), uint256(8)];
        uint256[4] memory publicInputs = [
            uint256(0),
            uint256(INITIAL_STATE_ROOT),
            uint256(newStateRoot),
            uint256(1)
        ];

        vm.prank(sequencer);
        batchRegistry.registerBatch(
            batchHash,
            newStateRoot,
            batchData,
            a,
            b,
            c,
            publicInputs
        );

        assertTrue(batchRegistry.batchExists(batchHash));
    }

    function test_GetCurrentStateRoot() public view {
        bytes32 currentRoot = batchRegistry.getCurrentStateRoot();
        assertEq(currentRoot, INITIAL_STATE_ROOT);
    }

    function test_UpdateVerifier() public {
        MockVerifier newVerifier = new MockVerifier();
        newVerifier.setVerificationResult(true);

        batchRegistry.updateVerifier(address(newVerifier));
        assertEq(address(batchRegistry.verifier()), address(newVerifier));
    }

    function testRevert_UpdateVerifier_NotOwner() public {
        MockVerifier newVerifier = new MockVerifier();
        vm.prank(user);
        vm.expectRevert("BatchRegistry: caller is not the owner");
        batchRegistry.updateVerifier(address(newVerifier));
    }

    function testRevert_UpdateVerifier_ZeroAddress() public {
        vm.expectRevert("BatchRegistry: verifier cannot be zero address");
        batchRegistry.updateVerifier(address(0));
    }

    function test_TransferOwnership() public {
        address newOwner = makeAddr("newOwner");
        batchRegistry.transferOwnership(newOwner);
        assertEq(batchRegistry.owner(), newOwner);
    }

    function testRevert_TransferOwnership_NotOwner() public {
        vm.prank(user);
        vm.expectRevert("BatchRegistry: caller is not the owner");
        batchRegistry.transferOwnership(user);
    }

    function testRevert_TransferOwnership_ZeroAddress() public {
        vm.expectRevert("BatchRegistry: new owner is the zero address");
        batchRegistry.transferOwnership(address(0));
    }
}
