// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";
import {StateManager} from "../src/StateManager.sol";
import {BatchRegistry} from "../src/BatchRegistry.sol";
import {Verifier} from "../src/generated/RollupVerifier.sol";
import {VerifierAdapter} from "../src/VerifierAdapter.sol";

/**
 * @title Deploy
 * @notice Foundry deployment script for zk-sequencer contracts
 * @dev Usage:
 *      Localhost:
 *        forge script script/Deploy.s.sol:Deploy --rpc-url http://localhost:8545 --broadcast -vvvv
 *
 *      Sepolia:
 *        forge script script/Deploy.s.sol:Deploy --rpc-url $SEPOLIA_RPC_URL --broadcast --verify -vvvv
 *
 *      Environment variables required:
 *        - PRIVATE_KEY: Deployer private key
 *        - INITIAL_STATE_ROOT (optional): Custom initial state root
 *        - FINALIZATION_DELAY (optional): Custom finalization delay in seconds
 *        - SEQUENCER_ADDRESS (optional): Custom sequencer address (defaults to deployer)
 */
contract Deploy is Script {
    // Deployment addresses
    StateManager public stateManager;
    BatchRegistry public batchRegistry;
    Verifier public gnarkVerifier;
    VerifierAdapter public verifierAdapter;

    // Configuration
    bytes32 public initialStateRoot;
    address public sequencerAddress;

    // Default values
    bytes32 public constant DEFAULT_INITIAL_STATE_ROOT =
        0x0000000000000000000000000000000000000000000000000000000000000001;

    function setUp() public {
        // Read configuration from environment or use defaults
        initialStateRoot = vm.envOr("INITIAL_STATE_ROOT", DEFAULT_INITIAL_STATE_ROOT);

        // Get deployer address
        uint256 deployerPrivateKey;
        try vm.envUint("PRIVATE_KEY") returns (uint256 key) {
            deployerPrivateKey = key;
        } catch {
            console.log(unicode"❌ Error: PRIVATE_KEY environment variable not found.");
            revert("PRIVATE_KEY not found");
        }
        address deployer = vm.addr(deployerPrivateKey);

        // Default sequencer to deployer if not specified
        sequencerAddress = vm.envOr("SEQUENCER_ADDRESS", deployer);

        console.log("=== Deployment Configuration ===");
        console.log("Deployer:", deployer);

        // Log Balance in ETH
        console.log("Deployer Balance (Wei):", deployer.balance);
        if (deployer.balance < 1 ether / 100) {
            console.log(unicode"⚠️  WARNING: Low balance!");
        }

        console.log("Chain ID:", block.chainid);
        console.log("Initial State Root:", vm.toString(initialStateRoot));

        console.log("Sequencer:", sequencerAddress);
        console.log("");
    }

    function run() external {
        uint256 deployerPrivateKey;
        try vm.envUint("PRIVATE_KEY") returns (uint256 key) {
            deployerPrivateKey = key;
        } catch {
            console.log(unicode"❌ Error: PRIVATE_KEY environment variable not found.");
            console.log("Please set PRIVATE_KEY in .env or environment.");
            revert("PRIVATE_KEY not found");
        }
        vm.startBroadcast(deployerPrivateKey);

        // Step 1: Deploy StateManager
        console.log("1. Deploying StateManager...");
        stateManager = new StateManager(initialStateRoot);
        console.log("   StateManager deployed at:", address(stateManager));
        console.log("");

        // Step 2: Deploy Real Verifier and Adapter
        console.log("2. Deploying Verifier & Adapter...");
        gnarkVerifier = new Verifier();
        console.log("   Gnark Verifier deployed at:", address(gnarkVerifier));

        verifierAdapter = new VerifierAdapter(address(gnarkVerifier));
        console.log("   Verifier Adapter deployed at:", address(verifierAdapter));
        console.log("");

        // Step 3: Deploy BatchRegistry
        console.log("3. Deploying BatchRegistry...");
        // Note: We pass 0 as finalizationDelay because ZK rollups finalize immediately
        batchRegistry = new BatchRegistry(address(verifierAdapter), address(stateManager), sequencerAddress);
        console.log("   BatchRegistry deployed at:", address(batchRegistry));
        console.log("");

        // Step 4: Transfer StateManager ownership to BatchRegistry
        console.log("4. Transferring StateManager ownership to BatchRegistry...");
        stateManager.transferOwnership(address(batchRegistry));
        console.log("   Ownership transferred successfully");
        console.log("");

        vm.stopBroadcast();

        // Step 5: Verify deployment
        console.log("=== Deployment Verification ===");
        verifyDeployment();

        // Step 6: Save deployment addresses
        saveDeploymentAddresses();

        console.log("");
        console.log("=== Deployment Complete ===");
        console.log("StateManager:", address(stateManager));
        console.log("GnarkVerifier:", address(gnarkVerifier));
        console.log("VerifierAdapter:", address(verifierAdapter));
        console.log("BatchRegistry:", address(batchRegistry));
        console.log("");
        console.log("IMPORTANT NEXT STEPS:");
        console.log("1. Transfer BatchRegistry ownership to a multisig");
        console.log("2. Run comprehensive test suite");
        console.log("3. Get security audit");
    }

    function verifyDeployment() internal view {
        console.log("Verifying deployment...");

        // Verify StateManager
        require(stateManager.owner() == address(batchRegistry), "StateManager ownership not transferred");
        require(stateManager.getCurrentStateRoot() == initialStateRoot, "Initial state root mismatch");
        require(stateManager.lastFinalizedBatch() == 0, "Last finalized batch should be 0");
        console.log("  StateManager: OK");

        // Verify BatchRegistry
        require(address(batchRegistry.STATE_MANAGER()) == address(stateManager), "BatchRegistry STATE_MANAGER mismatch");
        require(address(batchRegistry.verifier()) == address(verifierAdapter), "BatchRegistry verifier mismatch");
        require(batchRegistry.sequencer() == sequencerAddress, "Sequencer not set correctly");
        require(batchRegistry.nextBatchNumber() == 1, "Next batch number should be 1");
        require(!batchRegistry.paused(), "BatchRegistry should not be paused");
        console.log("  BatchRegistry: OK");

        // Verify Adapter
        require(address(verifierAdapter.gnarkVerifier()) == address(gnarkVerifier), "Adapter verifier mismatch");
        console.log("  Verifier Adapter: OK");

        console.log("All contracts verified successfully!");
    }

    function saveDeploymentAddresses() internal {
        string memory json = "deployment";

        vm.serializeAddress(json, "StateManager", address(stateManager));
        vm.serializeAddress(json, "GnarkVerifier", address(gnarkVerifier));
        vm.serializeAddress(json, "VerifierAdapter", address(verifierAdapter));
        vm.serializeAddress(json, "BatchRegistry", address(batchRegistry));
        vm.serializeAddress(json, "Sequencer", sequencerAddress);
        vm.serializeAddress(json, "Sequencer", sequencerAddress);

        string memory finalJson = vm.serializeBytes32(json, "InitialStateRoot", initialStateRoot);

        string memory outputDir = string.concat(vm.projectRoot(), "/deployments/");
        string memory outputFile = string.concat(outputDir, "deployments.json");

        vm.writeJson(finalJson, outputFile);
        console.log("");
        console.log("Deployment addresses saved to:", outputFile);
        console.log("Chain ID:", block.chainid);
    }
}
