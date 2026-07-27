pragma solidity ^0.8.0;
contract Test {
    function test() public pure returns (bytes32) {
        uint256[] memory arr = new uint256[](120);
        return keccak256(abi.encodePacked(uint256(1), uint256(2), arr));
    }
}
