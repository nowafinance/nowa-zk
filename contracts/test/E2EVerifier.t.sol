// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "./data/TestVerifier.sol";

contract E2EVerifierTest is Test {
    Verifier public verifier;

    function setUp() public {
        verifier = new Verifier();
    }

    function test_E2E_Proof() public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, "/test/data/test_proof.json");
        string memory json = vm.readFile(path);

        string[] memory proofStr = vm.parseJsonStringArray(json, ".proof");
        uint256[8] memory proof;
        for (uint i = 0; i < 8; i++) {
            proof[i] = vm.parseUint(proofStr[i]);
        }

        string[] memory commitmentsStr = vm.parseJsonStringArray(json, ".commitments");
        uint256[2] memory commitments;
        for (uint i = 0; i < 2; i++) {
            commitments[i] = vm.parseUint(commitmentsStr[i]);
        }

        string[] memory commitmentPokStr = vm.parseJsonStringArray(json, ".commitmentPok");
        uint256[2] memory commitmentPok;
        for (uint i = 0; i < 2; i++) {
            commitmentPok[i] = vm.parseUint(commitmentPokStr[i]);
        }

        string[] memory pubInputsStr = vm.parseJsonStringArray(json, ".publicInputs");
        uint256[301] memory publicInputs;
        for (uint i = 0; i < 301; i++) {
            publicInputs[i] = vm.parseUint(pubInputsStr[i]);
        }

        verifier.verifyProof(proof, commitments, commitmentPok, publicInputs);
    }
}
