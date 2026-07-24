// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";
import {TradeRegistry} from "../src/TradeRegistry.sol";
import {Verifier} from "../src/generated/TradeVerifier.sol";

contract Deploy is Script {
    TradeRegistry public tradeRegistry;
    Verifier public tradeVerifier;

    function setUp() public {
        // Setup logic here if needed
    }

    function run() external {
        uint256 deployerPrivateKey;
        try vm.envUint("PRIVATE_KEY") returns (uint256 key) {
            deployerPrivateKey = key;
        } catch {
            console.log(unicode"❌ Error: PRIVATE_KEY environment variable not found.");
            revert("PRIVATE_KEY not found");
        }
        vm.startBroadcast(deployerPrivateKey);

        // Step 1: Deploy Verifier
        console.log("1. Deploying TradeVerifier...");
        tradeVerifier = new Verifier();
        console.log("   TradeVerifier deployed at:", address(tradeVerifier));
        console.log("");

        // Step 2: Deploy TradeRegistry
        console.log("2. Deploying TradeRegistry...");
        tradeRegistry = new TradeRegistry(address(tradeVerifier));
        console.log("   TradeRegistry deployed at:", address(tradeRegistry));
        console.log("");

        vm.stopBroadcast();

        // Step 3: Save deployment addresses
        saveDeploymentAddresses();
    }

    function saveDeploymentAddresses() internal {
        string memory json = "deployment";

        vm.serializeAddress(json, "TradeVerifier", address(tradeVerifier));
        vm.serializeAddress(json, "TradeRegistry", address(tradeRegistry));
        // We still serialize BatchRegistry key just in case older tooling expects it, but map to TradeRegistry
        vm.serializeAddress(json, "BatchRegistry", address(tradeRegistry));

        string memory outputDir = string.concat(vm.projectRoot(), "/deployments/");
        string memory outputFile = string.concat(outputDir, "deployments.json");

        if (!vm.isDir(outputDir)) {
            vm.createDir(outputDir, true);
        }

        string memory finalJson = vm.serializeAddress(json, "TradeRegistry", address(tradeRegistry));
        vm.writeJson(finalJson, outputFile);
        
        console.log("");
        console.log("Deployment addresses saved to:", outputFile);
    }
}
