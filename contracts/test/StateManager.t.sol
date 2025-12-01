// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {Test} from "forge-std/Test.sol";
import {StateManager} from "../src/StateManager.sol";

/**
 * @title StateManagerTest
 * @notice Comprehensive test suite for StateManager
 */
contract StateManagerTest is Test {
    StateManager public stateManager;

    address public owner;
    address public batchRegistry;
    address public user;

    bytes32 public constant INITIAL_STATE_ROOT = bytes32(uint256(1));

    event StateRootUpdated(bytes32 indexed stateRoot, uint256 indexed batchNumber, uint256 timestamp);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event Paused(address account);
    event Unpaused(address account);

    function setUp() public {
        owner = address(this);
        batchRegistry = makeAddr("batchRegistry");
        user = makeAddr("user");

        stateManager = new StateManager(INITIAL_STATE_ROOT);
    }

    // ============ Deployment Tests ============
    function test_Deployment() public view {
        assertEq(stateManager.owner(), owner);
        assertEq(stateManager.getCurrentStateRoot(), INITIAL_STATE_ROOT);
        assertEq(stateManager.totalStateRoots(), 1);
        assertEq(stateManager.lastFinalizedBatch(), 0);
        assertEq(stateManager.getStateRoot(0), INITIAL_STATE_ROOT);
        assertFalse(stateManager.paused());
    }

    function testRevert_DeploymentWithZeroStateRoot() public {
        vm.expectRevert("StateManager: initial state root cannot be zero");
        new StateManager(bytes32(0));
    }

    // ============ Update State Root Tests ============

    function test_UpdateStateRoot() public {
        bytes32 newStateRoot = bytes32(uint256(2));

        vm.expectEmit(true, true, false, true);
        emit StateRootUpdated(newStateRoot, 1, block.timestamp);

        stateManager.updateStateRoot(newStateRoot, 1);

        assertEq(stateManager.getCurrentStateRoot(), newStateRoot);
        assertEq(stateManager.getStateRoot(1), newStateRoot);
        assertEq(stateManager.getBatchNumber(newStateRoot), 1);
        assertEq(stateManager.totalStateRoots(), 2);
        assertEq(stateManager.lastFinalizedBatch(), 1);
    }

    function test_UpdateStateRoot_Sequential() public {
        for (uint256 i = 1; i <= 5; i++) {
            bytes32 newStateRoot = bytes32(uint256(i + 1));
            stateManager.updateStateRoot(newStateRoot, i);
            assertEq(stateManager.lastFinalizedBatch(), i);
        }
    }

    function testRevert_UpdateStateRoot_NotOwner() public {
        bytes32 newStateRoot = bytes32(uint256(2));

        vm.prank(user);
        vm.expectRevert("StateManager: caller is not the owner");
        stateManager.updateStateRoot(newStateRoot, 1);
    }

    function testRevert_UpdateStateRoot_ZeroStateRoot() public {
        vm.expectRevert("StateManager: state root cannot be zero");
        stateManager.updateStateRoot(bytes32(0), 1);
    }

    function testRevert_UpdateStateRoot_Unchanged() public {
        vm.expectRevert("StateManager: state root unchanged");
        stateManager.updateStateRoot(INITIAL_STATE_ROOT, 1);
    }

    function testRevert_UpdateStateRoot_NotSequential() public {
        bytes32 newStateRoot = bytes32(uint256(2));
        stateManager.updateStateRoot(newStateRoot, 1);

        // Try to update batch 3 (skipping batch 2)
        bytes32 newStateRoot2 = bytes32(uint256(3));
        vm.expectRevert("StateManager: batches must be sequential");
        stateManager.updateStateRoot(newStateRoot2, 3);
    }

    function testRevert_UpdateStateRoot_WhenPaused() public {
        stateManager.pause();

        bytes32 newStateRoot = bytes32(uint256(2));
        vm.expectRevert("StateManager: contract is paused");
        stateManager.updateStateRoot(newStateRoot, 1);
    }

    // ============ Validation Tests ============

    function test_ValidateStateRoot() public {
        bytes32 newStateRoot = bytes32(uint256(2));
        stateManager.updateStateRoot(newStateRoot, 1);

        assertTrue(stateManager.validateStateRoot(newStateRoot, 1));
        assertFalse(stateManager.validateStateRoot(newStateRoot, 2));
        assertFalse(stateManager.validateStateRoot(bytes32(uint256(999)), 1));
    }

    function test_StateRootExists() public {
        bytes32 newStateRoot = bytes32(uint256(2));

        assertFalse(stateManager.stateRootExists(newStateRoot));

        stateManager.updateStateRoot(newStateRoot, 1);

        assertTrue(stateManager.stateRootExists(newStateRoot));
        assertTrue(stateManager.stateRootExists(INITIAL_STATE_ROOT));
    }

    // ============ Pause/Unpause Tests ============

    function test_PauseUnpause() public {
        vm.expectEmit(false, false, false, true);
        emit Paused(owner);

        stateManager.pause();
        assertTrue(stateManager.paused());

        vm.expectEmit(false, false, false, true);
        emit Unpaused(owner);

        stateManager.unpause();
        assertFalse(stateManager.paused());
    }

    function testRevert_Pause_NotOwner() public {
        vm.prank(user);
        vm.expectRevert("StateManager: caller is not the owner");
        stateManager.pause();
    }

    function testRevert_Pause_AlreadyPaused() public {
        stateManager.pause();

        vm.expectRevert("StateManager: already paused");
        stateManager.pause();
    }

    function testRevert_Unpause_NotPaused() public {
        vm.expectRevert("StateManager: not paused");
        stateManager.unpause();
    }

    // ============ Ownership Tests ============

    function test_TransferOwnership() public {
        vm.expectEmit(true, true, false, false);
        emit OwnershipTransferred(owner, batchRegistry);

        stateManager.transferOwnership(batchRegistry);
        assertEq(stateManager.owner(), batchRegistry);
    }

    function testRevert_TransferOwnership_ZeroAddress() public {
        vm.expectRevert("StateManager: new owner is the zero address");
        stateManager.transferOwnership(address(0));
    }

    function testRevert_TransferOwnership_NotOwner() public {
        vm.prank(user);
        vm.expectRevert("StateManager: caller is not the owner");
        stateManager.transferOwnership(user);
    }

    // ============ Fuzzing Tests ============

    function testFuzz_UpdateStateRoot(bytes32 stateRoot, uint256 batchNumber) public {
        vm.assume(stateRoot != bytes32(0));
        vm.assume(stateRoot != INITIAL_STATE_ROOT);
        batchNumber = bound(batchNumber, 1, 1000);

        // Update sequentially
        for (uint256 i = 1; i <= batchNumber; i++) {
            bytes32 newStateRoot = bytes32(uint256(keccak256(abi.encode(stateRoot, i))));
            vm.assume(newStateRoot != bytes32(0));

            if (i == 1) {
                vm.assume(newStateRoot != INITIAL_STATE_ROOT);
            }

            stateManager.updateStateRoot(newStateRoot, i);
        }

        assertEq(stateManager.lastFinalizedBatch(), batchNumber);
    }

    function testFuzz_ValidateStateRoot(bytes32 stateRoot) public {
        vm.assume(stateRoot != bytes32(0));
        vm.assume(stateRoot != INITIAL_STATE_ROOT);

        assertFalse(stateManager.validateStateRoot(stateRoot, 1));

        stateManager.updateStateRoot(stateRoot, 1);

        assertTrue(stateManager.validateStateRoot(stateRoot, 1));
    }

    // ============ Integration Tests ============

    function test_FullWorkflow() public {
        // Simulate BatchRegistry workflow
        stateManager.transferOwnership(batchRegistry);

        // Batch Registry updates state roots
        for (uint256 i = 1; i <= 10; i++) {
            bytes32 newStateRoot = bytes32(uint256(i + 1));

            vm.prank(batchRegistry);
            stateManager.updateStateRoot(newStateRoot, i);

            assertEq(stateManager.getCurrentStateRoot(), newStateRoot);
            assertTrue(stateManager.validateStateRoot(newStateRoot, i));
        }

        assertEq(stateManager.totalStateRoots(), 11); // Initial + 10 updates
        assertEq(stateManager.lastFinalizedBatch(), 10);
    }

    // ============ View Function Tests ============

    function test_GetStateRoot() public {
        bytes32 newStateRoot = bytes32(uint256(2));
        stateManager.updateStateRoot(newStateRoot, 1);

        assertEq(stateManager.getStateRoot(1), newStateRoot);
        assertEq(stateManager.getStateRoot(0), INITIAL_STATE_ROOT);
        assertEq(stateManager.getStateRoot(999), bytes32(0)); // Non-existent
    }

    function test_GetBatchNumber() public {
        bytes32 newStateRoot = bytes32(uint256(2));
        stateManager.updateStateRoot(newStateRoot, 1);

        assertEq(stateManager.getBatchNumber(newStateRoot), 1);
        assertEq(stateManager.getBatchNumber(INITIAL_STATE_ROOT), 0);
        assertEq(stateManager.getBatchNumber(bytes32(uint256(999))), 0); // Non-existent
    }

    function test_GetLastFinalizedBatch() public {
        assertEq(stateManager.getLastFinalizedBatch(), 0);

        bytes32 newStateRoot1 = bytes32(uint256(2));
        stateManager.updateStateRoot(newStateRoot1, 1);
        assertEq(stateManager.getLastFinalizedBatch(), 1);

        bytes32 newStateRoot2 = bytes32(uint256(3));
        stateManager.updateStateRoot(newStateRoot2, 2);
        assertEq(stateManager.getLastFinalizedBatch(), 2);
    }
}
