// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

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

    address alice = address(0x1);

    event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY);

    function setUp() public {
        verifier = new Verifier();
        rollup = new NowaRollup(address(verifier), bytes32(0));

        mockToken = new MockERC20();
        rollup.registerToken(address(mockToken));
    }

    function testRegisterToken() public {
        assertEq(rollup.nextTokenId(), 2);
        assertEq(rollup.tokens(1), address(mockToken));
        assertEq(rollup.tokenIds(address(mockToken)), 1);
    }

    function testDeposit() public {
        mockToken.mint(alice, 1000);
        vm.prank(alice);
        mockToken.approve(address(rollup), 500);

        vm.expectEmit(true, true, false, true);
        emit Deposit(alice, 1, 500, 12345, 67890);

        vm.prank(alice);
        rollup.deposit(1, 500, 12345, 67890);

        assertEq(mockToken.balanceOf(alice), 500);
        assertEq(mockToken.balanceOf(address(rollup)), 500);
    }

    function testCannotDepositUnregisteredToken() public {
        vm.prank(alice);
        vm.expectRevert("Token ID not registered");
        rollup.deposit(999, 100, 123, 456);
    }

    function testSubmitBatchRequiresBlob() public {
        uint256[8] memory proof;
        // No blobhash set → must revert with DA blob required (before proof verify)
        vm.expectRevert("DA blob required");
        rollup.submitBatch(proof, bytes32(0), bytes32(0), bytes32(0), bytes32(0), bytes32(uint256(1)));
    }

    function testSubmitBatchRevertsWithInvalidProofWhenBlobPresent() public {
        uint256[8] memory proof;
        bytes32[] memory hashes = new bytes32[](1);
        // versioned blob hash prefix 0x01
        hashes[0] = bytes32(uint256(0x01) << 248 | uint256(0xdead));
        vm.blobhashes(hashes);

        vm.expectRevert(); // verifier rejects invalid proof
        rollup.submitBatch(proof, bytes32(0), bytes32(0), bytes32(0), bytes32(0), bytes32(uint256(1)));
    }

    function testDepositZeroAmountReverts() public {
        vm.prank(alice);
        vm.expectRevert("Deposit amount must be > 0");
        rollup.deposit(1, 0, 12345, 67890);
    }

    function testSubmitBatchAsNonProverReverts() public {
        uint256[8] memory proof;
        bytes32[] memory hashes = new bytes32[](1);
        hashes[0] = bytes32(uint256(0x01) << 248);
        vm.blobhashes(hashes);

        vm.prank(alice);
        vm.expectRevert("Not an authorized prover");
        rollup.submitBatch(proof, bytes32(0), bytes32(0), bytes32(0), bytes32(0), bytes32(uint256(1)));
    }

    function testWithdrawOnlyOwner() public {
        mockToken.mint(address(rollup), 1000);
        bytes32[] memory emptyProof;

        vm.prank(alice);
        vm.expectRevert("Not owner");
        rollup.withdraw(1, 300, emptyProof);
    }
}
