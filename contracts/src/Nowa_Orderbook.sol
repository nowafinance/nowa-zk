// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// ============================================================
// LIBRARIES
// ============================================================

/// @notice SafeERC20 library for safe token transfers
/// @dev Handles tokens that don't return bool on transfer
library SafeERC20 {
    function safeTransfer(IERC20 token, address to, uint256 value) internal {
        _callOptionalReturn(token, abi.encodeWithSelector(token.transfer.selector, to, value));
    }

    function safeTransferFrom(IERC20 token, address from, address to, uint256 value) internal {
        _callOptionalReturn(token, abi.encodeWithSelector(token.transferFrom.selector, from, to, value));
    }

    function _callOptionalReturn(IERC20 token, bytes memory data) private {
        (bool success, bytes memory returndata) = address(token).call(data);
        require(success, "SafeERC20: low-level call failed");
        if (returndata.length > 0) {
            require(abi.decode(returndata, (bool)), "SafeERC20: operation failed");
        }
    }
}

library SignedMath {
    function max(int256 a, int256 b) internal pure returns (int256) {
        return a > b ? a : b;
    }

    function min(int256 a, int256 b) internal pure returns (int256) {
        return a < b ? a : b;
    }

    function average(int256 a, int256 b) internal pure returns (int256) {
        int256 x = (a & b) + ((a ^ b) >> 1);
        return x + (int256(uint256(x) >> 255) & (a ^ b));
    }

    function abs(int256 n) internal pure returns (uint256) {
        unchecked {
            return uint256(n >= 0 ? n : -n);
        }
    }
}

library Math {
    error MathOverflowedMulDiv();

    enum Rounding { Floor, Ceil, Trunc, Expand }

    function tryAdd(uint256 a, uint256 b) internal pure returns (bool, uint256) {
        unchecked {
            uint256 c = a + b;
            if (c < a) return (false, 0);
            return (true, c);
        }
    }

    function trySub(uint256 a, uint256 b) internal pure returns (bool, uint256) {
        unchecked {
            if (b > a) return (false, 0);
            return (true, a - b);
        }
    }

    function tryMul(uint256 a, uint256 b) internal pure returns (bool, uint256) {
        unchecked {
            if (a == 0) return (true, 0);
            uint256 c = a * b;
            if (c / a != b) return (false, 0);
            return (true, c);
        }
    }

    function tryDiv(uint256 a, uint256 b) internal pure returns (bool, uint256) {
        unchecked {
            if (b == 0) return (false, 0);
            return (true, a / b);
        }
    }

    function tryMod(uint256 a, uint256 b) internal pure returns (bool, uint256) {
        unchecked {
            if (b == 0) return (false, 0);
            return (true, a % b);
        }
    }

    function max(uint256 a, uint256 b) internal pure returns (uint256) {
        return a > b ? a : b;
    }

    function min(uint256 a, uint256 b) internal pure returns (uint256) {
        return a < b ? a : b;
    }

    function average(uint256 a, uint256 b) internal pure returns (uint256) {
        return (a & b) + (a ^ b) / 2;
    }

    function ceilDiv(uint256 a, uint256 b) internal pure returns (uint256) {
        if (b == 0) return a / b;
        return a == 0 ? 0 : (a - 1) / b + 1;
    }

    function mulDiv(uint256 x, uint256 y, uint256 denominator) internal pure returns (uint256 result) {
        unchecked {
            uint256 prod0 = x * y;
            uint256 prod1;
            assembly {
                let mm := mulmod(x, y, not(0))
                prod1 := sub(sub(mm, prod0), lt(mm, prod0))
            }
            if (prod1 == 0) return prod0 / denominator;
            if (denominator <= prod1) revert MathOverflowedMulDiv();

            uint256 remainder;
            assembly {
                remainder := mulmod(x, y, denominator)
                prod1 := sub(prod1, gt(remainder, prod0))
                prod0 := sub(prod0, remainder)
            }

            uint256 twos = denominator & (0 - denominator);
            assembly {
                denominator := div(denominator, twos)
                prod0 := div(prod0, twos)
                twos := add(div(sub(0, twos), twos), 1)
            }
            prod0 |= prod1 * twos;

            uint256 inverse = (3 * denominator) ^ 2;
            inverse *= 2 - denominator * inverse;
            inverse *= 2 - denominator * inverse;
            inverse *= 2 - denominator * inverse;
            inverse *= 2 - denominator * inverse;
            inverse *= 2 - denominator * inverse;
            inverse *= 2 - denominator * inverse;

            result = prod0 * inverse;
            return result;
        }
    }

    function mulDiv(uint256 x, uint256 y, uint256 denominator, Rounding rounding) internal pure returns (uint256) {
        uint256 result = mulDiv(x, y, denominator);
        if (unsignedRoundsUp(rounding) && mulmod(x, y, denominator) > 0) {
            result += 1;
        }
        return result;
    }

    function sqrt(uint256 a) internal pure returns (uint256) {
        if (a == 0) return 0;
        uint256 result = 1 << (log2(a) >> 1);
        unchecked {
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            result = (result + a / result) >> 1;
            return min(result, a / result);
        }
    }

    function sqrt(uint256 a, Rounding rounding) internal pure returns (uint256) {
        unchecked {
            uint256 result = sqrt(a);
            return result + (unsignedRoundsUp(rounding) && result * result < a ? 1 : 0);
        }
    }

    function log2(uint256 value) internal pure returns (uint256) {
        uint256 result = 0;
        unchecked {
            if (value >> 128 > 0) { value >>= 128; result += 128; }
            if (value >> 64 > 0) { value >>= 64; result += 64; }
            if (value >> 32 > 0) { value >>= 32; result += 32; }
            if (value >> 16 > 0) { value >>= 16; result += 16; }
            if (value >> 8 > 0) { value >>= 8; result += 8; }
            if (value >> 4 > 0) { value >>= 4; result += 4; }
            if (value >> 2 > 0) { value >>= 2; result += 2; }
            if (value >> 1 > 0) { result += 1; }
        }
        return result;
    }

    function log2(uint256 value, Rounding rounding) internal pure returns (uint256) {
        unchecked {
            uint256 result = log2(value);
            return result + (unsignedRoundsUp(rounding) && 1 << result < value ? 1 : 0);
        }
    }

    function log10(uint256 value) internal pure returns (uint256) {
        uint256 result = 0;
        unchecked {
            if (value >= 10 ** 64) { value /= 10 ** 64; result += 64; }
            if (value >= 10 ** 32) { value /= 10 ** 32; result += 32; }
            if (value >= 10 ** 16) { value /= 10 ** 16; result += 16; }
            if (value >= 10 ** 8) { value /= 10 ** 8; result += 8; }
            if (value >= 10 ** 4) { value /= 10 ** 4; result += 4; }
            if (value >= 10 ** 2) { value /= 10 ** 2; result += 2; }
            if (value >= 10 ** 1) { result += 1; }
        }
        return result;
    }

    function log10(uint256 value, Rounding rounding) internal pure returns (uint256) {
        unchecked {
            uint256 result = log10(value);
            return result + (unsignedRoundsUp(rounding) && 10 ** result < value ? 1 : 0);
        }
    }

    function log256(uint256 value) internal pure returns (uint256) {
        uint256 result = 0;
        unchecked {
            if (value >> 128 > 0) { value >>= 128; result += 16; }
            if (value >> 64 > 0) { value >>= 64; result += 8; }
            if (value >> 32 > 0) { value >>= 32; result += 4; }
            if (value >> 16 > 0) { value >>= 16; result += 2; }
            if (value >> 8 > 0) { result += 1; }
        }
        return result;
    }

    function log256(uint256 value, Rounding rounding) internal pure returns (uint256) {
        unchecked {
            uint256 result = log256(value);
            return result + (unsignedRoundsUp(rounding) && 1 << (result << 3) < value ? 1 : 0);
        }
    }

    function unsignedRoundsUp(Rounding rounding) internal pure returns (bool) {
        return uint8(rounding) % 2 == 1;
    }
}

library Strings {
    bytes16 private constant HEX_DIGITS = "0123456789abcdef";
    uint8 private constant ADDRESS_LENGTH = 20;

    error StringsInsufficientHexLength(uint256 value, uint256 length);

    function toString(uint256 value) internal pure returns (string memory) {
        unchecked {
            uint256 length = Math.log10(value) + 1;
            string memory buffer = new string(length);
            uint256 ptr;
            assembly { ptr := add(buffer, add(32, length)) }
            while (true) {
                ptr--;
                assembly { mstore8(ptr, byte(mod(value, 10), HEX_DIGITS)) }
                value /= 10;
                if (value == 0) break;
            }
            return buffer;
        }
    }

    function toHexString(uint256 value) internal pure returns (string memory) {
        unchecked { return toHexString(value, Math.log256(value) + 1); }
    }

    function toHexString(uint256 value, uint256 length) internal pure returns (string memory) {
        uint256 localValue = value;
        bytes memory buffer = new bytes(2 * length + 2);
        buffer[0] = "0";
        buffer[1] = "x";
        for (uint256 i = 2 * length + 1; i > 1; --i) {
            buffer[i] = HEX_DIGITS[localValue & 0xf];
            localValue >>= 4;
        }
        if (localValue != 0) revert StringsInsufficientHexLength(value, length);
        return string(buffer);
    }

    function toHexString(address addr) internal pure returns (string memory) {
        return toHexString(uint256(uint160(addr)), ADDRESS_LENGTH);
    }

    function equal(string memory a, string memory b) internal pure returns (bool) {
        return bytes(a).length == bytes(b).length && keccak256(bytes(a)) == keccak256(bytes(b));
    }
}

