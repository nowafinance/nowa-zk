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
        // `make deploy` computes this automatically (sequencer/cmd/print-root, reading
        // whatever's actually in the Sequencer's LevelDB tree right now) and passes it
        // as INITIAL_STATE_ROOT — a fresh/empty tree does NOT root to 0, so defaulting
        // to bytes32(0) here would make every submitBatch() revert until manually
        // fixed. Only falls back to 0 if INITIAL_STATE_ROOT truly isn't set at all
        // (e.g. calling this script directly, bypassing the Makefile).
        bytes32 initialRoot = bytes32(0);
        try vm.envBytes32("INITIAL_STATE_ROOT") returns (bytes32 r) {
            initialRoot = r;
        } catch {}
        // Escape hatch: how long submitBatch() can go quiet before emergencyWithdraw() unlocks.
        // Fixed at deploy, never owner-adjustable afterward. Defaults to 7 days (matches the
        // figure documented in FAQ-ZK.md / the internal roadmap) — override for local/testnet
        // deployments where waiting a real week isn't practical.
        uint256 escapeTimeout = 7 days;
        try vm.envUint("ESCAPE_TIMEOUT_SECONDS") returns (uint256 t) {
            escapeTimeout = t;
        } catch {}
        nowaRollup = new NowaRollup(address(verifier), initialRoot, escapeTimeout);
        console.log("   NowaRollup deployed at:", address(nowaRollup));
        console.log("   initial stateRoot:", vm.toString(initialRoot));
        console.log("   escapeTimeout (seconds):", escapeTimeout);
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
