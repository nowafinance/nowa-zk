// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./generated/Verifier.sol";

/// @title NowaRollup
/// @notice L1 Smart Contract for managing ZK-Rollup deposits, tokens, and State Transitions
contract NowaRollup {
    Verifier public verifier;
    
    // The current state root of the LevelDB Sparse Merkle Tree (L2)
    bytes32 public stateRoot;

    // Token Registry: TokenID -> ERC20 Address
    mapping(uint32 => address) public tokens;
    // ERC20 Address -> TokenID
    mapping(address => uint32) public tokenIds;
    
    uint32 public nextTokenId = 1; // 0 is reserved (or could be ETH)
    
    // Admins who can submit batches
    mapping(address => bool) public isProver;
    address public owner;

    event TokenRegistered(uint32 indexed tokenId, address indexed tokenAddress);
    event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY);
    event Withdrawal(address indexed user, uint32 indexed tokenId, uint256 amount);
    event StateTransition(bytes32 indexed oldRoot, bytes32 indexed newRoot, bytes32 withdrawalHash, bytes32 depositHash);

    constructor(address _verifier) {
        verifier = Verifier(_verifier);
        owner = msg.sender;
        isProver[msg.sender] = true;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    modifier onlyProver() {
        require(isProver[msg.sender], "Not an authorized prover");
        _;
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

        // Use low-level call to handle ERC20 transferFrom to support safe transfers
        (bool success, bytes memory data) = tokenAddress.call(
            abi.encodeWithSignature("transferFrom(address,address,uint256)", msg.sender, address(this), _amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "TransferFrom failed");

        // The Sequencer listens to this event to update the LevelDB Merkle Tree!
        emit Deposit(msg.sender, _tokenId, _amount, _pubKeyX, _pubKeyY);
    }

    /// @notice Processes a withdrawal using a Merkle Proof against the latest withdrawal hash
    function withdraw(uint32 _tokenId, uint256 _amount, bytes32[] calldata _merkleProof) external {
        address tokenAddress = tokens[_tokenId];
        require(tokenAddress != address(0), "Token ID not registered");
        
        // TODO: Implement Merkle verification using the withdrawalHash tree root
        // For now, this is a placeholder interface for Phase 7
        
        // Transfer funds back to user
        (bool success, bytes memory data) = tokenAddress.call(
            abi.encodeWithSignature("transfer(address,uint256)", msg.sender, _amount)
        );
        require(success && (data.length == 0 || abi.decode(data, (bool))), "Transfer failed");
        
        emit Withdrawal(msg.sender, _tokenId, _amount);
    }

    /// @notice Prover submits a ZK-SNARK batch of off-chain transactions
    function submitBatch(
        uint256[8] calldata proof,
        bytes32 _oldRoot,
        bytes32 _newRoot,
        bytes32 _withdrawalHash,
        bytes32 _depositHash
    ) external onlyProver {
        require(_oldRoot == stateRoot, "Invalid old state root");

        // The Verifier expects a public input array.
        // Our GNARK circuit defines 4 public variables: OldRoot, NewRoot, WithdrawalHash, DepositHash.
        uint256[4] memory publicInputs;
        publicInputs[0] = uint256(_oldRoot);
        publicInputs[1] = uint256(_newRoot);
        publicInputs[2] = uint256(_withdrawalHash);
        publicInputs[3] = uint256(_depositHash);

        // This function reverting means the math didn't add up (hacked proof)
        verifier.verifyProof(proof, publicInputs);

        // If we reach here, the cryptographical proof is 100% valid!
        // Update the L1 view of the L2 state.
        stateRoot = _newRoot;
        
        emit StateTransition(_oldRoot, _newRoot, _withdrawalHash, _depositHash);
    }
}
