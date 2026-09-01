// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

/// @notice Minimal, permissionless-mint ERC20 for local/testnet development only —
/// same shape as contracts/test/NowaRollup.t.sol's MockERC20, duplicated here (not
/// imported from the test file) so this script has no dependency on test code.
/// Never deploy this to anything but a test network.
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

/// @notice Deploys MockERC20 and mints a large test supply to the deployer — for
/// registerToken()/deposit() testing against NowaRollup on a testnet.
contract DeployMockToken is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address deployer = vm.addr(deployerPrivateKey);

        vm.startBroadcast(deployerPrivateKey);
        MockERC20 token = new MockERC20();
        token.mint(deployer, 1_000_000 ether);
        vm.stopBroadcast();

        console.log("MockERC20 deployed at:", address(token));
        console.log("Minted 1,000,000 (18-decimals) to deployer:", deployer);
    }
}
