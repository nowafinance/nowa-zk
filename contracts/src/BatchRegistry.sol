// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {IVerifier} from "./interfaces/IVerifier.sol";
import {StateManager} from "./StateManager.sol";

/**
 * @title BatchRegistry
 * @notice Core contract for batch registration and verification in the zk-sequencer system
 * @dev This contract registers batches of transactions, verifies ZK proofs using IVerifier,
 *      and manages batch state through StateManager. It implements a two-phase commit:
 *      1. Register & Verify (immediate)
 *      2. Finalize (after challenge period)
 *
 * @custom:security This contract is part of the core security infrastructure.
 *                  All batch registrations must include valid ZK proofs.
 */
contract BatchRegistry {
    /// @notice Batch status enumeration
    enum BatchStatus {
        NonExistent,
        Verified, // Proof verified, awaiting finalization
        Finalized // Finalized and state root updated
    }

    /// @notice Batch metadata structure
    struct Batch {
        bytes32 batchHash;
        bytes32 oldStateRoot;
        bytes32 newStateRoot;
        address submitter;
        uint256 timestamp;
        uint256 verifiedAt;
        BatchStatus status;
    }

    /// @notice Emitted when a new batch is registered and verified
    event BatchRegistered(
        uint256 indexed batchNumber,
        bytes32 indexed batchHash,
        bytes32 oldStateRoot,
        bytes32 newStateRoot,
        address indexed submitter,
        uint256 timestamp
    );

    /// @notice Emitted when a batch is finalized
    event BatchFinalized(uint256 indexed batchNumber, bytes32 indexed newStateRoot);

    /// @notice Emitted when sequencer is updated
    event SequencerUpdated(address indexed oldSequencer, address indexed newSequencer);

    /// @notice Emitted when ownership is transferred
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /// @notice Emitted when the verifier is updated
    event VerifierUpdated(address indexed oldVerifier, address indexed newVerifier);

    /// @notice Emitted when the contract is paused
    event Paused(address account);

    /// @notice Emitted when the contract is unpaused
    event Unpaused(address account);

    /// @notice Mapping from batch number to Batch struct
    mapping(uint256 => Batch) public batches;

    /// @notice Mapping from batch hash to batch number
    mapping(bytes32 => uint256) public batchHashToNumber;

    /// @notice Total number of batches registered
    uint256 public totalBatches;

    /// @notice Next batch number to be assigned
    uint256 public nextBatchNumber;

    /// @notice Verifier contract interface
    IVerifier public verifier;

    /// @notice StateManager contract
    StateManager public immutable STATE_MANAGER;

    /// @notice Owner address (can be updated to a multisig in the future)
    address public owner;

    /// @notice Authorized sequencer address
    address public sequencer;

    /// @notice Paused flag to stop batch registration in emergencies
    bool public paused;

    /// @notice Modifier to restrict access to owner
    modifier onlyOwner() {
        _onlyOwner();
        _;
    }

    /// @notice Modifier to restrict access to sequencer
    modifier onlySequencer() {
        _onlySequencer();
        _;
    }

    /// @notice Modifier to check if contract is not paused
    modifier whenNotPaused() {
        _whenNotPaused();
        _;
    }

    /// @notice Internal function to check ownership
    function _onlyOwner() internal view {
        require(msg.sender == owner, "BatchRegistry: caller is not the owner");
    }

    /// @notice Internal function to check sequencer
    function _onlySequencer() internal view {
        require(msg.sender == sequencer, "BatchRegistry: caller is not the sequencer");
    }

    /// @notice Internal function to check if contract is not paused
    function _whenNotPaused() internal view {
        require(!paused, "BatchRegistry: contract is paused");
    }

    /**
     * @notice Constructor initializes the BatchRegistry
     * @param _verifier Address of the IVerifier contract
     * @param _stateManager Address of the StateManager contract
     * @param _sequencer Address of the authorized sequencer
     */
    constructor(address _verifier, address _stateManager, address _sequencer) {
        require(_verifier != address(0), "BatchRegistry: verifier cannot be zero address");
        require(_stateManager != address(0), "BatchRegistry: stateManager cannot be zero address");
        require(_sequencer != address(0), "BatchRegistry: sequencer cannot be zero address");

        owner = msg.sender;
        verifier = IVerifier(_verifier);
        STATE_MANAGER = StateManager(_stateManager);
        sequencer = _sequencer;
        nextBatchNumber = 1;
    }

    /**
     * @notice Registers a new batch with proof verification
     * @dev This function verifies the ZK proof and stores batch metadata.
     *      Since this is a ZK-Rollup, verification implies finality.
     *      The batch is marked as Finalized immediately and state root is updated.
     *
     * @param batchHash The hash of the batch being registered
     * @param newStateRoot The state root after batch execution
     * @param batchData The raw batch data (stored in calldata for data availability)
     * @param proofA The first part of the ZK proof (G1 point)
     * @param proofB The second part of the ZK proof (G2 point)
     * @param proofC The third part of the ZK proof (G1 point)
     * @param publicInputs The public inputs [BatchRoot, PrevStateRoot, NewStateRoot, BatchNumber, Timestamp, SequencerAddr]
     *
     * @return batchNumber The assigned batch number
     */
    function registerBatch(
        bytes32 batchHash,
        bytes32 newStateRoot,
        bytes calldata batchData,
        uint256[2] calldata proofA,
        uint256[2][2] calldata proofB,
        uint256[2] calldata proofC,
        uint256[6] calldata publicInputs
    ) external onlySequencer whenNotPaused returns (uint256 batchNumber) {
        // Input validation
        require(batchHash != bytes32(0), "BatchRegistry: batch hash cannot be zero");
        require(newStateRoot != bytes32(0), "BatchRegistry: new state root cannot be zero");
        require(batchData.length > 0, "BatchRegistry: batch data cannot be empty");
        require(batchHashToNumber[batchHash] == 0, "BatchRegistry: batch hash already exists");

        // Verify batch hash matches data (data availability check)
        require(keccak256(batchData) == batchHash, "BatchRegistry: batch hash mismatch");

        // Get expected old state root (chain from last batch or use finalized root)
        bytes32 expectedOldStateRoot;
        if (totalBatches > 0) {
            expectedOldStateRoot = batches[totalBatches].newStateRoot;
        } else {
            expectedOldStateRoot = STATE_MANAGER.getCurrentStateRoot();
        }

        // Verify state transition
        require(
            bytes32(publicInputs[1]) == expectedOldStateRoot,
            "BatchRegistry: publicInputs[1] must match expected old state root"
        );
        require(bytes32(publicInputs[1]) != newStateRoot, "BatchRegistry: state root unchanged");

        // Assign batch number
        batchNumber = nextBatchNumber;

        // Validate public inputs
        // publicInputs[0] = BatchRoot (Merkle root of transactions)
        // publicInputs[1] = PrevStateRoot
        // publicInputs[2] = NewStateRoot
        // publicInputs[3] = BatchNumber
        // publicInputs[4] = Timestamp
        // publicInputs[5] = SequencerAddr

        require(bytes32(publicInputs[2]) == newStateRoot, "BatchRegistry: publicInputs[2] must match newStateRoot");
        require(publicInputs[3] == batchNumber, "BatchRegistry: publicInputs[3] must match batchNumber");
        require(publicInputs[4] <= block.timestamp + 300, "BatchRegistry: timestamp cannot be too far in future");
        require(publicInputs[5] == uint256(uint160(msg.sender)), "BatchRegistry: sequencer address mismatch");

        // Verify the ZK proof
        bool proofValid = verifier.verifyProof(proofA, proofB, proofC, publicInputs);
        require(proofValid, "BatchRegistry: invalid proof");

        // Store batch metadata (Finalized immediately)
        batches[batchNumber] = Batch({
            batchHash: batchHash,
            oldStateRoot: expectedOldStateRoot,
            newStateRoot: newStateRoot,
            submitter: msg.sender,
            timestamp: block.timestamp,
            verifiedAt: block.number,
            status: BatchStatus.Finalized
        });

        batchHashToNumber[batchHash] = batchNumber;
        totalBatches++;
        nextBatchNumber++;

        // Update StateManager immediately
        STATE_MANAGER.updateStateRoot(newStateRoot, batchNumber);

        emit BatchRegistered(batchNumber, batchHash, expectedOldStateRoot, newStateRoot, msg.sender, block.timestamp);
        emit BatchFinalized(batchNumber, newStateRoot);

        return batchNumber;
    }

    /**
     * @notice Gets batch information by batch number
     * @param batchNumber The batch number to query
     * @return batch The Batch struct for the given batch number
     */
    function getBatch(uint256 batchNumber) external view returns (Batch memory batch) {
        require(batchNumber > 0 && batchNumber < nextBatchNumber, "BatchRegistry: invalid batch number");
        return batches[batchNumber];
    }

    /**
     * @notice Gets batch number by batch hash
     * @param batchHash The batch hash to query
     * @return batchNumber The batch number associated with the hash, or zero if not found
     */
    function getBatchNumber(bytes32 batchHash) external view returns (uint256 batchNumber) {
        return batchHashToNumber[batchHash];
    }

    /**
     * @notice Checks if a batch hash exists
     * @param batchHash The batch hash to check
     * @return exists True if the batch hash has been registered
     */
    function batchExists(bytes32 batchHash) external view returns (bool exists) {
        return batchHashToNumber[batchHash] != 0;
    }

    /**
     * @notice Gets the current state root from StateManager
     * @return stateRoot The current state root
     */
    function getCurrentStateRoot() external view returns (bytes32 stateRoot) {
        return STATE_MANAGER.getCurrentStateRoot();
    }

    // ========== Admin Functions ==========

    /**
     * @notice Updates the sequencer address
     * @dev Only callable by owner
     * @param newSequencer The address of the new sequencer
     */
    function updateSequencer(address newSequencer) external onlyOwner {
        require(newSequencer != address(0), "BatchRegistry: sequencer cannot be zero address");
        address oldSequencer = sequencer;
        sequencer = newSequencer;
        emit SequencerUpdated(oldSequencer, newSequencer);
    }

    /**
     * @notice Updates the verifier contract address
     * @dev Only callable by owner. Use with caution - changing verifier affects all future batches.
     * @param newVerifier The address of the new verifier contract
     */
    function updateVerifier(address newVerifier) external onlyOwner {
        require(newVerifier != address(0), "BatchRegistry: verifier cannot be zero address");
        address oldVerifier = address(verifier);
        verifier = IVerifier(newVerifier);
        emit VerifierUpdated(oldVerifier, newVerifier);
    }

    /**
     * @notice Pauses batch registration (emergency function)
     * @dev Only callable by owner. Can be used to stop batch registration in emergencies.
     */
    function pause() external onlyOwner {
        require(!paused, "BatchRegistry: already paused");
        paused = true;
        emit Paused(msg.sender);
    }

    /**
     * @notice Unpauses batch registration
     * @dev Only callable by owner
     */
    function unpause() external onlyOwner {
        require(paused, "BatchRegistry: not paused");
        paused = false;
        emit Unpaused(msg.sender);
    }

    /**
     * @notice Transfers ownership to a new address
     * @dev This allows transferring ownership to a multisig
     * @param newOwner The address of the new owner
     */
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "BatchRegistry: new owner is the zero address");
        address oldOwner = owner;
        owner = newOwner;
        emit OwnershipTransferred(oldOwner, newOwner);
    }
}
