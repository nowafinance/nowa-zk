// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./generated/Verifier.sol";
import "./MiMC.sol";

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

    /// @notice Escape hatch: fixed at deploy, never owner-adjustable.
    uint256 public immutable escapeTimeout;

    /// @notice Timestamp of the last successful submitBatch (or deploy, if none yet).
    uint256 public lastBatchAt;

    /// @notice keccak256(pubKeyX, pubKeyY) -> first address that ever deposited into that L2 pubkey.
    /// @dev First-depositor-wins: prevents an attacker depositing dust into someone else's
    ///      already-funded pubkey to hijack escape-hatch rights over it.
    mapping(bytes32 => address) public depositorOf;

    /// @notice Leaf index -> already claimed via emergencyWithdraw (prevents double-withdrawal).
    mapping(uint256 => bool) public escapeWithdrawn;

    event TokenRegistered(uint32 indexed tokenId, address indexed tokenAddress);
    event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY);
    event Withdrawal(address indexed user, uint32 indexed tokenId, uint256 amount);
    event EscapeWithdrawal(address indexed user, uint32 indexed tokenId, uint256 indexed index, uint256 amount);
    event StateTransition(
        uint64 indexed batchId,
        bytes32 indexed oldRoot,
        bytes32 indexed newRoot,
        bytes32 withdrawalHash,
        bytes32 depositHash,
        bytes32 blobHash,
        bytes32 dataHash
    );

    constructor(address _verifier, bytes32 _initialStateRoot, uint256 _escapeTimeout) {
        verifier = Verifier(_verifier);
        owner = msg.sender;
        isProver[msg.sender] = true;
        stateRoot = _initialStateRoot;
        escapeTimeout = _escapeTimeout;
        lastBatchAt = block.timestamp;
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

        bytes32 key = keccak256(abi.encode(_pubKeyX, _pubKeyY));
        if (depositorOf[key] == address(0)) {
            depositorOf[key] = msg.sender;
        }

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
        lastBatchAt = block.timestamp;

        emit StateTransition(batchId, _oldRoot, _newRoot, _withdrawalHash, _depositHash, blobHash, _dataHash);
    }

    /// @notice Parameters for emergencyWithdraw, bundled into a struct — Solidity's legacy codegen
    ///         hits "stack too deep" with this many individual parameters (2 of them length-28
    ///         arrays) on a single external function.
    struct EscapeProof {
        uint32 tokenId;
        uint256 balance;
        uint256 nonce;
        uint256 pubX;
        uint256 pubY;
        uint256 index;
        bytes32[28] siblings;
        bool[28] pathBits;
    }

    /// @notice Trustless fallback withdrawal if the Sequencer has stalled for `escapeTimeout`.
    /// @dev Deposit-bound: only the address recorded in `depositorOf` for this leaf's pubkey may
    ///      claim it (first-depositor-wins, set in `deposit()`). This does NOT cover balance a
    ///      pubkey only ever received via L2 trades and never deposited into directly — that's a
    ///      documented limitation, not an oversight (see docs/architecture/overview.md).
    /// @param p.tokenId Token being withdrawn — must match `index % 256` (the circuit's leaf layout).
    /// @param p.balance The leaf's full current balance for this token — withdrawn in full, no partial exit.
    /// @param p.nonce The leaf's current nonce (part of the leaf hash the Sequencer/circuit compute).
    /// @param p.pubX L2 EdDSA public key X coordinate for this account.
    /// @param p.pubY L2 EdDSA public key Y coordinate for this account.
    /// @param p.index Merkle leaf index (`accountID*256 + tokenId`).
    /// @param p.siblings The 28 sibling hashes from leaf to root (depth-28 SMT).
    /// @param p.pathBits Per-level direction bits; `true` means this node is the right child (matches
    ///        prover/circuits/state_circuit.go's merkleRoot() convention exactly).
    function emergencyWithdraw(EscapeProof calldata p) external {
        require(block.timestamp > lastBatchAt + escapeTimeout, "Sequencer not stalled");
        require(p.index % 256 == p.tokenId, "Token/index mismatch");
        bytes32 key = keccak256(abi.encode(p.pubX, p.pubY));
        require(depositorOf[key] == msg.sender, "Not original depositor");
        require(!escapeWithdrawn[p.index], "Already withdrawn");

        uint256 leaf = _accountLeaf(p.index, p.pubX, p.pubY, p.balance, p.nonce);
        uint256 root = _foldMerklePath(leaf, p.siblings, p.pathBits);
        require(bytes32(root) == stateRoot, "Invalid Merkle proof");

        // Effects before interaction.
        escapeWithdrawn[p.index] = true;

        address tokenAddress = tokens[p.tokenId];
        require(tokenAddress != address(0), "Token ID not registered");
        (bool success, bytes memory data) = tokenAddress.call(
            abi.encodeWithSignature("transfer(address,uint256)", msg.sender, p.balance)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "Transfer failed");

        emit EscapeWithdrawal(msg.sender, p.tokenId, p.index, p.balance);
    }

    /// @dev Mirrors accountLeaf() in prover/circuits/state_circuit.go: MiMC(index, pubX, pubY, balance, nonce).
    function _accountLeaf(uint256 index, uint256 pubX, uint256 pubY, uint256 balance, uint256 nonce)
        internal
        pure
        returns (uint256)
    {
        uint256[] memory leafData = new uint256[](5);
        leafData[0] = index;
        leafData[1] = pubX;
        leafData[2] = pubY;
        leafData[3] = balance;
        leafData[4] = nonce;
        return MiMC.hash(leafData);
    }

    /// @dev Mirrors merkleRoot() in prover/circuits/state_circuit.go: bit=1 means the current
    ///      node is the right child, sibling goes on the left; bit=0 is the reverse.
    function _foldMerklePath(uint256 leaf, bytes32[28] calldata siblings, bool[28] calldata pathBits)
        internal
        pure
        returns (uint256)
    {
        uint256 cur = leaf;
        for (uint256 i = 0; i < 28; i++) {
            uint256[] memory pair = new uint256[](2);
            if (pathBits[i]) {
                pair[0] = uint256(siblings[i]);
                pair[1] = cur;
            } else {
                pair[0] = cur;
                pair[1] = uint256(siblings[i]);
            }
            cur = MiMC.hash(pair);
        }
        return cur;
    }
}
