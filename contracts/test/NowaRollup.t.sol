// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/NowaRollup.sol";
import "../src/generated/Verifier.sol";
import "../src/MiMC.sol";

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

    uint256 constant ESCAPE_TIMEOUT = 7 days;

    function setUp() public {
        verifier = new Verifier();
        rollup = new NowaRollup(address(verifier), bytes32(0), ESCAPE_TIMEOUT);

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

    // ============================================================
    // Escape Hatch (emergencyWithdraw)
    // ============================================================

    /// @dev Depth-28 SMT with every leaf empty except one — a single-leaf tree is enough to
    ///      exercise real Merkle folding without needing a full Sequencer-produced fixture.
    ///      zeros[0] is the empty-leaf value (0, matching prover/circuits' leaf convention);
    ///      zeros[i] is the root of an empty subtree of depth i.
    function _zeroNodes() internal pure returns (uint256[28] memory zeros) {
        zeros[0] = 0;
        for (uint256 i = 1; i < 28; i++) {
            uint256[] memory pair = new uint256[](2);
            pair[0] = zeros[i - 1];
            pair[1] = zeros[i - 1];
            zeros[i] = MiMC.hash(pair);
        }
    }

    /// @dev Builds siblings/pathBits for `index` in a tree where every leaf except `index` is
    ///      empty, and returns the resulting root for `leafHash` at that index.
    function _buildProof(uint256 leafHash, uint256 index)
        internal
        pure
        returns (bytes32[28] memory siblings, bool[28] memory pathBits, uint256 root)
    {
        uint256[28] memory zeros = _zeroNodes();
        uint256 cur = leafHash;
        for (uint256 i = 0; i < 28; i++) {
            bool bit = (index >> i) & 1 == 1;
            pathBits[i] = bit;
            siblings[i] = bytes32(zeros[i]);
            uint256[] memory pair = new uint256[](2);
            if (bit) {
                pair[0] = zeros[i];
                pair[1] = cur;
            } else {
                pair[0] = cur;
                pair[1] = zeros[i];
            }
            cur = MiMC.hash(pair);
        }
        root = cur;
    }

    function _accountLeaf(uint256 index, uint256 pubX, uint256 pubY, uint256 balance, uint256 nonce)
        internal
        pure
        returns (uint256)
    {
        uint256[] memory data = new uint256[](5);
        data[0] = index;
        data[1] = pubX;
        data[2] = pubY;
        data[3] = balance;
        data[4] = nonce;
        return MiMC.hash(data);
    }

    /// @dev One consistent fixture reused across the escape-hatch tests: Alice deposits `balance`
    ///      of mockToken against (pubX, pubY), which both funds the contract and records Alice as
    ///      the depositor; the state root is set to match a tree containing exactly that leaf.
    function _fundEscapeFixture(uint256 index, uint256 pubX, uint256 pubY, uint256 balance)
        internal
        returns (NowaRollup.EscapeProof memory proof)
    {
        mockToken.mint(alice, balance);
        vm.prank(alice);
        mockToken.approve(address(rollup), balance);
        vm.prank(alice);
        rollup.deposit(1, balance, pubX, pubY);

        uint256 leaf = _accountLeaf(index, pubX, pubY, balance, 0);
        (bytes32[28] memory siblings, bool[28] memory pathBits, uint256 root) = _buildProof(leaf, index);
        rollup.setStateRoot(bytes32(root));

        proof = NowaRollup.EscapeProof({
            tokenId: 1,
            balance: balance,
            nonce: 0,
            pubX: pubX,
            pubY: pubY,
            index: index,
            siblings: siblings,
            pathBits: pathBits
        });
    }

    function testEmergencyWithdrawSucceedsAfterTimeout() public {
        NowaRollup.EscapeProof memory proof = _fundEscapeFixture(1281, 111, 222, 500);
        vm.warp(block.timestamp + ESCAPE_TIMEOUT + 1);

        vm.expectEmit(true, true, true, true);
        emit NowaRollup.EscapeWithdrawal(alice, 1, 1281, 500);

        vm.prank(alice);
        rollup.emergencyWithdraw(proof);

        assertEq(mockToken.balanceOf(alice), 500);
        assertTrue(rollup.escapeWithdrawn(1281));
    }

    function testEmergencyWithdrawRevertsBeforeTimeout() public {
        NowaRollup.EscapeProof memory proof = _fundEscapeFixture(1281, 111, 222, 500);
        // No warp — still within the liveness window.
        vm.prank(alice);
        vm.expectRevert("Sequencer not stalled");
        rollup.emergencyWithdraw(proof);
    }

    function testEmergencyWithdrawRevertsForWrongCaller() public {
        NowaRollup.EscapeProof memory proof = _fundEscapeFixture(1281, 111, 222, 500);
        vm.warp(block.timestamp + ESCAPE_TIMEOUT + 1);

        address bob = address(0x2);
        vm.prank(bob);
        vm.expectRevert("Not original depositor");
        rollup.emergencyWithdraw(proof);
    }

    function testEmergencyWithdrawRevertsOnDoubleWithdraw() public {
        NowaRollup.EscapeProof memory proof = _fundEscapeFixture(1281, 111, 222, 500);
        vm.warp(block.timestamp + ESCAPE_TIMEOUT + 1);

        vm.prank(alice);
        rollup.emergencyWithdraw(proof);

        vm.prank(alice);
        vm.expectRevert("Already withdrawn");
        rollup.emergencyWithdraw(proof);
    }

    function testEmergencyWithdrawRevertsOnTamperedBalance() public {
        NowaRollup.EscapeProof memory proof = _fundEscapeFixture(1281, 111, 222, 500);
        vm.warp(block.timestamp + ESCAPE_TIMEOUT + 1);

        proof.balance = 999999; // doesn't match the leaf the root was built from
        vm.prank(alice);
        vm.expectRevert("Invalid Merkle proof");
        rollup.emergencyWithdraw(proof);
    }

    function testEmergencyWithdrawRevertsOnTokenIndexMismatch() public {
        NowaRollup.EscapeProof memory proof = _fundEscapeFixture(1281, 111, 222, 500);
        vm.warp(block.timestamp + ESCAPE_TIMEOUT + 1);

        proof.tokenId = 2; // 1281 % 256 == 1, not 2
        vm.prank(alice);
        vm.expectRevert("Token/index mismatch");
        rollup.emergencyWithdraw(proof);
    }

    function testDepositorOfIsFirstDepositorWins() public {
        mockToken.mint(alice, 500);
        vm.prank(alice);
        mockToken.approve(address(rollup), 500);
        vm.prank(alice);
        rollup.deposit(1, 500, 111, 222);

        address bob = address(0x2);
        mockToken.mint(bob, 1);
        vm.prank(bob);
        mockToken.approve(address(rollup), 1);
        vm.prank(bob);
        rollup.deposit(1, 1, 111, 222); // same pubkey — should NOT steal escape rights

        bytes32 key = keccak256(abi.encode(uint256(111), uint256(222)));
        assertEq(rollup.depositorOf(key), alice);
    }
}
