// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/NowaRollup.sol";
import "../src/generated/Verifier.sol";

contract MockERC20 {
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    
    function mint(address to, uint256 amount) public {
        balanceOf[to] += amount;
    }
    
    function approve(address spender, uint256 amount) public returns (bool) {
        allowance[msg.sender][spender] = amount;
        return true;
    }
    
    function transferFrom(address from, address to, uint256 amount) public returns (bool) {
        require(allowance[from][msg.sender] >= amount, "ERC20: insufficient allowance");
        require(balanceOf[from] >= amount, "ERC20: insufficient balance");
        allowance[from][msg.sender] -= amount;
        balanceOf[from] -= amount;
        balanceOf[to] += amount;
        return true;
    }
    
    function transfer(address to, uint256 amount) public returns (bool) {
        require(balanceOf[msg.sender] >= amount, "ERC20: insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
}

contract NowaRollupTest is Test {
    NowaRollup rollup;
    Verifier verifier;
    MockERC20 mockToken;
    
    address owner = address(this);
    address alice = address(0x1);
    
    event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY);
    
    function setUp() public {
        verifier = new Verifier();
        rollup = new NowaRollup(address(verifier));
        
        mockToken = new MockERC20();
        
        // Register token
        rollup.registerToken(address(mockToken));
    }
    
    function testRegisterToken() public {
        assertEq(rollup.nextTokenId(), 2);
        assertEq(rollup.tokens(1), address(mockToken));
        assertEq(rollup.tokenIds(address(mockToken)), 1);
    }
    
    function testDeposit() public {
        // Setup Alice
        mockToken.mint(alice, 1000);
        vm.prank(alice);
        mockToken.approve(address(rollup), 500);
        
        // Deposit
        vm.expectEmit(true, true, false, true);
        emit Deposit(alice, 1, 500, 12345, 67890);
        
        vm.prank(alice);
        rollup.deposit(1, 500, 12345, 67890); // Dummy pubkey coords
        
        // Check balances
        assertEq(mockToken.balanceOf(alice), 500);
        assertEq(mockToken.balanceOf(address(rollup)), 500);
    }
    
    function testCannotDepositUnregisteredToken() public {
        vm.prank(alice);
        vm.expectRevert("Token ID not registered");
        rollup.deposit(999, 100, 123, 456);
    }
    
    // We can't trivially test submitBatch because we need a valid SNARK proof!
    // But we can test that it reverts with an invalid proof.
    function testSubmitBatchRevertsWithInvalidProof() public {
        uint256[8] memory proof;
        // The Verifier expects groth16 proofs to be mathematically valid on BN254.
        // Sending all 0s will revert.
        vm.expectRevert();
        rollup.submitBatch(proof, bytes32(0), bytes32(0), bytes32(0), bytes32(0));
    }
}
