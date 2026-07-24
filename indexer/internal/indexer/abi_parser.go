package indexer

import (
	"encoding/json"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/nowafinance/nowa-zk/indexer/internal/indexer/types"
	"github.com/nowafinance/nowa-zk/indexer/pkg/logger"
)

const orderbookABIJSON = `[{"inputs":[{"components":[{"internalType":"address","name":"userAddress","type":"address"},{"internalType":"address","name":"baseToken","type":"address"},{"internalType":"address","name":"quoteToken","type":"address"},{"internalType":"uint64","name":"amount","type":"uint64"},{"internalType":"bool","name":"isAmountInQuote","type":"bool"},{"internalType":"uint128","name":"nonce","type":"uint128"},{"internalType":"enum LibOrder.OrderType","name":"orderType","type":"uint8"},{"internalType":"enum LibOrder.OrderSide","name":"side","type":"uint8"},{"internalType":"uint64","name":"limitPrice","type":"uint64"},{"internalType":"uint64","name":"stopPrice","type":"uint64"},{"internalType":"enum LibOrder.OrderTimeInForce","name":"timeInForce","type":"uint8"},{"internalType":"uint64","name":"cancelAfter","type":"uint64"},{"internalType":"bytes","name":"walletSignature","type":"bytes"},{"internalType":"uint64","name":"balanceSnapshot","type":"uint64"},{"internalType":"uint64","name":"allowanceSnapshot","type":"uint64"}],"internalType":"struct LibOrder.Order[]","name":"buy","type":"tuple[]"},{"components":[{"internalType":"address","name":"userAddress","type":"address"},{"internalType":"address","name":"baseToken","type":"address"},{"internalType":"address","name":"quoteToken","type":"address"},{"internalType":"uint64","name":"amount","type":"uint64"},{"internalType":"bool","name":"isAmountInQuote","type":"bool"},{"internalType":"uint128","name":"nonce","type":"uint128"},{"internalType":"enum LibOrder.OrderType","name":"orderType","type":"uint8"},{"internalType":"enum LibOrder.OrderSide","name":"side","type":"uint8"},{"internalType":"uint64","name":"limitPrice","type":"uint64"},{"internalType":"uint64","name":"stopPrice","type":"uint64"},{"internalType":"enum LibOrder.OrderTimeInForce","name":"timeInForce","type":"uint8"},{"internalType":"uint64","name":"cancelAfter","type":"uint64"},{"internalType":"bytes","name":"walletSignature","type":"bytes"},{"internalType":"uint64","name":"balanceSnapshot","type":"uint64"},{"internalType":"uint64","name":"allowanceSnapshot","type":"uint64"}],"internalType":"struct LibOrder.Order[]","name":"sell","type":"tuple[]"},{"components":[{"internalType":"address","name":"baseToken","type":"address"},{"internalType":"address","name":"quoteToken","type":"address"},{"internalType":"uint64","name":"grossBaseAmount","type":"uint64"},{"internalType":"uint64","name":"grossQuoteAmount","type":"uint64"},{"internalType":"uint64","name":"netBaseAmount","type":"uint64"},{"internalType":"uint64","name":"netQuoteAmount","type":"uint64"},{"internalType":"address","name":"makerFeeToken","type":"address"},{"internalType":"address","name":"takerFeeToken","type":"address"},{"internalType":"uint64","name":"makerFeeAmount","type":"uint64"},{"internalType":"uint64","name":"takerFeeAmount","type":"uint64"},{"internalType":"uint64","name":"price","type":"uint64"},{"internalType":"enum LibOrder.OrderSide","name":"makerSide","type":"uint8"}],"internalType":"struct LibOrder.OrderBookTrade[]","name":"orderBookTrade","type":"tuple[]"},{"internalType":"bool[]","name":"isFinalFill","type":"bool[]"}],"name":"executeBatchTradeMultiFill","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

var parsedABI abi.ABI

func init() {
	var err error
	parsedABI, err = abi.JSON(strings.NewReader(orderbookABIJSON))
	if err != nil {
		panic(fmt.Sprintf("Failed to parse Orderbook ABI: %v", err))
	}
}

type Order struct {
	UserAddress       common.Address
	BaseToken         common.Address
	QuoteToken        common.Address
	Amount            uint64
	IsAmountInQuote   bool
	Nonce             *big.Int
	OrderType         uint8
	Side              uint8
	LimitPrice        uint64
	StopPrice         uint64
	TimeInForce       uint8
	CancelAfter       uint64
	WalletSignature   []byte
	BalanceSnapshot   uint64
	AllowanceSnapshot uint64
}

// ParseTradesFromTxData parses the executeBatchTradeMultiFill calldata and extracts the trades
func ParseTradesFromTxData(txDataHex string) ([]*types.ParsedTrade, error) {
	if strings.HasPrefix(txDataHex, "0x") {
		txDataHex = txDataHex[2:]
	}
	
	txData, err := hex.DecodeString(txDataHex)
	if err != nil {
		return nil, err
	}
	
	if len(txData) < 4 {
		return nil, fmt.Errorf("transaction data too short")
	}

	method, err := parsedABI.MethodById(txData[:4])
	if err != nil {
		return nil, err
	}

	args, err := method.Inputs.Unpack(txData[4:])
	if err != nil {
		return nil, err
	}

	// args[0] is buy orders, args[1] is sell orders
	// They are parsed as slices of structs, which in go-ethereum abi unmarshals to []struct{...}
	buyOrdersInterface := args[0]
	sellOrdersInterface := args[1]

	var buys []Order
	if err := mapToStruct(buyOrdersInterface, &buys); err != nil {
		return nil, fmt.Errorf("failed to map buy orders: %w", err)
	}

	var sells []Order
	if err := mapToStruct(sellOrdersInterface, &sells); err != nil {
		return nil, fmt.Errorf("failed to map sell orders: %w", err)
	}

	var parsedTrades []*types.ParsedTrade

	for _, order := range buys {
		trade, err := extractTrade(order)
		if err == nil && trade != nil {
			parsedTrades = append(parsedTrades, trade)
		} else if err != nil {
			logger.Warn("Failed to extract trade from buy order (bad signature?): %v", err)
		}
	}
	
	for _, order := range sells {
		trade, err := extractTrade(order)
		if err == nil && trade != nil {
			parsedTrades = append(parsedTrades, trade)
		} else if err != nil {
			logger.Warn("Failed to extract trade from sell order (bad signature?): %v", err)
		}
	}

	return parsedTrades, nil
}

// hacky map to struct
func mapToStruct(input interface{}, output interface{}) error {
	b, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, output)
}

func extractTrade(order Order) (*types.ParsedTrade, error) {
	domain := apitypes.TypedDataDomain{
		Name:              "Nowa_Orderbook",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(262144), // 0x40000
		VerifyingContract: "0xd28c36adcd614d4ec42ca800114eded644b28189",
	}

	typesDef := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Order": {
			{Name: "userAddress", Type: "address"},
			{Name: "baseToken", Type: "address"},
			{Name: "quoteToken", Type: "address"},
			{Name: "amount", Type: "uint64"},
			{Name: "isAmountInQuote", Type: "bool"},
			{Name: "nonce", Type: "uint128"},
			{Name: "orderType", Type: "uint8"},
			{Name: "side", Type: "uint8"},
			{Name: "limitPrice", Type: "uint64"},
			{Name: "stopPrice", Type: "uint64"},
			{Name: "timeInForce", Type: "uint8"},
			{Name: "cancelAfter", Type: "uint64"},
			{Name: "balanceSnapshot", Type: "uint64"},
			{Name: "allowanceSnapshot", Type: "uint64"},
		},
	}

	message := map[string]interface{}{
		"userAddress":       fmt.Sprintf("0x%x", order.UserAddress),
		"baseToken":         fmt.Sprintf("0x%x", order.BaseToken),
		"quoteToken":        fmt.Sprintf("0x%x", order.QuoteToken),
		"amount":            fmt.Sprintf("%d", order.Amount),
		"isAmountInQuote":   order.IsAmountInQuote,
		"nonce":             order.Nonce.String(),
		"orderType":         fmt.Sprintf("%d", order.OrderType),
		"side":              fmt.Sprintf("%d", order.Side),
		"limitPrice":        fmt.Sprintf("%d", order.LimitPrice),
		"stopPrice":         fmt.Sprintf("%d", order.StopPrice),
		"timeInForce":       fmt.Sprintf("%d", order.TimeInForce),
		"cancelAfter":       fmt.Sprintf("%d", order.CancelAfter),
		"balanceSnapshot":   fmt.Sprintf("%d", order.BalanceSnapshot),
		"allowanceSnapshot": fmt.Sprintf("%d", order.AllowanceSnapshot),
	}

	typedData := apitypes.TypedData{
		Types:       typesDef,
		PrimaryType: "Order",
		Domain:      domain,
		Message:     message,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}

	rawData := append(append([]byte("\x19\x01"), domainSeparator...), typedDataHash...)
	messageHash := crypto.Keccak256(rawData)

	sig := order.WalletSignature
	if len(sig) < 65 {
		return nil, fmt.Errorf("invalid signature length: %d", len(sig))
	}

	if sig[64] >= 27 {
		sig[64] -= 27
	}

	pubKeyBytes, err := crypto.Ecrecover(messageHash, sig)
	if err != nil {
		return nil, err
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	return &types.ParsedTrade{
		MessageHash: hex.EncodeToString(messageHash),
		PubKeyX:     hex.EncodeToString(pubKey.X.Bytes()),
		PubKeyY:     hex.EncodeToString(pubKey.Y.Bytes()),
		SigR:        hex.EncodeToString(sig[:32]),
		SigS:        hex.EncodeToString(sig[32:64]),
	}, nil
}