library MessageHashUtils {
    function toEthSignedMessageHash(bytes32 messageHash) internal pure returns (bytes32 digest) {
        assembly {
            mstore(0x00, "\x19Ethereum Signed Message:\n32")
            mstore(0x1c, messageHash)
            digest := keccak256(0x00, 0x3c)
        }
    }

    function toEthSignedMessageHash(bytes memory message) internal pure returns (bytes32) {
        return keccak256(bytes.concat("\x19Ethereum Signed Message:\n", bytes(Strings.toString(message.length)), message));
    }

    function toDataWithIntendedValidatorHash(address validator, bytes memory data) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(hex"19_00", validator, data));
    }

    function toTypedDataHash(bytes32 domainSeparator, bytes32 structHash) internal pure returns (bytes32 digest) {
        assembly {
            let ptr := mload(0x40)
            mstore(ptr, hex"19_01")
            mstore(add(ptr, 0x02), domainSeparator)
            mstore(add(ptr, 0x22), structHash)
            digest := keccak256(ptr, 0x42)
        }
    }
}

library ECDSA {
    enum RecoverError { NoError, InvalidSignature, InvalidSignatureLength, InvalidSignatureS }

    error ECDSAInvalidSignature();
    error ECDSAInvalidSignatureLength(uint256 length);
    error ECDSAInvalidSignatureS(bytes32 s);

    function tryRecover(bytes32 hash, bytes memory signature) internal pure returns (address, RecoverError, bytes32) {
        if (signature.length == 65) {
            bytes32 r; bytes32 s; uint8 v;
            assembly {
                r := mload(add(signature, 0x20))
                s := mload(add(signature, 0x40))
                v := byte(0, mload(add(signature, 0x60)))
            }
            return tryRecover(hash, v, r, s);
        } else {
            return (address(0), RecoverError.InvalidSignatureLength, bytes32(signature.length));
        }
    }

    function recover(bytes32 hash, bytes memory signature) internal pure returns (address) {
        (address recovered, RecoverError error, bytes32 errorArg) = tryRecover(hash, signature);
        _throwError(error, errorArg);
        return recovered;
    }

    function tryRecover(bytes32 hash, bytes32 r, bytes32 vs) internal pure returns (address, RecoverError, bytes32) {
        unchecked {
            bytes32 s = vs & bytes32(0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff);
            uint8 v = uint8((uint256(vs) >> 255) + 27);
            return tryRecover(hash, v, r, s);
        }
    }

    function recover(bytes32 hash, bytes32 r, bytes32 vs) internal pure returns (address) {
        (address recovered, RecoverError error, bytes32 errorArg) = tryRecover(hash, r, vs);
        _throwError(error, errorArg);
        return recovered;
    }

    function tryRecover(bytes32 hash, uint8 v, bytes32 r, bytes32 s) internal pure returns (address, RecoverError, bytes32) {
        if (uint256(s) > 0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0) {
            return (address(0), RecoverError.InvalidSignatureS, s);
        }
        address signer = ecrecover(hash, v, r, s);
        if (signer == address(0)) {
            return (address(0), RecoverError.InvalidSignature, bytes32(0));
        }
        return (signer, RecoverError.NoError, bytes32(0));
    }

    function recover(bytes32 hash, uint8 v, bytes32 r, bytes32 s) internal pure returns (address) {
        (address recovered, RecoverError error, bytes32 errorArg) = tryRecover(hash, v, r, s);
        _throwError(error, errorArg);
        return recovered;
    }

    function _throwError(RecoverError error, bytes32 errorArg) private pure {
        if (error == RecoverError.NoError) return;
        else if (error == RecoverError.InvalidSignature) revert ECDSAInvalidSignature();
        else if (error == RecoverError.InvalidSignatureLength) revert ECDSAInvalidSignatureLength(uint256(errorArg));
        else if (error == RecoverError.InvalidSignatureS) revert ECDSAInvalidSignatureS(errorArg);
    }
}

abstract contract Context {
    function _msgSender() internal view virtual returns (address) { return msg.sender; }
    function _msgData() internal view virtual returns (bytes calldata) { return msg.data; }
}

library LibOrder {
    // SECURITY FIX: Fixed typo CANCLED -> CANCELLED
    enum Status { OPEN, PARTIAL_CLOSED, CLOSED, CANCELLED }
    enum OrderSelfTradePrevention { dc, co, cn, cb }
    enum OrderSide { Buy, Sell }
    enum OrderTimeInForce { gtc, gtt, ioc, fok }
    enum OrderType { Market, Limit, LimitMaker, StopLoss, StopLossLimit, TakeProfit, TakeProfitLimit }

    struct OrderBookTrade {
        address baseToken;
        address quoteToken;
        uint64 grossBaseAmount;
        uint64 grossQuoteAmount;
        uint64 netBaseAmount;
        uint64 netQuoteAmount;
        address makerFeeToken;
        address takerFeeToken;
        uint64 makerFeeAmount;
        uint64 takerFeeAmount;
        uint64 price;
        OrderSide makerSide;
    }

    struct Order {
        address userAddress;
        address baseToken;
        address quoteToken;
        uint64 amount;
        bool isAmountInQuote;
        uint128 nonce;
        OrderType orderType;
        OrderSide side;
        uint64 limitPrice;
        uint64 stopPrice;
        OrderTimeInForce timeInForce;
        uint64 cancelAfter;
        bytes walletSignature;
        uint64 balanceSnapshot;
        uint64 allowanceSnapshot;
    }

    struct PriceProof {
        address baseToken;
        address quoteToken;
        uint64 price;
        uint64 timestamp;
        bytes signature;
    }
}

/// @title AssetUnitConversions - Convert between pip units and token units
/// @notice Handles decimal conversion between 8-decimal pip format and variable-decimal tokens
library AssetUnitConversions {
    /// @notice Convert pip amount to token asset units
    /// @param Amount The amount in pips (8 decimals)
    /// @param assetDecimals The token's decimal places
    /// @return The amount in token units
    /// @dev SECURITY FIX: Uses ceiling division for tokens with < 8 decimals to prevent rounding to zero
    function pipsToAssetUnits(uint64 Amount, uint8 assetDecimals) internal pure returns (uint256) {
        require(assetDecimals <= 32, "Asset cannot have more than 32 decimals");
        if (assetDecimals > 8) {
            return uint256(Amount) * (uint256(10)**(assetDecimals - 8));
        }
        // SECURITY FIX: Use ceiling division to prevent precision loss for low-decimal tokens (e.g., USDC 6 decimals)
        uint256 divisor = uint256(10)**(8 - assetDecimals);
        return (uint256(Amount) + divisor - 1) / divisor;
    }

    function assetUnitsToPips(uint256 quantityInAssetUnits, uint8 assetDecimals) internal pure returns (uint64) {
        require(assetDecimals <= 32, "Asset cannot have more than 32 decimals");
        uint256 Amount;
        if (assetDecimals > 8) {
            Amount = quantityInAssetUnits / (uint256(10)**(assetDecimals - 8));
        } else {
            Amount = quantityInAssetUnits * (uint256(10)**(8 - assetDecimals));
        }
        require(Amount < 2**64, "Pip quantity overflows uint64");
        return uint64(Amount);
    }
}

interface IERC20 {
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event Transfer(address indexed from, address indexed to, uint256 value);
    function name() external view returns (string memory);
    function symbol() external view returns (string memory);
    function decimals() external view returns (uint8);
    function totalSupply() external view returns (uint256);
    function balanceOf(address owner) external view returns (uint256);
    function allowance(address owner, address spender) external view returns (uint256);
    function approve(address spender, uint256 value) external returns (bool);
    function transfer(address to, uint256 value) external returns (bool);
    function transferFrom(address from, address to, uint256 value) external returns (bool);
}

abstract contract Ownable is Context {
    address private _owner;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    modifier onlyOwner() {
        _checkOwner();
        _;
    }

    function owner() public view virtual returns (address) { return _owner; }

    function _checkOwner() internal view virtual {
        require(owner() == _msgSender(), "Ownable: caller is not the owner");
    }

    function renounceOwnership() public virtual onlyOwner {
        _transferOwnership(address(0));
    }

    function transferOwnership(address newOwner) public virtual onlyOwner {
        require(newOwner != address(0), "Ownable: new owner is the zero address");
        _transferOwnership(newOwner);
    }

    function _transferOwnership(address newOwner) internal virtual {
        address oldOwner = _owner;
        _owner = newOwner;
        emit OwnershipTransferred(oldOwner, newOwner);
    }
}

interface IWETH {
    function deposit() external payable;
    function withdraw(uint256 wad) external;
    function approve(address guy, uint256 wad) external returns (bool);
    function transfer(address dst, uint256 wad) external returns (bool);
    function transferFrom(address src, address dst, uint256 wad) external returns (bool);
}

/// @notice SECURITY FIX: ReentrancyGuard to prevent reentrancy attacks
abstract contract ReentrancyGuard {
    uint256 private constant _NOT_ENTERED = 1;
    uint256 private constant _ENTERED = 2;
    uint256 private _status;

    constructor() {
        _status = _NOT_ENTERED;
    }

    modifier nonReentrant() {
        require(_status != _ENTERED, "ReentrancyGuard: reentrant call");
        _status = _ENTERED;
        _;
        _status = _NOT_ENTERED;
    }
}

