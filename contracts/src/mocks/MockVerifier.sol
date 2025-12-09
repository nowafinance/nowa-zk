// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import {IVerifier} from "../interfaces/IVerifier.sol";

/**
 * @title MockVerifier
 * @notice Mock verifier for testing purposes
 * @dev This contract implements the IVerifier interface but always returns true.
 *      DO NOT USE IN PRODUCTION. Replace with actual Gnark-generated verifier.
 *
 * @custom:security FOR TESTING ONLY - This bypasses all cryptographic verification
 */
contract MockVerifier is IVerifier {
    /// @notice Mock verification key hash
    bytes32 private constant MOCK_VK_HASH = keccak256("MOCK_VERIFICATION_KEY_v1");

    /// @notice Mock circuit size (1 million constraints)
    uint256 private constant MOCK_CIRCUIT_SIZE = 1_000_000;

    /// @notice Mock max batch size (1000 transactions)
    uint256 private constant MOCK_MAX_BATCH_SIZE = 1000;

    /// @notice Flag to control verification result (for testing)
    bool public verificationResult = true;

    /**
     * @notice Verifies a zero-knowledge proof (MOCK - always returns true)
     * @dev FOR TESTING ONLY. This does not perform actual cryptographic verification.
     *
     * @return valid Always returns verificationResult
     */
    function verifyProof(uint256[2] calldata, uint256[2][2] calldata, uint256[2] calldata, uint256[6] calldata)
        external
        view
        override
        returns (bool valid)
    {
        return verificationResult;
    }

    /**
     * @notice Returns the mock verification key hash
     * @return vkHash The mock verification key hash
     */
    function getVerificationKeyHash() external pure override returns (bytes32 vkHash) {
        return MOCK_VK_HASH;
    }

    /**
     * @notice Returns the mock circuit size
     * @return circuitSize The mock number of constraints
     */
    function getCircuitSize() external pure override returns (uint256 circuitSize) {
        return MOCK_CIRCUIT_SIZE;
    }

    /**
     * @notice Returns the mock maximum batch size
     * @return maxBatchSize The mock maximum number of transactions per batch
     */
    function getMaxBatchSize() external pure override returns (uint256 maxBatchSize) {
        return MOCK_MAX_BATCH_SIZE;
    }

    /**
     * @notice Sets the verification result for testing
     * @dev Allows tests to simulate proof verification failures
     * @param result The result to return from verifyProof
     */
    function setVerificationResult(bool result) external {
        verificationResult = result;
    }
}
