// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// NowaRollupMetaData contains all meta data concerning the NowaRollup contract.
var NowaRollupMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_verifier\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[{\"name\":\"_tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_pubKeyX\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_pubKeyY\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isProver\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nextTokenId\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerToken\",\"inputs\":[{\"name\":\"_tokenAddress\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stateRoot\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"submitBatch\",\"inputs\":[{\"name\":\"proof\",\"type\":\"uint256[8]\",\"internalType\":\"uint256[8]\"},{\"name\":\"_oldRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_newRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_withdrawalHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_depositHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"tokenIds\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"tokens\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"verifier\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractVerifier\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[{\"name\":\"_tokenId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_merkleProof\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Deposit\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"pubKeyX\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"pubKeyY\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"StateTransition\",\"inputs\":[{\"name\":\"oldRoot\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"newRoot\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"withdrawalHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"depositHash\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TokenRegistered\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"tokenAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Withdrawal\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"tokenId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
	Bin: "0x60806040526004805463ffffffff19166001179055348015601e575f5ffd5b50604051610beb380380610beb833981016040819052603b916081565b5f80546001600160a01b039092166001600160a01b03199283161781556006805490921633908117909255908152600560205260409020805460ff1916600117905560ac565b5f602082840312156090575f5ffd5b81516001600160a01b038116811460a5575f5ffd5b9392505050565b610b32806100b95f395ff3fe608060405234801561000f575f5ffd5b50600436106100a6575f3560e01c80638da5cb5b1161006e5780638da5cb5b146101585780639588eca21461016b57806398c54bd014610182578063e05edf4714610195578063fbb6272d146101a8578063fc97a303146101d0575f5ffd5b806309824a80146100aa5780630a245924146100bf5780632b7ac3f3146100f657806331dc2ac91461012057806375794a3c14610133575b5f5ffd5b6100bd6100b83660046108f8565b6101f5565b005b6100e16100cd3660046108f8565b60056020525f908152604090205460ff1681565b60405190151581526020015b60405180910390f35b5f54610108906001600160a01b031681565b6040516001600160a01b0390911681526020016100ed565b6100bd61012e36600461093d565b6103bf565b6004546101439063ffffffff1681565b60405163ffffffff90911681526020016100ed565b600654610108906001600160a01b031681565b61017460015481565b6040519081526020016100ed565b6100bd610190366004610973565b6105cf565b6100bd6101a33660046109bc565b61073e565b6101086101b6366004610a42565b60026020525f90815260409020546001600160a01b031681565b6101436101de3660046108f8565b60036020525f908152604090205463ffffffff1681565b6006546001600160a01b031633146102405760405162461bcd60e51b81526020600482015260096024820152682737ba1037bbb732b960b91b60448201526064015b60405180910390fd5b6001600160a01b0381165f9081526003602052604090205463ffffffff16156102ab5760405162461bcd60e51b815260206004820152601860248201527f546f6b656e20616c7265616479207265676973746572656400000000000000006044820152606401610237565b60045461010063ffffffff909116106103165760405162461bcd60e51b815260206004820152602760248201527f4d61782032353620746f6b656e7320737570706f7274656420696e204d65726b6044820152666c65205472656560c81b6064820152608401610237565b600480545f9163ffffffff909116908261032f83610a5b565b82546101009290920a63ffffffff81810219909316918316021790915581165f81815260026020908152604080832080546001600160a01b0319166001600160a01b0389169081179091558084526003909252808320805463ffffffff19168517905551939450927f5ff9640201b39f40edb488543c2ae401d4c46a473002fe04940802f52bb5dc869190a35050565b63ffffffff84165f908152600260205260409020546001600160a01b0316806104245760405162461bcd60e51b8152602060048201526017602482015276151bdad95b881251081b9bdd081c9959da5cdd195c9959604a1b6044820152606401610237565b5f84116104735760405162461bcd60e51b815260206004820152601a60248201527f4465706f73697420616d6f756e74206d757374206265203e20300000000000006044820152606401610237565b604051336024820152306044820152606481018590525f9081906001600160a01b0384169060840160408051601f198184030181529181526020820180516001600160e01b03166323b872dd60e01b179052516104d09190610a8b565b5f604051808303815f865af19150503d805f8114610509576040519150601f19603f3d011682016040523d82523d5f602084013e61050e565b606091505b50915091508180156105385750805115806105385750808060200190518101906105389190610aa1565b61057a5760405162461bcd60e51b8152602060048201526013602482015272151c985b9cd9995c919c9bdb4819985a5b1959606a1b6044820152606401610237565b604080518781526020810187905290810185905263ffffffff88169033907ff26e5eca58d1c9efeeccf6feb2d1e8fb950cb24cb35af2fd9e04d802f1329baa906060015b60405180910390a350505050505050565b335f9081526005602052604090205460ff1661062d5760405162461bcd60e51b815260206004820152601860248201527f4e6f7420616e20617574686f72697a65642070726f76657200000000000000006044820152606401610237565b60015484146106775760405162461bcd60e51b8152602060048201526016602482015275125b9d985b1a59081bdb19081cdd185d19481c9bdbdd60521b6044820152606401610237565b61067f6108da565b848152602081018490526040808201849052606082018390525f549051632357251160e01b81526001600160a01b03909116906323572511906106c89089908590600401610ac0565b5f6040518083038186803b1580156106de575f5ffd5b505afa1580156106f0573d5f5f3e3d5ffd5b5050506001859055506040805184815260208101849052859187917fb210818d9fd1c2e0bac5252783fbc1a2e23d398fa7c0032fbcf356b60a211d09910160405180910390a3505050505050565b63ffffffff84165f908152600260205260409020546001600160a01b0316806107a35760405162461bcd60e51b8152602060048201526017602482015276151bdad95b881251081b9bdd081c9959da5cdd195c9959604a1b6044820152606401610237565b604051336024820152604481018590525f9081906001600160a01b0384169060640160408051601f198184030181529181526020820180516001600160e01b031663a9059cbb60e01b179052516107fa9190610a8b565b5f604051808303815f865af19150503d805f8114610833576040519150601f19603f3d011682016040523d82523d5f602084013e610838565b606091505b50915091508180156108625750805115806108625750808060200190518101906108629190610aa1565b6108a05760405162461bcd60e51b815260206004820152600f60248201526e151c985b9cd9995c8819985a5b1959608a1b6044820152606401610237565b60405186815263ffffffff88169033907f8b60e29d3c07d9ba40987d25265f2d18da3cf27979c7155c0428d6b10c93de75906020016105be565b60405180608001604052806004906020820280368337509192915050565b5f60208284031215610908575f5ffd5b81356001600160a01b038116811461091e575f5ffd5b9392505050565b803563ffffffff81168114610938575f5ffd5b919050565b5f5f5f5f60808587031215610950575f5ffd5b61095985610925565b966020860135965060408601359560600135945092505050565b5f5f5f5f5f6101808688031215610988575f5ffd5b610100860187811115610999575f5ffd5b959795359650505061012086013593610140870135935061016087013592509050565b5f5f5f5f606085870312156109cf575f5ffd5b6109d885610925565b935060208501359250604085013567ffffffffffffffff8111156109fa575f5ffd5b8501601f81018713610a0a575f5ffd5b803567ffffffffffffffff811115610a20575f5ffd5b8760208260051b8401011115610a34575f5ffd5b949793965060200194505050565b5f60208284031215610a52575f5ffd5b61091e82610925565b5f63ffffffff821663ffffffff8103610a8257634e487b7160e01b5f52601160045260245ffd5b60010192915050565b5f82518060208501845e5f920191825250919050565b5f60208284031215610ab1575f5ffd5b8151801515811461091e575f5ffd5b61018081016101008483376101008201835f5b6004811015610af2578151835260209283019290910190600101610ad3565b505050939250505056fea2646970667358221220313822c3330657fceac8c0125fa809bb86d2a9226852493c18051d413b37b0f064736f6c634300081e0033",
}

// NowaRollupABI is the input ABI used to generate the binding from.
// Deprecated: Use NowaRollupMetaData.ABI instead.
var NowaRollupABI = NowaRollupMetaData.ABI

// NowaRollupBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use NowaRollupMetaData.Bin instead.
var NowaRollupBin = NowaRollupMetaData.Bin

// DeployNowaRollup deploys a new Ethereum contract, binding an instance of NowaRollup to it.
func DeployNowaRollup(auth *bind.TransactOpts, backend bind.ContractBackend, _verifier common.Address) (common.Address, *types.Transaction, *NowaRollup, error) {
	parsed, err := NowaRollupMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(NowaRollupBin), backend, _verifier)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &NowaRollup{NowaRollupCaller: NowaRollupCaller{contract: contract}, NowaRollupTransactor: NowaRollupTransactor{contract: contract}, NowaRollupFilterer: NowaRollupFilterer{contract: contract}}, nil
}

// NowaRollup is an auto generated Go binding around an Ethereum contract.
type NowaRollup struct {
	NowaRollupCaller     // Read-only binding to the contract
	NowaRollupTransactor // Write-only binding to the contract
	NowaRollupFilterer   // Log filterer for contract events
}

// NowaRollupCaller is an auto generated read-only Go binding around an Ethereum contract.
type NowaRollupCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NowaRollupTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NowaRollupTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NowaRollupFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NowaRollupFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NowaRollupSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NowaRollupSession struct {
	Contract     *NowaRollup       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// NowaRollupCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NowaRollupCallerSession struct {
	Contract *NowaRollupCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// NowaRollupTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NowaRollupTransactorSession struct {
	Contract     *NowaRollupTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// NowaRollupRaw is an auto generated low-level Go binding around an Ethereum contract.
type NowaRollupRaw struct {
	Contract *NowaRollup // Generic contract binding to access the raw methods on
}

// NowaRollupCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NowaRollupCallerRaw struct {
	Contract *NowaRollupCaller // Generic read-only contract binding to access the raw methods on
}

// NowaRollupTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NowaRollupTransactorRaw struct {
	Contract *NowaRollupTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNowaRollup creates a new instance of NowaRollup, bound to a specific deployed contract.
func NewNowaRollup(address common.Address, backend bind.ContractBackend) (*NowaRollup, error) {
	contract, err := bindNowaRollup(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NowaRollup{NowaRollupCaller: NowaRollupCaller{contract: contract}, NowaRollupTransactor: NowaRollupTransactor{contract: contract}, NowaRollupFilterer: NowaRollupFilterer{contract: contract}}, nil
}

// NewNowaRollupCaller creates a new read-only instance of NowaRollup, bound to a specific deployed contract.
func NewNowaRollupCaller(address common.Address, caller bind.ContractCaller) (*NowaRollupCaller, error) {
	contract, err := bindNowaRollup(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NowaRollupCaller{contract: contract}, nil
}

// NewNowaRollupTransactor creates a new write-only instance of NowaRollup, bound to a specific deployed contract.
func NewNowaRollupTransactor(address common.Address, transactor bind.ContractTransactor) (*NowaRollupTransactor, error) {
	contract, err := bindNowaRollup(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NowaRollupTransactor{contract: contract}, nil
}

// NewNowaRollupFilterer creates a new log filterer instance of NowaRollup, bound to a specific deployed contract.
func NewNowaRollupFilterer(address common.Address, filterer bind.ContractFilterer) (*NowaRollupFilterer, error) {
	contract, err := bindNowaRollup(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NowaRollupFilterer{contract: contract}, nil
}

// bindNowaRollup binds a generic wrapper to an already deployed contract.
func bindNowaRollup(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NowaRollupMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NowaRollup *NowaRollupRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NowaRollup.Contract.NowaRollupCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NowaRollup *NowaRollupRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NowaRollup.Contract.NowaRollupTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NowaRollup *NowaRollupRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NowaRollup.Contract.NowaRollupTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NowaRollup *NowaRollupCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NowaRollup.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NowaRollup *NowaRollupTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NowaRollup.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NowaRollup *NowaRollupTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NowaRollup.Contract.contract.Transact(opts, method, params...)
}

// IsProver is a free data retrieval call binding the contract method 0x0a245924.
//
// Solidity: function isProver(address ) view returns(bool)
func (_NowaRollup *NowaRollupCaller) IsProver(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "isProver", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsProver is a free data retrieval call binding the contract method 0x0a245924.
//
// Solidity: function isProver(address ) view returns(bool)
func (_NowaRollup *NowaRollupSession) IsProver(arg0 common.Address) (bool, error) {
	return _NowaRollup.Contract.IsProver(&_NowaRollup.CallOpts, arg0)
}

// IsProver is a free data retrieval call binding the contract method 0x0a245924.
//
// Solidity: function isProver(address ) view returns(bool)
func (_NowaRollup *NowaRollupCallerSession) IsProver(arg0 common.Address) (bool, error) {
	return _NowaRollup.Contract.IsProver(&_NowaRollup.CallOpts, arg0)
}

// NextTokenId is a free data retrieval call binding the contract method 0x75794a3c.
//
// Solidity: function nextTokenId() view returns(uint32)
func (_NowaRollup *NowaRollupCaller) NextTokenId(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "nextTokenId")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// NextTokenId is a free data retrieval call binding the contract method 0x75794a3c.
//
// Solidity: function nextTokenId() view returns(uint32)
func (_NowaRollup *NowaRollupSession) NextTokenId() (uint32, error) {
	return _NowaRollup.Contract.NextTokenId(&_NowaRollup.CallOpts)
}

// NextTokenId is a free data retrieval call binding the contract method 0x75794a3c.
//
// Solidity: function nextTokenId() view returns(uint32)
func (_NowaRollup *NowaRollupCallerSession) NextTokenId() (uint32, error) {
	return _NowaRollup.Contract.NextTokenId(&_NowaRollup.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NowaRollup *NowaRollupCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NowaRollup *NowaRollupSession) Owner() (common.Address, error) {
	return _NowaRollup.Contract.Owner(&_NowaRollup.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_NowaRollup *NowaRollupCallerSession) Owner() (common.Address, error) {
	return _NowaRollup.Contract.Owner(&_NowaRollup.CallOpts)
}

// StateRoot is a free data retrieval call binding the contract method 0x9588eca2.
//
// Solidity: function stateRoot() view returns(bytes32)
func (_NowaRollup *NowaRollupCaller) StateRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "stateRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// StateRoot is a free data retrieval call binding the contract method 0x9588eca2.
//
// Solidity: function stateRoot() view returns(bytes32)
func (_NowaRollup *NowaRollupSession) StateRoot() ([32]byte, error) {
	return _NowaRollup.Contract.StateRoot(&_NowaRollup.CallOpts)
}

// StateRoot is a free data retrieval call binding the contract method 0x9588eca2.
//
// Solidity: function stateRoot() view returns(bytes32)
func (_NowaRollup *NowaRollupCallerSession) StateRoot() ([32]byte, error) {
	return _NowaRollup.Contract.StateRoot(&_NowaRollup.CallOpts)
}

// TokenIds is a free data retrieval call binding the contract method 0xfc97a303.
//
// Solidity: function tokenIds(address ) view returns(uint32)
func (_NowaRollup *NowaRollupCaller) TokenIds(opts *bind.CallOpts, arg0 common.Address) (uint32, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "tokenIds", arg0)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// TokenIds is a free data retrieval call binding the contract method 0xfc97a303.
//
// Solidity: function tokenIds(address ) view returns(uint32)
func (_NowaRollup *NowaRollupSession) TokenIds(arg0 common.Address) (uint32, error) {
	return _NowaRollup.Contract.TokenIds(&_NowaRollup.CallOpts, arg0)
}

// TokenIds is a free data retrieval call binding the contract method 0xfc97a303.
//
// Solidity: function tokenIds(address ) view returns(uint32)
func (_NowaRollup *NowaRollupCallerSession) TokenIds(arg0 common.Address) (uint32, error) {
	return _NowaRollup.Contract.TokenIds(&_NowaRollup.CallOpts, arg0)
}

// Tokens is a free data retrieval call binding the contract method 0xfbb6272d.
//
// Solidity: function tokens(uint32 ) view returns(address)
func (_NowaRollup *NowaRollupCaller) Tokens(opts *bind.CallOpts, arg0 uint32) (common.Address, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "tokens", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Tokens is a free data retrieval call binding the contract method 0xfbb6272d.
//
// Solidity: function tokens(uint32 ) view returns(address)
func (_NowaRollup *NowaRollupSession) Tokens(arg0 uint32) (common.Address, error) {
	return _NowaRollup.Contract.Tokens(&_NowaRollup.CallOpts, arg0)
}

// Tokens is a free data retrieval call binding the contract method 0xfbb6272d.
//
// Solidity: function tokens(uint32 ) view returns(address)
func (_NowaRollup *NowaRollupCallerSession) Tokens(arg0 uint32) (common.Address, error) {
	return _NowaRollup.Contract.Tokens(&_NowaRollup.CallOpts, arg0)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_NowaRollup *NowaRollupCaller) Verifier(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _NowaRollup.contract.Call(opts, &out, "verifier")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_NowaRollup *NowaRollupSession) Verifier() (common.Address, error) {
	return _NowaRollup.Contract.Verifier(&_NowaRollup.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_NowaRollup *NowaRollupCallerSession) Verifier() (common.Address, error) {
	return _NowaRollup.Contract.Verifier(&_NowaRollup.CallOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0x31dc2ac9.
//
// Solidity: function deposit(uint32 _tokenId, uint256 _amount, uint256 _pubKeyX, uint256 _pubKeyY) returns()
func (_NowaRollup *NowaRollupTransactor) Deposit(opts *bind.TransactOpts, _tokenId uint32, _amount *big.Int, _pubKeyX *big.Int, _pubKeyY *big.Int) (*types.Transaction, error) {
	return _NowaRollup.contract.Transact(opts, "deposit", _tokenId, _amount, _pubKeyX, _pubKeyY)
}

// Deposit is a paid mutator transaction binding the contract method 0x31dc2ac9.
//
// Solidity: function deposit(uint32 _tokenId, uint256 _amount, uint256 _pubKeyX, uint256 _pubKeyY) returns()
func (_NowaRollup *NowaRollupSession) Deposit(_tokenId uint32, _amount *big.Int, _pubKeyX *big.Int, _pubKeyY *big.Int) (*types.Transaction, error) {
	return _NowaRollup.Contract.Deposit(&_NowaRollup.TransactOpts, _tokenId, _amount, _pubKeyX, _pubKeyY)
}

// Deposit is a paid mutator transaction binding the contract method 0x31dc2ac9.
//
// Solidity: function deposit(uint32 _tokenId, uint256 _amount, uint256 _pubKeyX, uint256 _pubKeyY) returns()
func (_NowaRollup *NowaRollupTransactorSession) Deposit(_tokenId uint32, _amount *big.Int, _pubKeyX *big.Int, _pubKeyY *big.Int) (*types.Transaction, error) {
	return _NowaRollup.Contract.Deposit(&_NowaRollup.TransactOpts, _tokenId, _amount, _pubKeyX, _pubKeyY)
}

// RegisterToken is a paid mutator transaction binding the contract method 0x09824a80.
//
// Solidity: function registerToken(address _tokenAddress) returns()
func (_NowaRollup *NowaRollupTransactor) RegisterToken(opts *bind.TransactOpts, _tokenAddress common.Address) (*types.Transaction, error) {
	return _NowaRollup.contract.Transact(opts, "registerToken", _tokenAddress)
}

// RegisterToken is a paid mutator transaction binding the contract method 0x09824a80.
//
// Solidity: function registerToken(address _tokenAddress) returns()
func (_NowaRollup *NowaRollupSession) RegisterToken(_tokenAddress common.Address) (*types.Transaction, error) {
	return _NowaRollup.Contract.RegisterToken(&_NowaRollup.TransactOpts, _tokenAddress)
}

// RegisterToken is a paid mutator transaction binding the contract method 0x09824a80.
//
// Solidity: function registerToken(address _tokenAddress) returns()
func (_NowaRollup *NowaRollupTransactorSession) RegisterToken(_tokenAddress common.Address) (*types.Transaction, error) {
	return _NowaRollup.Contract.RegisterToken(&_NowaRollup.TransactOpts, _tokenAddress)
}

// SubmitBatch is a paid mutator transaction binding the contract method 0x98c54bd0.
//
// Solidity: function submitBatch(uint256[8] proof, bytes32 _oldRoot, bytes32 _newRoot, bytes32 _withdrawalHash, bytes32 _depositHash) returns()
func (_NowaRollup *NowaRollupTransactor) SubmitBatch(opts *bind.TransactOpts, proof [8]*big.Int, _oldRoot [32]byte, _newRoot [32]byte, _withdrawalHash [32]byte, _depositHash [32]byte) (*types.Transaction, error) {
	return _NowaRollup.contract.Transact(opts, "submitBatch", proof, _oldRoot, _newRoot, _withdrawalHash, _depositHash)
}

// SubmitBatch is a paid mutator transaction binding the contract method 0x98c54bd0.
//
// Solidity: function submitBatch(uint256[8] proof, bytes32 _oldRoot, bytes32 _newRoot, bytes32 _withdrawalHash, bytes32 _depositHash) returns()
func (_NowaRollup *NowaRollupSession) SubmitBatch(proof [8]*big.Int, _oldRoot [32]byte, _newRoot [32]byte, _withdrawalHash [32]byte, _depositHash [32]byte) (*types.Transaction, error) {
	return _NowaRollup.Contract.SubmitBatch(&_NowaRollup.TransactOpts, proof, _oldRoot, _newRoot, _withdrawalHash, _depositHash)
}

// SubmitBatch is a paid mutator transaction binding the contract method 0x98c54bd0.
//
// Solidity: function submitBatch(uint256[8] proof, bytes32 _oldRoot, bytes32 _newRoot, bytes32 _withdrawalHash, bytes32 _depositHash) returns()
func (_NowaRollup *NowaRollupTransactorSession) SubmitBatch(proof [8]*big.Int, _oldRoot [32]byte, _newRoot [32]byte, _withdrawalHash [32]byte, _depositHash [32]byte) (*types.Transaction, error) {
	return _NowaRollup.Contract.SubmitBatch(&_NowaRollup.TransactOpts, proof, _oldRoot, _newRoot, _withdrawalHash, _depositHash)
}

// Withdraw is a paid mutator transaction binding the contract method 0xe05edf47.
//
// Solidity: function withdraw(uint32 _tokenId, uint256 _amount, bytes32[] _merkleProof) returns()
func (_NowaRollup *NowaRollupTransactor) Withdraw(opts *bind.TransactOpts, _tokenId uint32, _amount *big.Int, _merkleProof [][32]byte) (*types.Transaction, error) {
	return _NowaRollup.contract.Transact(opts, "withdraw", _tokenId, _amount, _merkleProof)
}

// Withdraw is a paid mutator transaction binding the contract method 0xe05edf47.
//
// Solidity: function withdraw(uint32 _tokenId, uint256 _amount, bytes32[] _merkleProof) returns()
func (_NowaRollup *NowaRollupSession) Withdraw(_tokenId uint32, _amount *big.Int, _merkleProof [][32]byte) (*types.Transaction, error) {
	return _NowaRollup.Contract.Withdraw(&_NowaRollup.TransactOpts, _tokenId, _amount, _merkleProof)
}

// Withdraw is a paid mutator transaction binding the contract method 0xe05edf47.
//
// Solidity: function withdraw(uint32 _tokenId, uint256 _amount, bytes32[] _merkleProof) returns()
func (_NowaRollup *NowaRollupTransactorSession) Withdraw(_tokenId uint32, _amount *big.Int, _merkleProof [][32]byte) (*types.Transaction, error) {
	return _NowaRollup.Contract.Withdraw(&_NowaRollup.TransactOpts, _tokenId, _amount, _merkleProof)
}

// NowaRollupDepositIterator is returned from FilterDeposit and is used to iterate over the raw logs and unpacked data for Deposit events raised by the NowaRollup contract.
type NowaRollupDepositIterator struct {
	Event *NowaRollupDeposit // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NowaRollupDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NowaRollupDeposit)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NowaRollupDeposit)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NowaRollupDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NowaRollupDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NowaRollupDeposit represents a Deposit event raised by the NowaRollup contract.
type NowaRollupDeposit struct {
	User    common.Address
	TokenId uint32
	Amount  *big.Int
	PubKeyX *big.Int
	PubKeyY *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterDeposit is a free log retrieval operation binding the contract event 0xf26e5eca58d1c9efeeccf6feb2d1e8fb950cb24cb35af2fd9e04d802f1329baa.
//
// Solidity: event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY)
func (_NowaRollup *NowaRollupFilterer) FilterDeposit(opts *bind.FilterOpts, user []common.Address, tokenId []uint32) (*NowaRollupDepositIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NowaRollup.contract.FilterLogs(opts, "Deposit", userRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &NowaRollupDepositIterator{contract: _NowaRollup.contract, event: "Deposit", logs: logs, sub: sub}, nil
}

// WatchDeposit is a free log subscription operation binding the contract event 0xf26e5eca58d1c9efeeccf6feb2d1e8fb950cb24cb35af2fd9e04d802f1329baa.
//
// Solidity: event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY)
func (_NowaRollup *NowaRollupFilterer) WatchDeposit(opts *bind.WatchOpts, sink chan<- *NowaRollupDeposit, user []common.Address, tokenId []uint32) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NowaRollup.contract.WatchLogs(opts, "Deposit", userRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NowaRollupDeposit)
				if err := _NowaRollup.contract.UnpackLog(event, "Deposit", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDeposit is a log parse operation binding the contract event 0xf26e5eca58d1c9efeeccf6feb2d1e8fb950cb24cb35af2fd9e04d802f1329baa.
//
// Solidity: event Deposit(address indexed user, uint32 indexed tokenId, uint256 amount, uint256 pubKeyX, uint256 pubKeyY)
func (_NowaRollup *NowaRollupFilterer) ParseDeposit(log types.Log) (*NowaRollupDeposit, error) {
	event := new(NowaRollupDeposit)
	if err := _NowaRollup.contract.UnpackLog(event, "Deposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NowaRollupStateTransitionIterator is returned from FilterStateTransition and is used to iterate over the raw logs and unpacked data for StateTransition events raised by the NowaRollup contract.
type NowaRollupStateTransitionIterator struct {
	Event *NowaRollupStateTransition // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NowaRollupStateTransitionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NowaRollupStateTransition)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NowaRollupStateTransition)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NowaRollupStateTransitionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NowaRollupStateTransitionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NowaRollupStateTransition represents a StateTransition event raised by the NowaRollup contract.
type NowaRollupStateTransition struct {
	OldRoot        [32]byte
	NewRoot        [32]byte
	WithdrawalHash [32]byte
	DepositHash    [32]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterStateTransition is a free log retrieval operation binding the contract event 0xb210818d9fd1c2e0bac5252783fbc1a2e23d398fa7c0032fbcf356b60a211d09.
//
// Solidity: event StateTransition(bytes32 indexed oldRoot, bytes32 indexed newRoot, bytes32 withdrawalHash, bytes32 depositHash)
func (_NowaRollup *NowaRollupFilterer) FilterStateTransition(opts *bind.FilterOpts, oldRoot [][32]byte, newRoot [][32]byte) (*NowaRollupStateTransitionIterator, error) {

	var oldRootRule []interface{}
	for _, oldRootItem := range oldRoot {
		oldRootRule = append(oldRootRule, oldRootItem)
	}
	var newRootRule []interface{}
	for _, newRootItem := range newRoot {
		newRootRule = append(newRootRule, newRootItem)
	}

	logs, sub, err := _NowaRollup.contract.FilterLogs(opts, "StateTransition", oldRootRule, newRootRule)
	if err != nil {
		return nil, err
	}
	return &NowaRollupStateTransitionIterator{contract: _NowaRollup.contract, event: "StateTransition", logs: logs, sub: sub}, nil
}

// WatchStateTransition is a free log subscription operation binding the contract event 0xb210818d9fd1c2e0bac5252783fbc1a2e23d398fa7c0032fbcf356b60a211d09.
//
// Solidity: event StateTransition(bytes32 indexed oldRoot, bytes32 indexed newRoot, bytes32 withdrawalHash, bytes32 depositHash)
func (_NowaRollup *NowaRollupFilterer) WatchStateTransition(opts *bind.WatchOpts, sink chan<- *NowaRollupStateTransition, oldRoot [][32]byte, newRoot [][32]byte) (event.Subscription, error) {

	var oldRootRule []interface{}
	for _, oldRootItem := range oldRoot {
		oldRootRule = append(oldRootRule, oldRootItem)
	}
	var newRootRule []interface{}
	for _, newRootItem := range newRoot {
		newRootRule = append(newRootRule, newRootItem)
	}

	logs, sub, err := _NowaRollup.contract.WatchLogs(opts, "StateTransition", oldRootRule, newRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NowaRollupStateTransition)
				if err := _NowaRollup.contract.UnpackLog(event, "StateTransition", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseStateTransition is a log parse operation binding the contract event 0xb210818d9fd1c2e0bac5252783fbc1a2e23d398fa7c0032fbcf356b60a211d09.
//
// Solidity: event StateTransition(bytes32 indexed oldRoot, bytes32 indexed newRoot, bytes32 withdrawalHash, bytes32 depositHash)
func (_NowaRollup *NowaRollupFilterer) ParseStateTransition(log types.Log) (*NowaRollupStateTransition, error) {
	event := new(NowaRollupStateTransition)
	if err := _NowaRollup.contract.UnpackLog(event, "StateTransition", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NowaRollupTokenRegisteredIterator is returned from FilterTokenRegistered and is used to iterate over the raw logs and unpacked data for TokenRegistered events raised by the NowaRollup contract.
type NowaRollupTokenRegisteredIterator struct {
	Event *NowaRollupTokenRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NowaRollupTokenRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NowaRollupTokenRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NowaRollupTokenRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NowaRollupTokenRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NowaRollupTokenRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NowaRollupTokenRegistered represents a TokenRegistered event raised by the NowaRollup contract.
type NowaRollupTokenRegistered struct {
	TokenId      uint32
	TokenAddress common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterTokenRegistered is a free log retrieval operation binding the contract event 0x5ff9640201b39f40edb488543c2ae401d4c46a473002fe04940802f52bb5dc86.
//
// Solidity: event TokenRegistered(uint32 indexed tokenId, address indexed tokenAddress)
func (_NowaRollup *NowaRollupFilterer) FilterTokenRegistered(opts *bind.FilterOpts, tokenId []uint32, tokenAddress []common.Address) (*NowaRollupTokenRegisteredIterator, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var tokenAddressRule []interface{}
	for _, tokenAddressItem := range tokenAddress {
		tokenAddressRule = append(tokenAddressRule, tokenAddressItem)
	}

	logs, sub, err := _NowaRollup.contract.FilterLogs(opts, "TokenRegistered", tokenIdRule, tokenAddressRule)
	if err != nil {
		return nil, err
	}
	return &NowaRollupTokenRegisteredIterator{contract: _NowaRollup.contract, event: "TokenRegistered", logs: logs, sub: sub}, nil
}

// WatchTokenRegistered is a free log subscription operation binding the contract event 0x5ff9640201b39f40edb488543c2ae401d4c46a473002fe04940802f52bb5dc86.
//
// Solidity: event TokenRegistered(uint32 indexed tokenId, address indexed tokenAddress)
func (_NowaRollup *NowaRollupFilterer) WatchTokenRegistered(opts *bind.WatchOpts, sink chan<- *NowaRollupTokenRegistered, tokenId []uint32, tokenAddress []common.Address) (event.Subscription, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var tokenAddressRule []interface{}
	for _, tokenAddressItem := range tokenAddress {
		tokenAddressRule = append(tokenAddressRule, tokenAddressItem)
	}

	logs, sub, err := _NowaRollup.contract.WatchLogs(opts, "TokenRegistered", tokenIdRule, tokenAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NowaRollupTokenRegistered)
				if err := _NowaRollup.contract.UnpackLog(event, "TokenRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTokenRegistered is a log parse operation binding the contract event 0x5ff9640201b39f40edb488543c2ae401d4c46a473002fe04940802f52bb5dc86.
//
// Solidity: event TokenRegistered(uint32 indexed tokenId, address indexed tokenAddress)
func (_NowaRollup *NowaRollupFilterer) ParseTokenRegistered(log types.Log) (*NowaRollupTokenRegistered, error) {
	event := new(NowaRollupTokenRegistered)
	if err := _NowaRollup.contract.UnpackLog(event, "TokenRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// NowaRollupWithdrawalIterator is returned from FilterWithdrawal and is used to iterate over the raw logs and unpacked data for Withdrawal events raised by the NowaRollup contract.
type NowaRollupWithdrawalIterator struct {
	Event *NowaRollupWithdrawal // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *NowaRollupWithdrawalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(NowaRollupWithdrawal)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(NowaRollupWithdrawal)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *NowaRollupWithdrawalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *NowaRollupWithdrawalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// NowaRollupWithdrawal represents a Withdrawal event raised by the NowaRollup contract.
type NowaRollupWithdrawal struct {
	User    common.Address
	TokenId uint32
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterWithdrawal is a free log retrieval operation binding the contract event 0x8b60e29d3c07d9ba40987d25265f2d18da3cf27979c7155c0428d6b10c93de75.
//
// Solidity: event Withdrawal(address indexed user, uint32 indexed tokenId, uint256 amount)
func (_NowaRollup *NowaRollupFilterer) FilterWithdrawal(opts *bind.FilterOpts, user []common.Address, tokenId []uint32) (*NowaRollupWithdrawalIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NowaRollup.contract.FilterLogs(opts, "Withdrawal", userRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return &NowaRollupWithdrawalIterator{contract: _NowaRollup.contract, event: "Withdrawal", logs: logs, sub: sub}, nil
}

// WatchWithdrawal is a free log subscription operation binding the contract event 0x8b60e29d3c07d9ba40987d25265f2d18da3cf27979c7155c0428d6b10c93de75.
//
// Solidity: event Withdrawal(address indexed user, uint32 indexed tokenId, uint256 amount)
func (_NowaRollup *NowaRollupFilterer) WatchWithdrawal(opts *bind.WatchOpts, sink chan<- *NowaRollupWithdrawal, user []common.Address, tokenId []uint32) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}

	logs, sub, err := _NowaRollup.contract.WatchLogs(opts, "Withdrawal", userRule, tokenIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(NowaRollupWithdrawal)
				if err := _NowaRollup.contract.UnpackLog(event, "Withdrawal", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawal is a log parse operation binding the contract event 0x8b60e29d3c07d9ba40987d25265f2d18da3cf27979c7155c0428d6b10c93de75.
//
// Solidity: event Withdrawal(address indexed user, uint32 indexed tokenId, uint256 amount)
func (_NowaRollup *NowaRollupFilterer) ParseWithdrawal(log types.Log) (*NowaRollupWithdrawal, error) {
	event := new(NowaRollupWithdrawal)
	if err := _NowaRollup.contract.UnpackLog(event, "Withdrawal", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