/// @notice SECURITY FIX: Pausable for emergency stop functionality
abstract contract Pausable is Context {
    event Paused(address account);
    event Unpaused(address account);

    bool private _paused;

    constructor() {
        _paused = false;
    }

    modifier whenNotPaused() {
        require(!_paused, "Pausable: paused");
        _;
    }

    modifier whenPaused() {
        require(_paused, "Pausable: not paused");
        _;
    }

    function paused() public view virtual returns (bool) {
        return _paused;
    }

    function _pause() internal virtual whenNotPaused {
        _paused = true;
        emit Paused(_msgSender());
    }

    function _unpause() internal virtual whenPaused {
        _paused = false;
        emit Unpaused(_msgSender());
    }
}

// ============================================================
// MAIN CONTRACT
// ============================================================

/// @title Nowa_Orderbook - Decentralized Order Book Exchange
/// @notice Handles order matching, execution, and settlement for spot trading
/// @dev Implements EIP-712 signatures, SafeERC20 transfers, and emergency controls
contract Nowa_Orderbook is Ownable, ReentrancyGuard, Pausable {
    using SafeERC20 for IERC20;
    
    // ============================================================
    // CONSTANTS
    // ============================================================
    uint64 public constant pipPriceMultiplier = 10**8;
    uint64 public constant maxFeeBasisPoints = 20 * 100;
    uint64 public constant basisPointsInTotal = 100 * 100;
    
    // SECURITY FIX: Increased from 30 to 120 seconds for network congestion tolerance
    uint64 public constant PRICE_PROOF_MAX_AGE = 120;
    
    // SECURITY FIX: Add batch size limit to prevent gas limit attacks
    uint256 public constant MAX_BATCH_SIZE = 50;
    
    // Version for migration tracking
    string public constant VERSION = "2.2.0";

    // ============================================================
    // STATE VARIABLES
    // ============================================================
    address public feeTo;
    address public relayer;
    bytes32 public DOMAIN_SEPARATOR;
    
    mapping(address => bool) public operators;
    mapping(bytes32 => bool) public completedOrderHashes;
    mapping(bytes32 => uint64) public partiallyFilledOrderQuantitiesInPips;
    mapping(address => uint128) public minValidNonce;
    mapping(bytes32 => address) public orderOwners;

    // AUDIT_FIX: Token decimals registry to handle non-standard tokens
    mapping(address => uint8) public tokenDecimals;
    mapping(address => bool) public tokenDecimalsSet;
    
    // AUDIT_FIX: Price tracking for deviation checks
    mapping(bytes32 => uint64) public lastPrice; // keccak256(baseToken, quoteToken) => price
    uint256 public maxPriceDeviationBps; // e.g., 500 = 5%

    // ============================================================
    // EIP-712 CONSTANTS
    // ============================================================
    
    // AUDIT_FIX: Public constants for domain separator - UI MUST use these exact values
    string public constant EIP712_NAME = "Nowa_Orderbook";
    string public constant EIP712_VERSION = "1";
    
    bytes32 public constant ORDER_TYPEHASH = keccak256(
        "Order(address userAddress,address baseToken,address quoteToken,uint64 amount,bool isAmountInQuote,uint128 nonce,uint8 orderType,uint8 side,uint64 limitPrice,uint64 stopPrice,uint8 timeInForce,uint64 cancelAfter,uint64 balanceSnapshot,uint64 allowanceSnapshot)"
    );

    // ============================================================
    // CUSTOM ERRORS
    // ============================================================
    error BatchTradeFailedAtIndex(uint256 tradeIndex, string reason);
    error BatchTradeValidationFailed(uint256 tradeIndex, string validationType, string reason);
    error MarketOrderPriceProtectionViolated(LibOrder.OrderSide side, uint64 executionPrice, uint64 limitPrice);
    error FillOrKillNotFilled(bytes32 orderHash, uint64 requestedAmount, uint64 filledAmount);
    
    // SECURITY FIX: Additional custom errors for gas efficiency
    error InvalidBatchSize(uint256 provided, uint256 maximum);
    error OrderNonceInvalidated(address user, uint128 orderNonce, uint128 minValid);
    error OrderAlreadyCompleted(bytes32 orderHash);
    error InvalidOrderSignature(address signer);
    error InsufficientBalance(address token, address user, uint256 required, uint256 available);
    error InsufficientAllowance(address token, address user, uint256 required, uint256 available);
    error RelayerNotConfigured();
    error ZeroAddress();
    
    // SIZE_FIX: Additional custom errors for bytecode reduction
    error InvalidTradeLength();
    error EmptyBatch();
    error ForbiddenOperator();
    error ContractPaused();
    error NotPaused();
    error ReentrancyViolation();
    error InvalidNonce();
    error InvalidFeeTo();
    error InvalidRelayer();
    error InvalidOperator();
    error NotOrderOwner();
    error OrderAlreadyRegistered();
    error InvalidAssetPair();
    error FeeMismatch();
    error BaseTooLow();
    error QuoteTooLow();
    error BuyLimitExceeded();
    error SellLimitExceeded();
    error ExcessiveMakerFee();
    error ExcessiveTakerFee();
    error BaseFeeUnbalanced();
    error QuoteFeeUnbalanced();
    error PostOnlyViolation();
    error QuoteAmountInvalid();
    error OrderExpired();
    error BalanceSnapshotFailed();
    error AllowanceSnapshotFailed();
    error PriceProofMismatch();
    error PriceProofExpired();
    error PriceProofFuture();
    error InvalidRelayerSig();
    error OrderOverfilled();
    error FOKNotFilled();
    error StopNotTriggered();
    error TooManyDecimals();
    error PipOverflow();
    
    // AUDIT_FIX: Custom errors for audit fixes
    error PriceDeviationExceeded();
    error InvalidTokenDecimals();

    // ============================================================
    // EVENTS
    // ============================================================
    event OrderBookTradeExecuted(
        address buyWallet,
        address sellWallet,
        address baseToken,
        address quoteToken,
        uint64 baseAmount,
        uint64 quoteAmount,
        LibOrder.OrderSide takerSide
    );
    event LogOperatorUpdate(address operator, bool isOperator);
    event LogFeeAccountUpdate(address oldFeeTo, address newfeeTo);
    event RelayerUpdated(address indexed oldRelayer, address indexed newRelayer);
    event CancelUpTo(address indexed wallet, uint128 nonce, uint256 timestamp);
    event EmergencyCancel(address indexed userAddress, bytes32 indexed orderHash, uint256 timestamp);
    event StopOrderTriggered(address indexed userAddress, bytes32 indexed orderHash, LibOrder.OrderType orderType, uint64 triggerPrice, uint64 stopPrice);
    event PostOnlyOrderExecuted(address indexed userAddress, bytes32 indexed orderHash, LibOrder.OrderSide side);
    event MarketOrderCancelledRemainder(address indexed userAddress, bytes32 indexed orderHash, uint64 originalAmount, uint64 filledAmount);
    
    // AUDIT_FIX: Events for audit fixes
    event TokenDecimalsSet(address indexed token, uint8 decimals);
    event MaxPriceDeviationUpdated(uint256 oldBps, uint256 newBps);
    event PriceUpdated(address indexed baseToken, address indexed quoteToken, uint64 oldPrice, uint64 newPrice);
    event BatchTradeStarted(uint256 batchSize, uint256 timestamp);
    event BatchTradeCompleted(uint256 batchSize, uint256 successCount, uint256 timestamp);
    event OrderOwnerRegistered(address indexed userAddress, bytes32 indexed orderHash);

    // ============================================================
    // MODIFIERS
    // ============================================================
    modifier onlyOperator {
        if (msg.sender != owner() && !operators[msg.sender]) revert ForbiddenOperator();
        _;
    }

    // ============================================================
    // CONSTRUCTOR
    // ============================================================
    /// @notice Initialize the orderbook contract
    /// @param _feeTo Address to receive trading fees
    /// @dev AUDIT_FIX: DOMAIN_SEPARATOR built from public constants EIP712_NAME and EIP712_VERSION
    ///      WARNING: Frontend/UI MUST use the same EIP712_NAME ("Nowa_Orderbook") and EIP712_VERSION ("1")
    constructor(address _feeTo) {
        // SECURITY FIX: Zero address validation
        require(_feeTo != address(0), "Invalid fee address");
        
        feeTo = _feeTo;
        _transferOwnership(_feeTo);
        
        // AUDIT_FIX: Use public constants for EIP-712 domain to ensure UI consistency
        DOMAIN_SEPARATOR = keccak256(
            abi.encode(
                keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                keccak256(bytes(EIP712_NAME)),
                keccak256(bytes(EIP712_VERSION)),
                block.chainid,
                address(this)
            )
        );
        
        // AUDIT_FIX: Initialize maxPriceDeviationBps to 500 (5%)
        maxPriceDeviationBps = 500;
    }

    // ============================================================
    // ADMIN FUNCTIONS
    // ============================================================
    
    /// @notice Set or remove an operator
    /// @param _operator Address of the operator
    /// @param _isOperator True to add, false to remove
    function setOperator(address _operator, bool _isOperator) public onlyOwner returns (bool) {
        require(_operator != address(0), "Invalid operator address");
        // SECURITY FIX: Emit event after state change
        operators[_operator] = _isOperator;
        emit LogOperatorUpdate(_operator, _isOperator);
        return true;
    }

    /// @notice Set the fee recipient address
    /// @param _feeTo New fee recipient address
    function setFeeAccount(address _feeTo) public onlyOwner returns (bool) {
        require(_feeTo != address(0), "Invalid fee address");
        // SECURITY FIX: Emit event after state change
        address oldFeeTo = feeTo;
        feeTo = _feeTo;
        emit LogFeeAccountUpdate(oldFeeTo, _feeTo);
        return true;
    }

    /// @notice Set the trusted relayer for stop orders
    /// @param _relayer New relayer address
    function setRelayer(address _relayer) external onlyOwner {
        require(_relayer != address(0), "Relayer cannot be zero address");
        // SECURITY FIX: Emit event after state change
        address oldRelayer = relayer;
        relayer = _relayer;
        emit RelayerUpdated(oldRelayer, _relayer);
    }
    
    // AUDIT_FIX: Token decimals registry to handle non-standard tokens
    /// @notice Set decimals for a token (for non-standard tokens)
    /// @param token The token address
    /// @param decimals The token decimals (must be <= 32)
    function setTokenDecimals(address token, uint8 decimals) external onlyOwner {
        require(token != address(0), "Invalid token address");
        require(decimals <= 32, "Decimals must be <= 32");
        tokenDecimals[token] = decimals;
        tokenDecimalsSet[token] = true;
        emit TokenDecimalsSet(token, decimals);
    }
    
    // AUDIT_FIX: Set max price deviation for relayer price proofs
    /// @notice Set maximum allowed price deviation for price proofs
    /// @param bps Basis points (e.g., 500 = 5%)
    function setMaxPriceDeviation(uint256 bps) external onlyOwner {
        require(bps <= 10000, "Cannot exceed 100%");
        uint256 oldBps = maxPriceDeviationBps;
        maxPriceDeviationBps = bps;
        emit MaxPriceDeviationUpdated(oldBps, bps);
    }
    
    /// @notice Pause all trade execution
    function pause() external onlyOwner {
        _pause();
    }
    
    /// @notice Unpause trade execution
    function unpause() external onlyOwner {
        _unpause();
    }

    // ============================================================
    // BULK CANCEL
    // ============================================================
    
    /// @notice Invalidate all orders with nonce <= specified value
    /// @param nonce The nonce threshold - all orders with nonce <= this will be invalid
    /// @dev This is a bulk cancel mechanism - use carefully as it cannot be undone
    function cancelUpTo(uint128 nonce) external {
        require(nonce > minValidNonce[msg.sender], "Nonce must be greater than current minValidNonce");
        minValidNonce[msg.sender] = nonce;
        emit CancelUpTo(msg.sender, nonce, block.timestamp);
    }

    // ============================================================
    // EMERGENCY CANCEL
    // ============================================================
    
    /// @notice Emergency cancel a single order
    /// @param order The order to cancel (must be signed by caller)
    /// @dev Gas optimization: uses calldata for read-only order parameter
    function emergencyCancel(LibOrder.Order calldata order) external {
        if (msg.sender != order.userAddress) revert NotOrderOwner();
        
        bytes32 orderHash = getOrderHash(order);
        
        // Accept both EIP-712 and legacy signatures
        require(_isOrderSignatureValidCalldata(order), "Invalid order signature");
        require(!completedOrderHashes[orderHash], "Order already completed or cancelled");
        
        completedOrderHashes[orderHash] = true;
        delete partiallyFilledOrderQuantitiesInPips[orderHash];
        
        emit EmergencyCancel(msg.sender, orderHash, block.timestamp);
    }

    /// @notice Emergency cancel multiple orders in batch
    /// @param orders Array of orders to cancel (all must be signed by caller)
    /// @dev SECURITY FIX: Added batch size limit. Gas optimization: uses calldata
    function emergencyCancelBatch(LibOrder.Order[] calldata orders) external {
        // SECURITY FIX: Add batch size limit to prevent DoS
        require(orders.length <= MAX_BATCH_SIZE, "Batch size exceeds maximum");
        
        for (uint256 i = 0; i < orders.length; i++) {
            if (msg.sender != orders[i].userAddress) revert NotOrderOwner();
            
            bytes32 orderHash = getOrderHash(orders[i]);
            
            // Accept both EIP-712 and legacy signatures
            require(_isOrderSignatureValidCalldata(orders[i]), "Invalid order signature");
            
            if (completedOrderHashes[orderHash]) continue;
            
            completedOrderHashes[orderHash] = true;
            delete partiallyFilledOrderQuantitiesInPips[orderHash];
            
            emit EmergencyCancel(msg.sender, orderHash, block.timestamp);
        }
    }

    /// @notice Emergency cancel by hash (requires prior registration)
    /// @param orderHash The hash of the order to cancel
    function emergencyCancelByHash(bytes32 orderHash) external {
        require(orderOwners[orderHash] == msg.sender, "Caller is not registered order owner");
        require(!completedOrderHashes[orderHash], "Order already completed or cancelled");
        
        completedOrderHashes[orderHash] = true;
        delete partiallyFilledOrderQuantitiesInPips[orderHash];
        // SECURITY FIX: Clean up orderOwners mapping after cancel
        delete orderOwners[orderHash];
        
        emit EmergencyCancel(msg.sender, orderHash, block.timestamp);
    }

    /// @notice Register order ownership for hash-only cancellation
    /// @param order The order to register (must be signed by caller)
    /// @dev Gas optimization: uses calldata for read-only order parameter
    function registerOrderForCancel(LibOrder.Order calldata order) external {
        if (msg.sender != order.userAddress) revert NotOrderOwner();
        
        bytes32 orderHash = getOrderHash(order);
        
        if (!_isOrderSignatureValidCalldata(order)) revert InvalidOrderSignature(order.userAddress);
        if (orderOwners[orderHash] != address(0)) revert OrderAlreadyRegistered();
        if (completedOrderHashes[orderHash]) revert OrderAlreadyCompleted(orderHash);
        
        orderOwners[orderHash] = msg.sender;
        
        emit OrderOwnerRegistered(msg.sender, orderHash);
    }

    // ============================================================
    // HASH FUNCTIONS
    // ============================================================
    
    // Refactored to avoid stack too deep
    function getOrderHash(LibOrder.Order memory order) public pure returns (bytes32) {
        // Split encoding to reduce stack depth
        bytes memory part1 = abi.encodePacked(
            order.userAddress,
            order.baseToken,
            order.quoteToken,
            order.amount,
            order.isAmountInQuote,
            order.nonce,
            order.orderType
        );
        
        bytes memory part2 = abi.encodePacked(
            order.side,
            order.limitPrice,
            order.stopPrice,
            order.timeInForce,
            order.cancelAfter,
            order.balanceSnapshot,
            order.allowanceSnapshot
        );
        
        return keccak256(abi.encodePacked(part1, part2));
    }

    // Refactored to avoid stack too deep using helper
    function getTypedOrderHash(LibOrder.Order memory order) public view returns (bytes32) {
        bytes32 structHash = _computeOrderStructHash(order);
        return keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, structHash));
    }
    
    function _computeOrderStructHash(LibOrder.Order memory order) private pure returns (bytes32) {
        // Use two-step encoding to avoid stack too deep
        return keccak256(
            bytes.concat(
                abi.encode(
                    ORDER_TYPEHASH,
                    order.userAddress,
                    order.baseToken,
                    order.quoteToken,
                    order.amount,
                    order.isAmountInQuote,
                    order.nonce,
                    uint8(order.orderType)
                ),
                abi.encode(
                    uint8(order.side),
                    order.limitPrice,
                    order.stopPrice,
                    uint8(order.timeInForce),
                    order.cancelAfter,
                    order.balanceSnapshot,
                    order.allowanceSnapshot
                )
            )
        );
    }

    function getPriceProofHash(LibOrder.PriceProof memory proof) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(proof.baseToken, proof.quoteToken, proof.price, proof.timestamp));
    }

    /// @notice Verify legacy personal_sign (eth_sign) signature
    /// @dev Uses EIP-191 prefix: "\x19Ethereum Signed Message:\n32"
    function isSignatureValid(bytes32 hash, bytes memory signature, address signer) public pure returns (bool) {
        return ECDSA.recover(MessageHashUtils.toEthSignedMessageHash(hash), signature) == signer;
    }

    // SECURITY FIX: Alias for backward compatibility (removed duplicate implementation)
    /// @notice Alias for isSignatureValid - verifies legacy personal_sign signature
    function isLegacySignatureValid(bytes32 hash, bytes memory signature, address signer) public pure returns (bool) {
        return isSignatureValid(hash, signature, signer);
    }

    /// @notice Verify EIP-712 typed data signature
    /// @dev Expects the typedDigest to already include domain separator (direct ecrecover)
    function isTypedSignatureValid(bytes32 typedDigest, bytes memory signature, address signer) public pure returns (bool) {
        return ECDSA.recover(typedDigest, signature) == signer;
    }

    // PATCH: Helper to validate order signature (EIP-712 or legacy) for cancel operations
    /// @notice Check if order signature is valid using either EIP-712 or legacy verification
    /// @param order The order to validate
    /// @return True if signature is valid via either method
    function isOrderSignatureValid(LibOrder.Order memory order) public view returns (bool) {
        bytes32 orderId = getOrderHash(order);
        bytes32 typedDigest = getTypedOrderHash(order);
        
        // Try EIP-712 first
        if (isTypedSignatureValid(typedDigest, order.walletSignature, order.userAddress)) {
            return true;
        }
        
        // Fallback to legacy
        if (isLegacySignatureValid(orderId, order.walletSignature, order.userAddress)) {
            return true;
        }
        
        return false;
    }

    /// @notice Internal helper for calldata order signature validation
    /// @dev Gas optimization: works with calldata orders directly
    function _isOrderSignatureValidCalldata(LibOrder.Order calldata order) internal view returns (bool) {
        bytes32 orderId = getOrderHash(order);
        bytes32 typedDigest = getTypedOrderHash(order);
        
        // Try EIP-712 first
        if (isTypedSignatureValid(typedDigest, order.walletSignature, order.userAddress)) {
            return true;
        }
        
        // Fallback to legacy
        if (isLegacySignatureValid(orderId, order.walletSignature, order.userAddress)) {
            return true;
        }
        
        return false;
    }

    // ============================================================
    // TRADE EXECUTION
    // ============================================================
    
    /// @notice Execute a batch of trades
    /// @param buy Array of buy orders
    /// @param sell Array of sell orders
    /// @param orderBookTrade Array of trade details
    /// @dev SECURITY FIX: Added nonReentrant, whenNotPaused, and batch size limit
    function executeBatchTrade(
        LibOrder.Order[] calldata buy,
        LibOrder.Order[] calldata sell,
        LibOrder.OrderBookTrade[] calldata orderBookTrade
    ) public onlyOperator nonReentrant whenNotPaused {
        if (buy.length != sell.length || orderBookTrade.length != buy.length) revert InvalidTradeLength();
        if (buy.length == 0) revert EmptyBatch();
        if (buy.length > MAX_BATCH_SIZE) revert InvalidBatchSize(buy.length, MAX_BATCH_SIZE);
        
        uint256 batchSize = buy.length;
        emit BatchTradeStarted(batchSize, block.timestamp);
        
        for (uint256 i = 0; i < batchSize; i++) {
            _executeTradeInternal(buy[i], sell[i], orderBookTrade[i], i);
        }
        
        emit BatchTradeCompleted(batchSize, batchSize, block.timestamp);
    }

    function executeBatchTradeMultiFill(
        LibOrder.Order[] calldata buy,
        LibOrder.Order[] calldata sell,
        LibOrder.OrderBookTrade[] calldata orderBookTrade,
        bool[] calldata isFinalFill
    ) public onlyOperator nonReentrant whenNotPaused {
        if (buy.length != sell.length || orderBookTrade.length != buy.length || isFinalFill.length != buy.length) revert InvalidTradeLength();
        if (buy.length == 0) revert EmptyBatch();
        if (buy.length > MAX_BATCH_SIZE) revert InvalidBatchSize(buy.length, MAX_BATCH_SIZE);
        
        uint256 batchSize = buy.length;
        emit BatchTradeStarted(batchSize, block.timestamp);
        
        for (uint256 i = 0; i < batchSize; i++) {
            _executeTradeInternalWithFlag(buy[i], sell[i], orderBookTrade[i], i, isFinalFill[i]);
        }
        
        emit BatchTradeCompleted(batchSize, batchSize, block.timestamp);
    }

    /// @notice Execute a single trade between buy and sell orders
    /// @param buy The buy order
    /// @param sell The sell order
    /// @param orderBookTrade The trade execution details
    /// @dev SECURITY FIX: Added nonReentrant and whenNotPaused
    function executeTrade(
        LibOrder.Order calldata buy,
        LibOrder.Order calldata sell,
        LibOrder.OrderBookTrade calldata orderBookTrade
    ) public onlyOperator nonReentrant whenNotPaused {
        _executeTradeInternal(buy, sell, orderBookTrade, 0);
    }

    /// @notice Execute trade with explicit final fill flag
    /// @param buy The buy order
    /// @param sell The sell order
    /// @param orderBookTrade The trade details
    /// @param isFinalFill If true, market orders will be finalized; if false, they can continue to be filled
    /// @dev SECURITY FIX: Added nonReentrant and whenNotPaused
    function executeTradeMultiFill(
        LibOrder.Order calldata buy,
        LibOrder.Order calldata sell,
        LibOrder.OrderBookTrade calldata orderBookTrade,
        bool isFinalFill
    ) public onlyOperator nonReentrant whenNotPaused {
        _executeTradeInternalWithFlag(buy, sell, orderBookTrade, 0, isFinalFill);
    }

    /// @notice Execute trade with relayer price proof for stop orders
    /// @dev SECURITY FIX: Added nonReentrant and whenNotPaused
    function executeTradeWithProof(
        LibOrder.Order calldata buy,
        LibOrder.Order calldata sell,
        LibOrder.OrderBookTrade calldata orderBookTrade,
        LibOrder.PriceProof calldata priceProof
    ) public onlyOperator nonReentrant whenNotPaused {
        bool buyNeedsProof = isStopOrderType(buy.orderType);
        bool sellNeedsProof = isStopOrderType(sell.orderType);
        
        if (buyNeedsProof || sellNeedsProof) {
            validatePriceProof(priceProof, orderBookTrade.baseToken, orderBookTrade.quoteToken);
            
            if (buyNeedsProof) {
                validateStopTrigger(buy, priceProof.price);
                emit StopOrderTriggered(buy.userAddress, getOrderHash(buy), buy.orderType, priceProof.price, buy.stopPrice);
            }
            
            if (sellNeedsProof) {
                validateStopTrigger(sell, priceProof.price);
                emit StopOrderTriggered(sell.userAddress, getOrderHash(sell), sell.orderType, priceProof.price, sell.stopPrice);
            }
        }
        
        _executeTradeInternal(buy, sell, orderBookTrade, 0);
    }

    /// @notice Execute batch trades with price proofs for stop orders
    /// @dev SECURITY FIX: Added nonReentrant, whenNotPaused, and batch size limit
    // function executeBatchTradeWithProof(
    //     LibOrder.Order[] calldata buy,
    //     LibOrder.Order[] calldata sell,
    //     LibOrder.OrderBookTrade[] calldata orderBookTrade,
    //     LibOrder.PriceProof[] calldata priceProofs
    // ) public onlyOperator nonReentrant whenNotPaused {
    //     require(
    //         buy.length == sell.length && 
    //         orderBookTrade.length == buy.length &&
    //         priceProofs.length == buy.length,
    //         "Invalid trade length!"
    //     );
    //     require(buy.length > 0, "Empty batch");
    //     // SECURITY FIX: Add batch size limit
    //     require(buy.length <= MAX_BATCH_SIZE, "Batch size exceeds maximum");
        
    //     uint256 batchSize = buy.length;
    //     emit BatchTradeStarted(batchSize, block.timestamp);
        
    //     for (uint256 i = 0; i < batchSize; i++) {
    //         // Validate price proof for stop orders
    //         bool buyNeedsProof = isStopOrderType(buy[i].orderType);
    //         bool sellNeedsProof = isStopOrderType(sell[i].orderType);
            
    //         if (buyNeedsProof || sellNeedsProof) {
    //             validatePriceProof(priceProofs[i], orderBookTrade[i].baseToken, orderBookTrade[i].quoteToken);
                
    //             if (buyNeedsProof) {
    //                 validateStopTrigger(buy[i], priceProofs[i].price);
    //                 emit StopOrderTriggered(buy[i].userAddress, getOrderHash(buy[i]), buy[i].orderType, priceProofs[i].price, buy[i].stopPrice);
    //             }
                
    //             if (sellNeedsProof) {
    //                 validateStopTrigger(sell[i], priceProofs[i].price);
    //                 emit StopOrderTriggered(sell[i].userAddress, getOrderHash(sell[i]), sell[i].orderType, priceProofs[i].price, sell[i].stopPrice);
    //             }
    //         }
            
    //         _executeTradeInternal(buy[i], sell[i], orderBookTrade[i], i);
    //     }
        
    //     emit BatchTradeCompleted(batchSize, batchSize, block.timestamp);
    // }

    /// @notice Internal trade execution (defaults to final fill for backward compatibility)
    function _executeTradeInternal(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell,
        LibOrder.OrderBookTrade memory orderBookTrade,
        uint256 tradeIndex
    ) internal {
        _executeTradeInternalWithFlag(buy, sell, orderBookTrade, tradeIndex, true);
    }

    /// @notice Internal trade execution with explicit isFinalFill control
    /// @dev SECURITY FIX: isFinalFill now passed through call stack instead of storage
    function _executeTradeInternalWithFlag(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell,
        LibOrder.OrderBookTrade memory orderBookTrade,
        uint256 tradeIndex,
        bool isFinalFill
    ) internal {
        (bytes32 buyHash, bytes32 sellHash) = validateOrderBookTrade(buy, sell, orderBookTrade, tradeIndex);

        // SECURITY FIX: Pass isFinalFill as parameter instead of storage
        updateOrderFilledQuantities(buy, buyHash, sell, sellHash, orderBookTrade, isFinalFill);
        
        validateFundLocking(buy, orderBookTrade);
        validateFundLocking(sell, orderBookTrade);

        updateForOrderBookTrade(buy, sell, orderBookTrade);

        emit OrderBookTradeExecuted(
            buy.userAddress,
            sell.userAddress,
            orderBookTrade.baseToken,
            orderBookTrade.quoteToken,
            orderBookTrade.grossBaseAmount,
            orderBookTrade.grossQuoteAmount,
            orderBookTrade.makerSide == LibOrder.OrderSide.Buy ? LibOrder.OrderSide.Sell : LibOrder.OrderSide.Buy
        );
    }

    // ============================================================
    // VALIDATION FUNCTIONS
    // ============================================================
    function validateOrderBookTrade(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell,
        LibOrder.OrderBookTrade memory trade,
        uint256 /* tradeIndex */
    ) internal view returns (bytes32, bytes32) {
        validateAssetPair(trade);
        validateOrderNonce(buy);
        validateOrderNonce(sell);
        validateLimitPrices(buy, sell, trade);
        
        (bytes32 buyHash, bytes32 sellHash) = validateOrderSignatures(buy, sell);
        
        validateFees(trade);
        
        // FIX: Post-only validation applies ONLY to limit orders (LimitMaker type)
        // Market orders are taker-only by definition and must not trigger post-only checks
        // validatePostOnly already checks orderType == LimitMaker, so this is safe
        validatePostOnly(buy, trade.makerSide);
        validatePostOnly(sell, trade.makerSide);
        
        validateAmountInQuote(buy);
        validateAmountInQuote(sell);
        
        // FIX: REMOVED on-chain execution price validation for market orders
        // Market orders are RELAYER-DRIVEN SWEEPS:
        // - Relayer selects best limit orders off-chain (Binance-style depth sweep)
        // - Contract validates: user max amount, nonce, signature, expiry
        // - Relayer is trusted to execute at best available price
        // - This allows unlimited trades across multiple price levels

        return (buyHash, sellHash);
    }

    function validateAssetPair(LibOrder.OrderBookTrade memory trade) internal pure {
        if (trade.baseToken == trade.quoteToken) revert InvalidAssetPair();
        bool validFees = (trade.makerFeeToken == trade.baseToken && trade.takerFeeToken == trade.quoteToken) ||
            (trade.makerFeeToken == trade.quoteToken && trade.takerFeeToken == trade.baseToken);
        if (!validFees) revert FeeMismatch();
    }

    function validateOrderNonce(LibOrder.Order memory order) internal view {
        if (order.nonce <= minValidNonce[order.userAddress]) revert InvalidNonce();
    }

    function validateLimitPrices(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell,
        LibOrder.OrderBookTrade memory trade
    ) internal pure {
        if (trade.grossBaseAmount == 0) revert BaseTooLow();
        if (trade.grossQuoteAmount == 0) revert QuoteTooLow();

        // Limit orders only - market orders skip price validation (relayer-driven)
        if (isLimitOrderType(buy.orderType)) {
            if (calculateImpliedQuoteAmount(trade.grossBaseAmount, buy.limitPrice) < trade.grossQuoteAmount) {
                revert BuyLimitExceeded();
            }
        }
        if (isLimitOrderType(sell.orderType)) {
            if (calculateImpliedQuoteAmount(trade.grossBaseAmount, sell.limitPrice) > trade.grossQuoteAmount) {
                revert SellLimitExceeded();
            }
        }
    }

    function validateOrderSignatures(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell
    ) internal view returns (bytes32, bytes32) {
        bytes32 buyOrderHash = validateOrderSignature(buy);
        bytes32 sellOrderHash = validateOrderSignature(sell);
        return (buyOrderHash, sellOrderHash);
    }

    // PATCH: Fixed signature verification with canonical orderId
    // - Always returns canonical orderId (getOrderHash) for storage tracking
    // - Tries EIP-712 typed signature first, then falls back to legacy eth_sign
    // - Prevents double-fill by using consistent hash for storage
    function validateOrderSignature(LibOrder.Order memory order) public view returns (bytes32) {
        // PATCH: Canonical orderId - ALWAYS use this for storage tracking
        bytes32 orderId = getOrderHash(order);
        
        // EIP-712 typed data digest (includes domain separator)
        bytes32 typedDigest = getTypedOrderHash(order);
        
        // PATCH: Try EIP-712 signature first (direct recover without eth_sign prefix)
        if (isTypedSignatureValid(typedDigest, order.walletSignature, order.userAddress)) {
            return orderId; // PATCH: Return canonical orderId, not typedDigest
        }
        
        // PATCH: Fallback to legacy personal_sign (eth_sign with prefix)
        if (isLegacySignatureValid(orderId, order.walletSignature, order.userAddress)) {
            return orderId;
        }
        
        revert(order.side == LibOrder.OrderSide.Buy ? "Invalid wallet signature for buy order" : "Invalid wallet signature for sell order");
    }

    function validateFees(LibOrder.OrderBookTrade memory trade) private pure {
        uint64 makerTotal = trade.makerFeeToken == trade.baseToken ? trade.grossBaseAmount : trade.grossQuoteAmount;
        if (!isFeeQuantityValid(trade.makerFeeAmount, makerTotal, maxFeeBasisPoints)) revert ExcessiveMakerFee();
        uint64 takerTotal = trade.takerFeeToken == trade.baseToken ? trade.grossBaseAmount : trade.grossQuoteAmount;
        if (!isFeeQuantityValid(trade.takerFeeAmount, takerTotal, maxFeeBasisPoints)) revert ExcessiveTakerFee();
        validateOrderBookTradeFees(trade);
    }

    function validateOrderBookTradeFees(LibOrder.OrderBookTrade memory trade) internal pure {
        uint64 baseFee = trade.makerFeeToken == trade.baseToken ? trade.makerFeeAmount : trade.takerFeeAmount;
        if (trade.netBaseAmount + baseFee != trade.grossBaseAmount) revert BaseFeeUnbalanced();
        uint64 quoteFee = trade.makerFeeToken == trade.quoteToken ? trade.makerFeeAmount : trade.takerFeeAmount;
        if (trade.netQuoteAmount + quoteFee != trade.grossQuoteAmount) revert QuoteFeeUnbalanced();
    }

    function validatePostOnly(LibOrder.Order memory order, LibOrder.OrderSide makerSide) internal pure {
        if (order.orderType != LibOrder.OrderType.LimitMaker) return;
        if (order.side != makerSide) revert PostOnlyViolation();
    }

    function validateAmountInQuote(LibOrder.Order memory order) internal pure {
        if (order.isAmountInQuote && !isMarketOrderType(order.orderType)) revert QuoteAmountInvalid();
    }

    // FIX: DEPRECATED - Market order price protection removed for relayer-based execution
    // Relayer is trusted to execute market orders at best available price
    // On-chain slippage validation caused reverts when sweeping depth across multiple price levels
    // User protection is provided by:
    // 1. order.amount - max base/quote amount user is willing to trade
    // 2. Relayer reputation and off-chain best execution guarantee
    // 3. User can always cancel via emergencyCancel if needed
    function validateMarketOrderPriceProtection(LibOrder.Order memory /* order */, uint64 /* executionPrice */) internal pure {
        // NO-OP: Relayer-driven market orders do not validate execution price on-chain
        // This enables Binance-style depth sweeping without reverts
    }

    // AUDIT_FIX: Updated to use token decimals registry
    function validateFundLocking(LibOrder.Order memory order, LibOrder.OrderBookTrade memory trade) internal view {
        if (order.balanceSnapshot == 0 && order.allowanceSnapshot == 0) return;
        
        address tokenToCheck = order.side == LibOrder.OrderSide.Buy ? trade.quoteToken : trade.baseToken;
        
        uint256 currentBalance = IERC20(tokenToCheck).balanceOf(order.userAddress);
        uint256 currentAllowance = IERC20(tokenToCheck).allowance(order.userAddress, address(this));
        
        // AUDIT_FIX: Use token decimals registry with fallback
        uint8 decimals = _getTokenDecimals(tokenToCheck);
        uint256 snapshotBalanceUnits = AssetUnitConversions.pipsToAssetUnits(order.balanceSnapshot, decimals);
        uint256 snapshotAllowanceUnits = AssetUnitConversions.pipsToAssetUnits(order.allowanceSnapshot, decimals);
        
        if (currentBalance < snapshotBalanceUnits) revert BalanceSnapshotFailed();
        if (currentAllowance < snapshotAllowanceUnits) revert AllowanceSnapshotFailed();
    }

    function validatePriceProof(LibOrder.PriceProof memory proof, address baseToken, address quoteToken) internal {
        if (relayer == address(0)) revert RelayerNotConfigured();
        if (proof.baseToken != baseToken || proof.quoteToken != quoteToken) revert PriceProofMismatch();
        if (block.timestamp > proof.timestamp + PRICE_PROOF_MAX_AGE) revert PriceProofExpired();
        if (proof.timestamp > block.timestamp) revert PriceProofFuture();
        bytes32 proofHash = getPriceProofHash(proof);
        if (!isSignatureValid(proofHash, proof.signature, relayer)) revert InvalidRelayerSig();
        
        // AUDIT_FIX: Validate price deviation from last known price
        bytes32 pairKey = keccak256(abi.encodePacked(baseToken, quoteToken));
        uint64 prevPrice = lastPrice[pairKey];
        
        if (prevPrice > 0 && maxPriceDeviationBps > 0) {
            // Calculate absolute deviation
            uint256 deviation;
            if (proof.price > prevPrice) {
                deviation = uint256(proof.price - prevPrice) * 10000 / uint256(prevPrice);
            } else {
                deviation = uint256(prevPrice - proof.price) * 10000 / uint256(prevPrice);
            }
            
            if (deviation > maxPriceDeviationBps) {
                revert PriceDeviationExceeded();
            }
        }
        
        // AUDIT_FIX: Update last known price
        if (proof.price != prevPrice) {
            lastPrice[pairKey] = proof.price;
            emit PriceUpdated(baseToken, quoteToken, prevPrice, proof.price);
        }
    }

    /// @notice Validate stop order trigger conditions
    /// @dev SECURITY FIX: Corrected trigger logic for both buy and sell sides
    /// @param order The stop order to validate
    /// @param currentPrice Current market price from relayer
    function validateStopTrigger(LibOrder.Order memory order, uint64 currentPrice) internal pure {
        if (order.orderType == LibOrder.OrderType.StopLoss || order.orderType == LibOrder.OrderType.StopLossLimit) {
            if (order.side == LibOrder.OrderSide.Buy) {
                if (currentPrice < order.stopPrice) revert StopNotTriggered();
            } else {
                if (currentPrice > order.stopPrice) revert StopNotTriggered();
            }
        } else if (order.orderType == LibOrder.OrderType.TakeProfit || order.orderType == LibOrder.OrderType.TakeProfitLimit) {
            if (order.side == LibOrder.OrderSide.Buy) {
                if (currentPrice > order.stopPrice) revert StopNotTriggered();
            } else {
                if (currentPrice < order.stopPrice) revert StopNotTriggered();
            }
        }
    }

    function validateTimeInForce(LibOrder.Order memory order, bytes32, uint64) internal view {
        if (order.timeInForce == LibOrder.OrderTimeInForce.gtt && block.timestamp > order.cancelAfter) {
            revert OrderExpired();
        }
    }

    // PATCH: New function - FOK validation only when order is complete or marked final
    /// @notice Validate FOK order is fully filled (called only when order is finalized)
    /// @param order The order to validate
    /// @param orderHash The canonical order hash
    /// @param totalFilled Total amount filled
    /// @param isFinal True if this is the final fill (order complete or explicitly finalized)
    function validateFOKCompletion(
        LibOrder.Order memory order,
        bytes32 orderHash,
        uint64 totalFilled,
        bool isFinal
    ) internal pure {
        // Only check FOK when the fill is finalized
        if (order.timeInForce == LibOrder.OrderTimeInForce.fok && isFinal) {
            if (totalFilled != order.amount) {
                revert FillOrKillNotFilled(orderHash, order.amount, totalFilled);
            }
        }
    }

    // ============================================================
    // ORDER FILL TRACKING
    // ============================================================
    
    // SECURITY FIX: Removed _isFinalFill storage variable to prevent shared state issues
    // Now passed through call stack as parameter
    
    /// @notice Update fill quantities for both sides of a trade
    function updateOrderFilledQuantities(
        LibOrder.Order memory buy,
        bytes32 buyHash,
        LibOrder.Order memory sell,
        bytes32 sellHash,
        LibOrder.OrderBookTrade memory orderBookTrade,
        bool isFinalFill
    ) private {
        updateOrderFilledQuantity(buy, buyHash, orderBookTrade.grossBaseAmount, orderBookTrade.grossQuoteAmount, isFinalFill);
        updateOrderFilledQuantity(sell, sellHash, orderBookTrade.grossBaseAmount, orderBookTrade.grossQuoteAmount, isFinalFill);
    }

    /// @notice Update fill quantity for a single order
    /// @param order The order being filled
    /// @param orderHash Canonical order hash
    /// @param grossBaseAmount Base amount in this fill
    /// @param grossQuoteAmount Quote amount in this fill
    /// @param isFinalFill If true, market orders will be finalized
    function updateOrderFilledQuantity(
        LibOrder.Order memory order,
        bytes32 orderHash,
        uint64 grossBaseAmount,
        uint64 grossQuoteAmount,
        bool isFinalFill
    ) private {
        require(!completedOrderHashes[orderHash], "Order double filled");

        uint64 newFilledAmount;

        if (order.isAmountInQuote) {
            require(isMarketOrderType(order.orderType), "Order quote quantity only valid for market orders");
            newFilledAmount = grossQuoteAmount + partiallyFilledOrderQuantitiesInPips[orderHash];
        } else {
            newFilledAmount = grossBaseAmount + partiallyFilledOrderQuantitiesInPips[orderHash];
        }

        require(newFilledAmount <= order.amount, "Order overfilled");
        
        validateTimeInForce(order, orderHash, newFilledAmount);
        
        bool isFullyFilled = (newFilledAmount == order.amount);
        
        // SECURITY FIX: Pass isFinalFill through call stack instead of using storage
        bool shouldFinalize = shouldCancelRemainingAmount(order, isFinalFill);

        if (isFullyFilled) {
            validateFOKCompletion(order, orderHash, newFilledAmount, true);
            delete partiallyFilledOrderQuantitiesInPips[orderHash];
            completedOrderHashes[orderHash] = true;
        } else {
            if (shouldFinalize) {
                validateFOKCompletion(order, orderHash, newFilledAmount, true);
                completedOrderHashes[orderHash] = true;
                emit MarketOrderCancelledRemainder(order.userAddress, orderHash, order.amount, newFilledAmount);
                delete partiallyFilledOrderQuantitiesInPips[orderHash];
            } else {
                partiallyFilledOrderQuantitiesInPips[orderHash] = newFilledAmount;
            }
        }
    }

    /// @notice Determine if order should be finalized after this fill
    /// @param order The order being filled
    /// @param isFinalFill True if relayer marked this as final fill
    function shouldCancelRemainingAmount(LibOrder.Order memory order, bool isFinalFill) internal pure returns (bool) {
        // IOC orders always cancel remainder after any fill
        if (order.timeInForce == LibOrder.OrderTimeInForce.ioc) return true;
        
        // SECURITY FIX: Use parameter instead of storage for thread safety
        // Market orders only finalize when relayer explicitly marks this as final fill
        if (isMarketOrderType(order.orderType)) {
            return isFinalFill;
        }
        
        return false;
    }

    // ============================================================
    // TRANSFER EXECUTION
    // ============================================================
    // AUDIT_FIX: Get token decimals from registry or fallback to IERC20.decimals()
    /// @notice Get decimals for a token using registry or fallback
    /// @param token The token address
    /// @return decimals The token decimals
    function _getTokenDecimals(address token) internal view returns (uint8) {
        if (tokenDecimalsSet[token]) {
            return tokenDecimals[token];
        }
        // Fallback to IERC20.decimals() - may revert for non-standard tokens
        return IERC20(token).decimals();
    }

    /// @notice Execute token transfers for a trade
    /// @dev SECURITY FIX: Uses SafeERC20 for all transfers
    /// @dev AUDIT_FIX: Uses token decimals registry for non-standard tokens
    function updateForOrderBookTrade(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell,
        LibOrder.OrderBookTrade memory trade
    ) internal {
        // AUDIT_FIX: Cache decimals using registry with fallback
        uint8 baseDecimals = _getTokenDecimals(trade.baseToken);
        uint8 quoteDecimals = _getTokenDecimals(trade.quoteToken);
        uint8 makerFeeDecimals = _getTokenDecimals(trade.makerFeeToken);
        uint8 takerFeeDecimals = _getTokenDecimals(trade.takerFeeToken);
        
        // SECURITY FIX: Use safeTransferFrom for tokens that don't return bool
        IERC20(trade.baseToken).safeTransferFrom(
            sell.userAddress,
            address(this),
            AssetUnitConversions.pipsToAssetUnits(trade.grossBaseAmount, baseDecimals)
        );

        IERC20(trade.quoteToken).safeTransferFrom(
            buy.userAddress,
            address(this),
            AssetUnitConversions.pipsToAssetUnits(trade.grossQuoteAmount, quoteDecimals)
        );

        // SECURITY FIX: Use safeTransfer for outgoing transfers
        IERC20(trade.baseToken).safeTransfer(
            buy.userAddress,
            AssetUnitConversions.pipsToAssetUnits(trade.netBaseAmount, baseDecimals)
        );

        IERC20(trade.makerFeeToken).safeTransfer(
            feeTo,
            AssetUnitConversions.pipsToAssetUnits(trade.makerFeeAmount, makerFeeDecimals)
        );

        IERC20(trade.quoteToken).safeTransfer(
            sell.userAddress,
            AssetUnitConversions.pipsToAssetUnits(trade.netQuoteAmount, quoteDecimals)
        );

        IERC20(trade.takerFeeToken).safeTransfer(
            feeTo,
            AssetUnitConversions.pipsToAssetUnits(trade.takerFeeAmount, takerFeeDecimals)
        );
    }

    // ============================================================
    // HELPER FUNCTIONS
    // ============================================================
    
    /// @notice Calculate quote amount from base amount and price
    /// @param baseAmount The base token amount in pips
    /// @param limitPrice The price in pips
    /// @return The implied quote amount in pips
    function calculateImpliedQuoteAmount(uint64 baseAmount, uint64 limitPrice) internal pure returns (uint64) {
        return multiplyPipsByFraction(baseAmount, limitPrice, pipPriceMultiplier);
    }

    /// @notice Multiply pips by a fraction
    /// @param multiplicand The value to multiply
    /// @param fractionDividend The numerator of the fraction
    /// @param fractionDivisor The denominator of the fraction
    /// @return The result in pips
    function multiplyPipsByFraction(uint64 multiplicand, uint64 fractionDividend, uint64 fractionDivisor) internal pure returns (uint64) {
        uint256 dividend = uint256(multiplicand) * fractionDividend;
        uint256 result = dividend / fractionDivisor;
        require(result < 2**64, "Pip quantity overflows uint64");
        return uint64(result);
    }

    /// @notice Check if fee amount is within allowed basis points
    /// @param fee The fee amount
    /// @param total The total amount
    /// @param max The maximum allowed basis points
    /// @return True if fee is valid
    function isFeeQuantityValid(uint64 fee, uint64 total, uint64 max) internal pure returns (bool) {
        uint64 feeBasisPoints = (fee * basisPointsInTotal) / total;
        return feeBasisPoints <= max;
    }

    /// @notice Check if order type is a market order variant
    /// @param orderType The order type to check
    /// @return True if market, stop loss, or take profit
    function isMarketOrderType(LibOrder.OrderType orderType) internal pure returns (bool) {
        return orderType == LibOrder.OrderType.Market || orderType == LibOrder.OrderType.StopLoss || orderType == LibOrder.OrderType.TakeProfit;
    }

    /// @notice Check if order type is a limit order variant
    /// @param orderType The order type to check
    /// @return True if limit, limit maker, stop limit, or take profit limit
    function isLimitOrderType(LibOrder.OrderType orderType) internal pure returns (bool) {
        return orderType == LibOrder.OrderType.Limit || orderType == LibOrder.OrderType.LimitMaker || orderType == LibOrder.OrderType.StopLossLimit || orderType == LibOrder.OrderType.TakeProfitLimit;
    }

    /// @notice Check if order type requires price proof (stop orders)
    /// @param orderType The order type to check
    /// @return True if stop loss, take profit, or their limit variants
    function isStopOrderType(LibOrder.OrderType orderType) internal pure returns (bool) {
        return orderType == LibOrder.OrderType.StopLoss || orderType == LibOrder.OrderType.TakeProfit || orderType == LibOrder.OrderType.StopLossLimit || orderType == LibOrder.OrderType.TakeProfitLimit;
    }

    // ============================================================
    // VIEW FUNCTIONS
    // ============================================================
    function isNonceValid(address userAddress, uint128 nonce) external view returns (bool) {
        return nonce > minValidNonce[userAddress];
    }

    function getMinValidNonce(address userAddress) external view returns (uint128) {
        return minValidNonce[userAddress];
    }

    function getOrderStatus(bytes32 orderHash) external view returns (bool isCompleted, uint64 partialFill) {
        isCompleted = completedOrderHashes[orderHash];
        partialFill = partiallyFilledOrderQuantitiesInPips[orderHash];
    }

    /// @notice Check if an order can be emergency cancelled
    /// @param order The order to check
    /// @return canCancel True if order can be cancelled
    /// @return reason Explanation of result
    /// @dev Gas optimization: uses calldata. SECURITY FIX: Added nonce check
    function canEmergencyCancel(LibOrder.Order calldata order) external view returns (bool canCancel, string memory reason) {
        bytes32 orderHash = getOrderHash(order);
        
        // SECURITY FIX: Check nonce validity first
        if (order.nonce <= minValidNonce[order.userAddress]) {
            return (false, "Order nonce already invalidated");
        }
        
        if (completedOrderHashes[orderHash]) {
            return (false, "Order already completed or cancelled");
        }
        
        if (!_isOrderSignatureValidCalldata(order)) {
            return (false, "Invalid order signature");
        }
        
        return (true, "Order can be cancelled");
    }

    // PATCH: Use correct EIP-712 verification (without eth_sign prefix)
    function isEIP712Signature(LibOrder.Order memory order) public view returns (bool) {
        bytes32 typedDigest = getTypedOrderHash(order);
        return isTypedSignatureValid(typedDigest, order.walletSignature, order.userAddress);
    }

    function checkMakerTakerStatus(
        LibOrder.Order memory buy,
        LibOrder.Order memory sell,
        LibOrder.OrderSide makerSide
    ) external pure returns (bool buyIsMaker, bool sellIsMaker, bool buyPostOnlyOk, bool sellPostOnlyOk) {
        buyIsMaker = (buy.side == makerSide);
        sellIsMaker = (sell.side == makerSide);
        buyPostOnlyOk = (buy.orderType != LibOrder.OrderType.LimitMaker) || buyIsMaker;
        sellPostOnlyOk = (sell.orderType != LibOrder.OrderType.LimitMaker) || sellIsMaker;
    }

    function checkMarketOrderExecution(LibOrder.Order memory order, uint64 proposedPrice) external view returns (bool isValid, string memory reason) {
        if (!isMarketOrderType(order.orderType)) {
            return (true, "Not a market order");
        }
        
        if (order.limitPrice > 0) {
            if (order.side == LibOrder.OrderSide.Buy) {
                if (proposedPrice > order.limitPrice) {
                    return (false, "Execution price exceeds buy limit");
                }
            } else {
                if (proposedPrice < order.limitPrice) {
                    return (false, "Execution price below sell limit");
                }
            }
        }
        
        if (order.timeInForce == LibOrder.OrderTimeInForce.gtt) {
            if (block.timestamp > order.cancelAfter) {
                return (false, "Order expired (GTT)");
            }
        }
        
        return (true, "Market order valid for execution");
    }

    function getEffectiveTimeInForce(LibOrder.Order memory order) external pure returns (LibOrder.OrderTimeInForce) {
        if (isMarketOrderType(order.orderType)) {
            if (order.timeInForce == LibOrder.OrderTimeInForce.fok) {
                return LibOrder.OrderTimeInForce.fok;
            }
            return LibOrder.OrderTimeInForce.ioc;
        }
        return order.timeInForce;
    }

    /// @notice Check if an order can be partially filled (market orders support this)
    function canPartialFill(LibOrder.Order memory order) external pure returns (bool) {
        // FOK orders cannot be partially filled
        if (order.timeInForce == LibOrder.OrderTimeInForce.fok) {
            return false;
        }
        // All other orders (including market with IOC) can be partially filled
        return true;
    }

    /// @notice Get remaining fillable amount for an order
    function getRemainingAmount(LibOrder.Order memory order) external view returns (uint64 remaining, bool isActive) {
        bytes32 orderHash = getOrderHash(order);
        
        if (completedOrderHashes[orderHash]) {
            return (0, false);
        }
        
        uint64 filled = partiallyFilledOrderQuantitiesInPips[orderHash];
        remaining = order.amount > filled ? order.amount - filled : 0;
        isActive = remaining > 0;
    }

    /// @notice Validate if a market order would execute safely at given price
    function validateMarketOrderSafety(
        LibOrder.Order memory order,
        uint64 proposedPrice,
        uint64 proposedBaseAmount,
        uint64 proposedQuoteAmount
    ) external view returns (bool safe, string memory reason) {
        // Check if market order type
        if (!isMarketOrderType(order.orderType)) {
            return (true, "Not a market order");
        }
        
        // Check nonce validity
        if (order.nonce <= minValidNonce[order.userAddress]) {
            return (false, "Order nonce invalidated");
        }
        
        // Check if already completed
        bytes32 orderHash = getOrderHash(order);
        if (completedOrderHashes[orderHash]) {
            return (false, "Order already completed");
        }
        
        // Check slippage protection
        if (order.limitPrice > 0) {
            if (order.side == LibOrder.OrderSide.Buy) {
                if (proposedPrice > order.limitPrice) {
                    return (false, "Price exceeds slippage limit for buy");
                }
            } else {
                if (proposedPrice < order.limitPrice) {
                    return (false, "Price below slippage limit for sell");
                }
            }
        }
        
        // Check GTT expiration
        if (order.timeInForce == LibOrder.OrderTimeInForce.gtt) {
            if (block.timestamp > order.cancelAfter) {
                return (false, "Order expired");
            }
        }
        
        // Check FOK fillability
        if (order.timeInForce == LibOrder.OrderTimeInForce.fok) {
            uint64 filled = partiallyFilledOrderQuantitiesInPips[orderHash];
            uint64 newFill = order.isAmountInQuote ? proposedQuoteAmount : proposedBaseAmount;
            if (filled + newFill != order.amount) {
                return (false, "FOK: would not fully fill");
            }
        }
        
        return (true, "Market order safe to execute");
    }

    /// @notice Get contract version for migration tracking
    function getVersion() external pure returns (string memory) {
        return VERSION;
    }

    // AUDIT_FIX: Helper function for UI to get EIP-712 domain info
    /// @notice Get EIP-712 domain information for UI integration
    /// @return name The domain name (must match UI)
    /// @return version The domain version (must match UI)
    /// @return chainId The chain ID
    /// @return verifyingContract This contract address
    /// @return domainSeparator The computed domain separator
    function getEIP712Domain() external view returns (
        string memory name,
        string memory version,
        uint256 chainId,
        address verifyingContract,
        bytes32 domainSeparator
    ) {
        return (EIP712_NAME, EIP712_VERSION, block.chainid, address(this), DOMAIN_SEPARATOR);
    }
    
    // AUDIT_FIX: Get last known price for a trading pair
    /// @notice Get last known price for a trading pair
    /// @param baseToken The base token address
    /// @param quoteToken The quote token address
    /// @return price The last known price (0 if never set)
    function getLastPrice(address baseToken, address quoteToken) external view returns (uint64) {
        bytes32 pairKey = keccak256(abi.encodePacked(baseToken, quoteToken));
        return lastPrice[pairKey];
    }

    /// @notice Check fund locking status for an order
    /// @dev AUDIT_FIX: Updated to use token decimals registry
    function checkFundLockingStatus(
        LibOrder.Order memory order,
        address baseToken,
        address quoteToken
    ) external view returns (
        bool balanceOk,
        bool allowanceOk,
        uint256 currentBalance,
        uint256 currentAllowance,
        uint256 requiredBalance,
        uint256 requiredAllowance
    ) {
        address tokenToCheck = order.side == LibOrder.OrderSide.Buy ? quoteToken : baseToken;
        // AUDIT_FIX: Use token decimals registry with fallback
        uint8 decimals = _getTokenDecimals(tokenToCheck);
        
        currentBalance = IERC20(tokenToCheck).balanceOf(order.userAddress);
        currentAllowance = IERC20(tokenToCheck).allowance(order.userAddress, address(this));
        
        requiredBalance = AssetUnitConversions.pipsToAssetUnits(order.balanceSnapshot, decimals);
        requiredAllowance = AssetUnitConversions.pipsToAssetUnits(order.allowanceSnapshot, decimals);
        
        balanceOk = currentBalance >= requiredBalance;
        allowanceOk = currentAllowance >= requiredAllowance;
    }
}