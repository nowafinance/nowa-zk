// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";
import {NowaRollup} from "../src/NowaRollup.sol";
import {Verifier} from "../src/generated/Verifier.sol";

contract Deploy is Script {
    NowaRollup public nowaRollup;
    Verifier public verifier;

    function setUp() public {}

    function run() external {
        uint256 deployerPrivateKey;
        try vm.envUint("PRIVATE_KEY") returns (uint256 key) {
            deployerPrivateKey = key;
        } catch {
            console.log(unicode"❌ Error: PRIVATE_KEY environment variable not found.");
            revert("PRIVATE_KEY not found");
        }
        vm.startBroadcast(deployerPrivateKey);

        console.log("1. Deploying Verifier...");
        verifier = new Verifier();
        console.log("   Verifier deployed at:", address(verifier));
        console.log("");

        console.log("2. Deploying NowaRollup...");
        nowaRollup = new NowaRollup(address(verifier));
        console.log("   NowaRollup deployed at:", address(nowaRollup));
        console.log("");

        vm.stopBroadcast();

        saveDeploymentAddresses();
    }

    function saveDeploymentAddresses() internal {
        string memory json = "deployment";
        vm.serializeAddress(json, "Verifier", address(verifier));
        string memory finalJson = vm.serializeAddress(json, "NowaRollup", address(nowaRollup));

        string memory outputDir = string.concat(vm.projectRoot(), "/deployments/");
        string memory outputFile = string.concat(outputDir, "deployments.json");

        if (!vm.isDir(outputDir)) {
            vm.createDir(outputDir, true);
        }

        vm.writeJson(finalJson, outputFile);

        console.log("");
        console.log("Deployment addresses saved to:", outputFile);
    }
}
