// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/**
 * @title StateManager
 * @notice Manages state root storage, updates, and validation for the zk-sequencer system
 * @dev This contract handles the storage and retrieval of state roots, which represent
 *      the Merkle root of the entire system state at a given point in time. State roots
 *      are updated when batches are processed and verified.
 *
 * @custom:security This contract is part of the core security infrastructure.
 *                  All state root updates must be validated and authorized.
 */
contract StateManager {
    /// @notice Emitted when a new state root is set
    /// @param stateRoot The new state root
    /// @param batchNumber The batch number associated with this state root
    /// @param timestamp The timestamp when the state root was set
    event StateRootUpdated(bytes32 indexed stateRoot, uint256 indexed batchNumber, uint256 timestamp);

    /// @notice Emitted when a state root is validated
    /// @param stateRoot The validated state root
    /// @param batchNumber The batch number associated with this state root
    event StateRootValidated(bytes32 indexed stateRoot, uint256 indexed batchNumber);

    /// @notice Current state root
    bytes32 public currentStateRoot;

    /// @notice Mapping from batch number to state root
    mapping(uint256 => bytes32) public stateRoots;

    /// @notice Mapping from state root to batch number
    mapping(bytes32 => uint256) public stateRootToBatch;

    /// @notice Total number of state roots stored
    uint256 public totalStateRoots;

    /// @notice Owner address (can be updated to a multisig in the future)
    address public owner;

    /// @notice Modifier to restrict access to owner
    modifier onlyOwner() {
        _onlyOwner();
        _;
    }

    /// @notice Internal function to check ownership
    function _onlyOwner() internal view {
        require(msg.sender == owner, "StateManager: caller is not the owner");
    }

    /// @notice Constructor sets the initial state root and owner
    /// @param _initialStateRoot The initial state root (typically zero hash or genesis state)
    constructor(bytes32 _initialStateRoot) {
        owner = msg.sender;
        currentStateRoot = _initialStateRoot;
        stateRoots[0] = _initialStateRoot;
        stateRootToBatch[_initialStateRoot] = 0;
        totalStateRoots = 1;

        emit StateRootUpdated(_initialStateRoot, 0, block.timestamp);
    }

    /**
     * @notice Updates the state root for a new batch
     * @dev This function should only be called by authorized contracts (e.g., BatchRegistry)
     *      after a batch has been verified. The new state root must be different from the current one.
     *
     * @param newStateRoot The new state root after batch execution
     * @param batchNumber The batch number associated with this state root
     *
     * @custom:security Only callable by owner (will be updated to BatchRegistry in future)
     */
    function updateStateRoot(bytes32 newStateRoot, uint256 batchNumber) external onlyOwner {
        require(newStateRoot != bytes32(0), "StateManager: state root cannot be zero");
        require(newStateRoot != currentStateRoot, "StateManager: state root unchanged");
        require(
            stateRootToBatch[newStateRoot] == 0 || stateRootToBatch[newStateRoot] == batchNumber,
            "StateManager: state root already exists for different batch"
        );

        // If this state root was already set for this batch, skip
        if (stateRoots[batchNumber] == bytes32(0)) {
            totalStateRoots++;
        }

        currentStateRoot = newStateRoot;
        stateRoots[batchNumber] = newStateRoot;
        stateRootToBatch[newStateRoot] = batchNumber;

        emit StateRootUpdated(newStateRoot, batchNumber, block.timestamp);
    }

    /**
     * @notice Validates that a state root exists and is valid
     * @dev Checks if the state root has been recorded and matches the expected batch number
     *
     * @param stateRoot The state root to validate
     * @param batchNumber The expected batch number for this state root
     * @return isValid True if the state root is valid and matches the batch number
     */
    function validateStateRoot(bytes32 stateRoot, uint256 batchNumber) external view returns (bool isValid) {
        isValid = stateRoots[batchNumber] == stateRoot && stateRootToBatch[stateRoot] == batchNumber;

        if (isValid) {
            // Note: We can't emit events in view functions, but this is for validation
            // The caller can emit an event if needed
        }
    }

    /**
     * @notice Gets the state root for a specific batch number
     * @param batchNumber The batch number to query
     * @return stateRoot The state root for the given batch number, or zero if not set
     */
    function getStateRoot(uint256 batchNumber) external view returns (bytes32 stateRoot) {
        return stateRoots[batchNumber];
    }

    /**
     * @notice Gets the batch number for a specific state root
     * @param stateRoot The state root to query
     * @return batchNumber The batch number associated with this state root, or zero if not found
     */
    function getBatchNumber(bytes32 stateRoot) external view returns (uint256 batchNumber) {
        return stateRootToBatch[stateRoot];
    }

    /**
     * @notice Gets the current state root
     * @return stateRoot The current state root
     */
    function getCurrentStateRoot() external view returns (bytes32 stateRoot) {
        return currentStateRoot;
    }

    /**
     * @notice Checks if a state root exists
     * @param stateRoot The state root to check
     * @return exists True if the state root has been recorded
     */
    function stateRootExists(bytes32 stateRoot) external view returns (bool exists) {
        return stateRootToBatch[stateRoot] != 0 || stateRoot == currentStateRoot;
    }

    /**
     * @notice Transfers ownership to a new address
     * @dev This allows transferring ownership to a multisig or BatchRegistry contract
     * @param newOwner The address of the new owner
     */
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "StateManager: new owner is the zero address");
        owner = newOwner;
    }

    /**
     * @notice Renounces ownership (sets owner to zero address)
     * @dev This makes the contract ownerless. Use with extreme caution.
     */
    function renounceOwnership() external onlyOwner {
        owner = address(0);
    }
}

