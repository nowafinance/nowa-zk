// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "./interfaces/IVerifier.sol";
import "./StateManager.sol";

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

    /// @notice Emitted when finalization delay is updated
    event FinalizationDelayUpdated(uint256 oldDelay, uint256 newDelay);

    /// @notice Emitted when ownership is transferred
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

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

    /// @notice Finalization delay in seconds (default: 7 days)
    uint256 public finalizationDelay;

    /// @notice Minimum finalization delay (1 hour)
    uint256 public constant MIN_FINALIZATION_DELAY = 1 hours;

    /// @notice Maximum finalization delay (30 days)
    uint256 public constant MAX_FINALIZATION_DELAY = 30 days;

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
     * @param _finalizationDelay Time delay before finalization (in seconds)
     */
    constructor(address _verifier, address _stateManager, address _sequencer, uint256 _finalizationDelay) {
        require(_verifier != address(0), "BatchRegistry: verifier cannot be zero address");
        require(_stateManager != address(0), "BatchRegistry: stateManager cannot be zero address");
        require(_sequencer != address(0), "BatchRegistry: sequencer cannot be zero address");
        require(
            _finalizationDelay >= MIN_FINALIZATION_DELAY && _finalizationDelay <= MAX_FINALIZATION_DELAY,
            "BatchRegistry: invalid finalization delay"
        );

        owner = msg.sender;
        verifier = IVerifier(_verifier);
        STATE_MANAGER = StateManager(_stateManager);
        sequencer = _sequencer;
        finalizationDelay = _finalizationDelay;
        nextBatchNumber = 1;
    }

    /**
     * @notice Registers a new batch with proof verification
     * @dev This function verifies the ZK proof and stores batch metadata.
     *      The batch is marked as Verified but not Finalized until the challenge period passes.
     *      Data availability is ensured by storing batchData in calldata.
     *
     * @param batchHash The hash of the batch being registered
     * @param newStateRoot The state root after batch execution
     * @param batchData The raw batch data (stored in calldata for data availability)
     * @param proofA The first part of the ZK proof (G1 point)
     * @param proofB The second part of the ZK proof (G2 point)
     * @param proofC The third part of the ZK proof (G1 point)
     * @param publicInputs The public inputs to the proof [oldStateRoot, newStateRoot, batchNumber]
     *
     * @return batchNumber The assigned batch number
     *
     * @custom:security This function performs critical proof verification. All inputs must be validated.
     */
    function registerBatch(
        bytes32 batchHash,
        bytes32 newStateRoot,
        bytes calldata batchData,
        uint256[2] calldata proofA,
        uint256[2][2] calldata proofB,
        uint256[2] calldata proofC,
        uint256[3] calldata publicInputs
    ) external onlySequencer whenNotPaused returns (uint256 batchNumber) {
        // Input validation
        require(batchHash != bytes32(0), "BatchRegistry: batch hash cannot be zero");
        require(newStateRoot != bytes32(0), "BatchRegistry: new state root cannot be zero");
        require(batchData.length > 0, "BatchRegistry: batch data cannot be empty");
        require(batchHashToNumber[batchHash] == 0, "BatchRegistry: batch hash already exists");

        // Verify batch hash matches data (data availability check)
        require(keccak256(batchData) == batchHash, "BatchRegistry: batch hash mismatch");

        // Get current state root from StateManager
        bytes32 oldStateRoot = STATE_MANAGER.getCurrentStateRoot();
        require(oldStateRoot != newStateRoot, "BatchRegistry: state root unchanged");

        // Assign batch number
        batchNumber = nextBatchNumber;

        // Validate public inputs
        require(bytes32(publicInputs[0]) == oldStateRoot, "BatchRegistry: publicInputs[0] must match oldStateRoot");
        require(bytes32(publicInputs[1]) == newStateRoot, "BatchRegistry: publicInputs[1] must match newStateRoot");
        require(publicInputs[2] == batchNumber, "BatchRegistry: publicInputs[2] must match batchNumber");

        // Verify the ZK proof
        bool proofValid = verifier.verifyProof(proofA, proofB, proofC, publicInputs);
        require(proofValid, "BatchRegistry: invalid proof");

        // Store batch metadata (verified but not finalized)
        batches[batchNumber] = Batch({
            batchHash: batchHash,
            oldStateRoot: oldStateRoot,
            newStateRoot: newStateRoot,
            submitter: msg.sender,
            timestamp: block.timestamp,
            verifiedAt: block.number,
            status: BatchStatus.Verified
        });

        batchHashToNumber[batchHash] = batchNumber;
        totalBatches++;
        nextBatchNumber++;

        emit BatchRegistered(batchNumber, batchHash, oldStateRoot, newStateRoot, msg.sender, block.timestamp);

        return batchNumber;
    }

    /**
     * @notice Finalizes a verified batch after the finalization delay
     * @dev Updates the StateManager with the new state root. Batches must be finalized sequentially.
     *      Only callable by owner for controlled finalization.
     *
     * @param batchNumber The batch number to finalize
     */
    function finalizeBatch(uint256 batchNumber) external onlyOwner whenNotPaused {
        Batch storage batch = batches[batchNumber];

        require(batch.status == BatchStatus.Verified, "BatchRegistry: batch not verified");
        require(block.timestamp >= batch.timestamp + finalizationDelay, "BatchRegistry: finalization delay not met");

        // Ensure sequential finalization
        if (batchNumber > 1) {
            require(
                batches[batchNumber - 1].status == BatchStatus.Finalized, "BatchRegistry: previous batch not finalized"
            );
        }

        // Update batch status
        batch.status = BatchStatus.Finalized;

        // Update StateManager with the new state root
        STATE_MANAGER.updateStateRoot(batch.newStateRoot, batchNumber);

        emit BatchFinalized(batchNumber, batch.newStateRoot);
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

    /**
     * @notice Checks if a batch can be finalized
     * @param batchNumber The batch number to check
     * @return canFinalize True if the batch can be finalized
     */
    function canFinalizeBatch(uint256 batchNumber) external view returns (bool canFinalize) {
        Batch storage batch = batches[batchNumber];

        if (batch.status != BatchStatus.Verified) return false;
        if (block.timestamp < batch.timestamp + finalizationDelay) return false;
        if (batchNumber > 1 && batches[batchNumber - 1].status != BatchStatus.Finalized) return false;

        return true;
    }

    /**
     * @notice Gets the time remaining until a batch can be finalized
     * @param batchNumber The batch number to check
     * @return timeRemaining Seconds remaining (0 if ready or invalid)
     */
    function timeUntilFinalization(uint256 batchNumber) external view returns (uint256 timeRemaining) {
        Batch storage batch = batches[batchNumber];

        if (batch.status != BatchStatus.Verified) return 0;

        uint256 finalizationTime = batch.timestamp + finalizationDelay;
        if (block.timestamp >= finalizationTime) return 0;

        return finalizationTime - block.timestamp;
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
        verifier = IVerifier(newVerifier);
    }

    /**
     * @notice Updates the finalization delay
     * @dev Only callable by owner
     * @param newDelay The new finalization delay in seconds
     */
    function updateFinalizationDelay(uint256 newDelay) external onlyOwner {
        require(
            newDelay >= MIN_FINALIZATION_DELAY && newDelay <= MAX_FINALIZATION_DELAY,
            "BatchRegistry: invalid finalization delay"
        );
        uint256 oldDelay = finalizationDelay;
        finalizationDelay = newDelay;
        emit FinalizationDelayUpdated(oldDelay, newDelay);
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
