package circuits

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/consensys/gnark/test"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func TestTradeSignatureCircuit_RealData(t *testing.T) {
	assert := test.NewAssert(t)

	// 3. Extract the first buy order manually using the provided data
	userAddrBytes, _ := hex.DecodeString("c318336673be2B50171bA7adAE0A0a00EC1f8361")
	baseTokenBytes, _ := hex.DecodeString("4c3350f8C0877D575e985035285BFC16d78Ed118")
	quoteTokenBytes, _ := hex.DecodeString("5257bdd1F82de2b5906e83bdb87776DBA7DBDa97")
	sigBytes, _ := hex.DecodeString("c60a85c7013cefd64ffe7201097560e801d4f206c6bf074b3c52d1f59a708c953e5f4ddd5861f6b29bcb11ea92bb964e5ced9889d422f096979c75a4e9fa20a11c")

	type Order struct {
		UserAddress       [20]byte
		BaseToken         [20]byte
		QuoteToken        [20]byte
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

	order := Order{
		Amount:            4334831153,
		IsAmountInQuote:   false,
		Nonce:             big.NewInt(1784633722930),
		OrderType:         0,
		Side:              0,
		LimitPrice:        57672373200,
		StopPrice:         0,
		TimeInForce:       0,
		CancelAfter:       0,
		WalletSignature:   sigBytes,
		BalanceSnapshot:   0,
		AllowanceSnapshot: 0,
	}
	copy(order.UserAddress[:], userAddrBytes)
	copy(order.BaseToken[:], baseTokenBytes)
	copy(order.QuoteToken[:], quoteTokenBytes)

	// 4. Compute EIP-712 Message Hash
	domain := apitypes.TypedDataDomain{
		Name:              "Nowa_Orderbook",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(262144), // 0x40000
		VerifyingContract: "0xd28c36adcd614d4ec42ca800114eded644b28189",
	}

	// Reconstruct the Order type for go-ethereum TypedData
	types := apitypes.Types{
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
		Types:       types,
		PrimaryType: "Order",
		Domain:      domain,
		Message:     message,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	assert.NoError(err)
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	assert.NoError(err)

	rawData := fmt.Appendf(nil, "\x19\x01%s%s", string(domainSeparator), string(typedDataHash))
	messageHash := crypto.Keccak256(rawData)

	// 5. Recover Public Key using go-ethereum to ensure signature is correct
	sig := order.WalletSignature
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	
	pubKeyBytes, err := crypto.Ecrecover(messageHash, sig)
	assert.NoError(err)
	
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	assert.NoError(err)

	// 6. Assign values to circuit witness
	var witness TradeSignatureCircuit

	witness.MessageHash = emulated.ValueOf[emulated.Secp256k1Fr](messageHash)

	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()
	witness.PubKey.X = emulated.ValueOf[emulated.Secp256k1Fp](xBytes[:])
	witness.PubKey.Y = emulated.ValueOf[emulated.Secp256k1Fp](yBytes[:])

	rBytes := sig[:32]
	sBytes := sig[32:64]
	witness.Sig.R = emulated.ValueOf[emulated.Secp256k1Fr](rBytes[:])
	witness.Sig.S = emulated.ValueOf[emulated.Secp256k1Fr](sBytes[:])

	// 7. Test circuit with real data
	assert.ProverSucceeded(&TradeSignatureCircuit{}, &witness, test.WithCurves(ecc.BN254))
}
