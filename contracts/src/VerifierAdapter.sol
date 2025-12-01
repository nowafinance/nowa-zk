// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {IVerifier} from "./interfaces/IVerifier.sol";
import {Verifier} from "./generated/RollupVerifier.sol";

/**
 * @title VerifierAdapter
 * @notice Adapts the Gnark-generated Verifier contract to the IVerifier interface.
 * @dev Gnark's generated verifier reverts on failure and has no return value.
 *      This adapter wraps it to return true/false as expected by IVerifier.
 */
contract VerifierAdapter is IVerifier {
    Verifier public immutable gnarkVerifier;

    constructor(address _gnarkVerifier) {
        require(_gnarkVerifier != address(0), "VerifierAdapter: zero address");
        gnarkVerifier = Verifier(_gnarkVerifier);
    }

    function verifyProof(
        uint256[2] calldata proofA,
        uint256[2][2] calldata proofB,
        uint256[2] calldata proofC,
        uint256[6] calldata publicInputs
    ) external view override returns (bool valid) {
        // Gnark verifier reverts on failure.
        // Since we can't use try/catch in a view function easily with external calls that might revert with custom errors,
        // and we want to maintain the view modifier, we have to rely on the caller handling the revert
        // OR we just let it revert.

        // However, IVerifier expects a boolean.
        // The standard pattern for these verifiers is often just to let them revert if invalid.
        // But BatchRegistry checks `require(verifier.verifyProof(...), "Invalid proof")`.

        // Actually, we can use staticcall in assembly to catch the revert in a view function.

        // Prepare calldata for verifyProof(uint256[8],uint256[6])
        // Note: Gnark verifier takes (uint256[8] proof, uint256[6] input)
        // We need to flatten proofA, proofB, proofC into uint256[8]

        uint256[8] memory proof;
        proof[0] = proofA[0];
        proof[1] = proofA[1];
        proof[2] = proofB[0][0];
        proof[3] = proofB[0][1];
        proof[4] = proofB[1][0];
        proof[5] = proofB[1][1];
        proof[6] = proofC[0];
        proof[7] = proofC[1];

        try gnarkVerifier.verifyProof(proof, publicInputs) {
            return true;
        } catch {
            return false;
        }
    }

    function getVerificationKeyHash() external pure override returns (bytes32) {
        return bytes32(0); // Not implemented
    }

    function getCircuitSize() external pure override returns (uint256) {
        return 0; // Not implemented
    }

    function getMaxBatchSize() external pure override returns (uint256) {
        return 0; // Not implemented
    }
}
