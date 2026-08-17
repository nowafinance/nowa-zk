// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./generated/Verifier.sol";

/// @title NowaRollup
/// @notice L1 contract for ZK batch settlement with EIP-4844 blob data availability.
/// @dev Batch transition data must be posted in blob index 0 of the submitBatch transaction.
///      Anyone can reconstruct the proven state diffs from the blob sidecar + this contract's roots.
contract NowaRollup {
    Verifier public verifier;

    /// @notice Current L2 Sparse Merkle Tree root
    bytes32 public stateRoot;

    /// @notice Monotonic batch counter (increments on each successful submitBatch)
    uint64 public batchCount;

    /// @notice Token Registry: TokenID -> ERC20 Address
    mapping(uint32 => address) public tokens;
    /// @notice ERC20 Address -> TokenID
    mapping(address => uint32) public tokenIds;

    uint32 public nextTokenId = 1; // 0 is reserved

    /// @notice EIP-4844 versioned blob hash for each settled batch
    mapping(uint64 => bytes32) public batchBlobHash;

    /// @notice keccak256 of the DA payload (also embedded at the start of the blob for easy indexing)
    mapping(uint64 => bytes32) public batchDataHash;

    mapping(address => bool) public isProver;
    address public owner;

    event TokenRegistered(uint32 indexed tokenId, address indexed tokenAddress);
    event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY);
    event Withdrawal(address indexed user, uint32 indexed tokenId, uint256 amount);
    event StateTransition(
        uint64 indexed batchId,
        bytes32 indexed oldRoot,
        bytes32 indexed newRoot,
        bytes32 withdrawalHash,
        bytes32 depositHash,
        bytes32 blobHash,
        bytes32 dataHash
    );

    constructor(address _verifier, bytes32 _initialStateRoot) {
        verifier = Verifier(_verifier);
        owner = msg.sender;
        isProver[msg.sender] = true;
        stateRoot = _initialStateRoot;
    }

    /// @notice Bootstrap / resync L1 root before any batch is settled (owner only).
    function setStateRoot(bytes32 _root) external onlyOwner {
        require(batchCount == 0, "batches already settled");
        stateRoot = _root;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    modifier onlyProver() {
        require(isProver[msg.sender], "Not an authorized prover");
        _;
    }

    function setProver(address _prover, bool _allowed) external onlyOwner {
        isProver[_prover] = _allowed;
    }

    /// @notice Registers an ERC20 token to be traded on the L2
    function registerToken(address _tokenAddress) external onlyOwner {
        require(tokenIds[_tokenAddress] == 0, "Token already registered");
        require(nextTokenId < 256, "Max 256 tokens supported in Merkle Tree");

        uint32 tokenId = nextTokenId++;
        tokens[tokenId] = _tokenAddress;
        tokenIds[_tokenAddress] = tokenId;

        emit TokenRegistered(tokenId, _tokenAddress);
    }

    /// @notice User deposits L1 tokens to be bridged to the L2
    function deposit(uint32 _tokenId, uint256 _amount, uint256 _pubKeyX, uint256 _pubKeyY) external {
        address tokenAddress = tokens[_tokenId];
        require(tokenAddress != address(0), "Token ID not registered");
        require(_amount > 0, "Deposit amount must be > 0");

        (bool success, bytes memory data) = tokenAddress.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", msg.sender, address(this), _amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "TransferFrom failed");

        emit Deposit(msg.sender, _tokenId, _amount, _pubKeyX, _pubKeyY);
    }

    /// @notice Placeholder withdrawal (operator/bridge path). Not an escape hatch.
    function withdraw(uint32 _tokenId, uint256 _amount, bytes32[] calldata /* _merkleProof */) external onlyOwner {
        address tokenAddress = tokens[_tokenId];
        require(tokenAddress != address(0), "Token ID not registered");

        (bool success, bytes memory data) = tokenAddress.call(
            abi.encodeWithSignature("transfer(address,uint256)", msg.sender, _amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "Transfer failed");

        emit Withdrawal(msg.sender, _tokenId, _amount);
    }

    /// @notice Prover submits a ZK-SNARK batch. Must be an EIP-4844 tx with DA blob at index 0.
    /// @param proof Groth16 proof (8 field elements)
    /// @param _oldRoot Previous SMT root (must match stateRoot)
    /// @param _newRoot New SMT root after the batch
    /// @param _withdrawalHash Public withdrawal accumulator
    /// @param _depositHash Public deposit accumulator
    /// @param _dataHash keccak256 of the canonical DA payload (must match bytes stored in the blob)
    function submitBatch(
        uint256[8] calldata proof,
        bytes32 _oldRoot,
        bytes32 _newRoot,
        bytes32 _withdrawalHash,
        bytes32 _depositHash,
        bytes32 _dataHash
    ) external onlyProver {
        require(_oldRoot == stateRoot, "Invalid old state root");
        require(_dataHash != bytes32(0), "Empty data hash");

        // EIP-4844: blob versioned hash for blob index 0 must be present on this tx.
        bytes32 blobHash = blobhash(0);
        require(blobHash != bytes32(0), "DA blob required");

        uint256[4] memory publicInputs;
        publicInputs[0] = uint256(_oldRoot);
        publicInputs[1] = uint256(_newRoot);
        publicInputs[2] = uint256(_withdrawalHash);
        publicInputs[3] = uint256(_depositHash);

        verifier.verifyProof(proof, publicInputs);

        uint64 batchId = batchCount;
        batchCount = batchId + 1;

        stateRoot = _newRoot;
        batchBlobHash[batchId] = blobHash;
        batchDataHash[batchId] = _dataHash;

        emit StateTransition(batchId, _oldRoot, _newRoot, _withdrawalHash, _depositHash, blobHash, _dataHash);
    }
}
