// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "../../src/TradeRegistry.sol";

contract MockTradeVerifier is ITradeVerifier {
    bool public shouldRevert = false;

    function setShouldRevert(bool _shouldRevert) external {
        shouldRevert = _shouldRevert;
    }

    function verifyProof(
        uint256[8] calldata /* proof */,
        uint256[2] calldata /* commitments */,
        uint256[2] calldata /* commitmentPok */,
        uint256[1] calldata /* input */
    ) external view override {
        if (shouldRevert) {
            revert("MockTradeVerifier: proof invalid");
        }
    }
}
