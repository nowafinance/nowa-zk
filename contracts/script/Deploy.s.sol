// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "forge-std/Script.sol";
import "../src/StateManager.sol";
import "../src/BatchRegistry.sol";
import "../src/mocks/MockVerifier.sol";

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
    MockVerifier public mockVerifier;

    // Configuration
    bytes32 public initialStateRoot;
    uint256 public finalizationDelay;
    address public sequencerAddress;

    // Default values
    bytes32 public constant DEFAULT_INITIAL_STATE_ROOT =
        0x0000000000000000000000000000000000000000000000000000000000000001;
    uint256 public constant DEFAULT_FINALIZATION_DELAY = 7 days;

    function setUp() public {
        // Read configuration from environment or use defaults
        initialStateRoot = vm.envOr("INITIAL_STATE_ROOT", DEFAULT_INITIAL_STATE_ROOT);
        finalizationDelay = vm.envOr("FINALIZATION_DELAY", DEFAULT_FINALIZATION_DELAY);

        // Get deployer address
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address deployer = vm.addr(deployerPrivateKey);

        // Default sequencer to deployer if not specified
        sequencerAddress = vm.envOr("SEQUENCER_ADDRESS", deployer);

        console.log("=== Deployment Configuration ===");
        console.log("Deployer:", deployer);
        console.log("Deployer Balance:", deployer.balance);
        console.log("Initial State Root:", vm.toString(initialStateRoot));
        console.log("Finalization Delay:", finalizationDelay, "seconds");
        console.log("Sequencer:", sequencerAddress);
        console.log("");
    }

    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");

        vm.startBroadcast(deployerPrivateKey);

        // Step 1: Deploy StateManager
        console.log("1. Deploying StateManager...");
        stateManager = new StateManager(initialStateRoot);
        console.log("   StateManager deployed at:", address(stateManager));
        console.log("");

        // Step 2: Deploy MockVerifier (replace with actual Gnark verifier in production)
        console.log("2. Deploying MockVerifier...");
        console.log("   WARNING: Using MockVerifier - DO NOT USE IN PRODUCTION");
        mockVerifier = new MockVerifier();
        console.log("   MockVerifier deployed at:", address(mockVerifier));
        console.log("");

        // Step 3: Deploy BatchRegistry
        console.log("3. Deploying BatchRegistry...");
        batchRegistry =
            new BatchRegistry(address(mockVerifier), address(stateManager), sequencerAddress, finalizationDelay);
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
        console.log("MockVerifier:", address(mockVerifier));
        console.log("BatchRegistry:", address(batchRegistry));
        console.log("");
        console.log("IMPORTANT NEXT STEPS:");
        console.log("1. Replace MockVerifier with actual Gnark-generated verifier before mainnet");
        console.log("2. Transfer BatchRegistry ownership to a multisig");
        console.log("3. Run comprehensive test suite");
        console.log("4. Get security audit");
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
        require(address(batchRegistry.verifier()) == address(mockVerifier), "BatchRegistry verifier mismatch");
        require(batchRegistry.sequencer() == sequencerAddress, "Sequencer not set correctly");
        require(batchRegistry.finalizationDelay() == finalizationDelay, "Finalization delay mismatch");
        require(batchRegistry.nextBatchNumber() == 1, "Next batch number should be 1");
        require(!batchRegistry.paused(), "BatchRegistry should not be paused");
        console.log("  BatchRegistry: OK");

        // Verify MockVerifier
        require(mockVerifier.verificationResult(), "MockVerifier should return true by default");
        console.log("  MockVerifier: OK");

        console.log("All contracts verified successfully!");
    }

    function saveDeploymentAddresses() internal {
        string memory json = "deployment";

        vm.serializeAddress(json, "StateManager", address(stateManager));
        vm.serializeAddress(json, "MockVerifier", address(mockVerifier));
        vm.serializeAddress(json, "BatchRegistry", address(batchRegistry));
        vm.serializeAddress(json, "Sequencer", sequencerAddress);
        vm.serializeUint(json, "FinalizationDelay", finalizationDelay);
        string memory finalJson = vm.serializeBytes32(json, "InitialStateRoot", initialStateRoot);

        string memory outputDir = string.concat(vm.projectRoot(), "/deployments/");
        string memory chainId = vm.toString(block.chainid);
        string memory outputFile = string.concat(outputDir, chainId, ".json");

        vm.writeJson(finalJson, outputFile);
        console.log("");
        console.log("Deployment addresses saved to:", outputFile);
    }
}
