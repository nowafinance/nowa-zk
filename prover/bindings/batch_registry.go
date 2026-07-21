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

// BatchRegistryBatch is an auto generated low-level Go binding around an user-defined struct.
type BatchRegistryBatch struct {
	BatchHash    [32]byte
	OldStateRoot [32]byte
	NewStateRoot [32]byte
	Submitter    common.Address
	Timestamp    *big.Int
	VerifiedAt   *big.Int
	Status       uint8
}

// BatchRegistryMetaData contains all meta data concerning the BatchRegistry contract.
var BatchRegistryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_verifier\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_stateManager\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_indexer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_finalizationDelay\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"STATE_MANAGER\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractStateManager\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batchExists\",\"inputs\":[{\"name\":\"batchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"exists\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batchHashToNumber\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"batches\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"batchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"oldStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"newStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"submitter\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifiedAt\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"enumBatchRegistry.BatchStatus\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getBatch\",\"inputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"batch\",\"type\":\"tuple\",\"internalType\":\"structBatchRegistry.Batch\",\"components\":[{\"name\":\"batchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"oldStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"newStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"submitter\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifiedAt\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"status\",\"type\":\"uint8\",\"internalType\":\"enumBatchRegistry.BatchStatus\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getBatchNumber\",\"inputs\":[{\"name\":\"batchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCurrentStateRoot\",\"inputs\":[],\"outputs\":[{\"name\":\"stateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nextBatchNumber\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"paused\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerBatch\",\"inputs\":[{\"name\":\"batchHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"newStateRoot\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"batchData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"proofA\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"proofB\",\"type\":\"uint256[2][2]\",\"internalType\":\"uint256[2][2]\"},{\"name\":\"proofC\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"publicInputs\",\"type\":\"uint256[6]\",\"internalType\":\"uint256[6]\"}],\"outputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"indexer\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalBatches\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unpause\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateIndexer\",\"inputs\":[{\"name\":\"newIndexer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateVerifier\",\"inputs\":[{\"name\":\"newVerifier\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"verifier\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIVerifier\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"BatchFinalized\",\"inputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"newStateRoot\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BatchRegistered\",\"inputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"batchHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"oldStateRoot\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"newStateRoot\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"submitter\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"timestamp\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FinalizationDelayUpdated\",\"inputs\":[{\"name\":\"oldDelay\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"newDelay\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Paused\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"IndexerUpdated\",\"inputs\":[{\"name\":\"oldIndexer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newIndexer\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Unpaused\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"VerifierUpdated\",\"inputs\":[{\"name\":\"oldVerifier\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newVerifier\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false}]",
}

// BatchRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use BatchRegistryMetaData.ABI instead.
var BatchRegistryABI = BatchRegistryMetaData.ABI

// BatchRegistry is an auto generated Go binding around an Ethereum contract.
type BatchRegistry struct {
	BatchRegistryCaller     // Read-only binding to the contract
	BatchRegistryTransactor // Write-only binding to the contract
	BatchRegistryFilterer   // Log filterer for contract events
}

// BatchRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type BatchRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BatchRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BatchRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BatchRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BatchRegistrySession struct {
	Contract     *BatchRegistry    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BatchRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BatchRegistryCallerSession struct {
	Contract *BatchRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// BatchRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BatchRegistryTransactorSession struct {
	Contract     *BatchRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// BatchRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type BatchRegistryRaw struct {
	Contract *BatchRegistry // Generic contract binding to access the raw methods on
}

// BatchRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BatchRegistryCallerRaw struct {
	Contract *BatchRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// BatchRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BatchRegistryTransactorRaw struct {
	Contract *BatchRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBatchRegistry creates a new instance of BatchRegistry, bound to a specific deployed contract.
func NewBatchRegistry(address common.Address, backend bind.ContractBackend) (*BatchRegistry, error) {
	contract, err := bindBatchRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BatchRegistry{BatchRegistryCaller: BatchRegistryCaller{contract: contract}, BatchRegistryTransactor: BatchRegistryTransactor{contract: contract}, BatchRegistryFilterer: BatchRegistryFilterer{contract: contract}}, nil
}

// NewBatchRegistryCaller creates a new read-only instance of BatchRegistry, bound to a specific deployed contract.
func NewBatchRegistryCaller(address common.Address, caller bind.ContractCaller) (*BatchRegistryCaller, error) {
	contract, err := bindBatchRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryCaller{contract: contract}, nil
}

// NewBatchRegistryTransactor creates a new write-only instance of BatchRegistry, bound to a specific deployed contract.
func NewBatchRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*BatchRegistryTransactor, error) {
	contract, err := bindBatchRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryTransactor{contract: contract}, nil
}

// NewBatchRegistryFilterer creates a new log filterer instance of BatchRegistry, bound to a specific deployed contract.
func NewBatchRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*BatchRegistryFilterer, error) {
	contract, err := bindBatchRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryFilterer{contract: contract}, nil
}

// bindBatchRegistry binds a generic wrapper to an already deployed contract.
func bindBatchRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := BatchRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BatchRegistry *BatchRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BatchRegistry.Contract.BatchRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BatchRegistry *BatchRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BatchRegistry.Contract.BatchRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BatchRegistry *BatchRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BatchRegistry.Contract.BatchRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BatchRegistry *BatchRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BatchRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BatchRegistry *BatchRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BatchRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BatchRegistry *BatchRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BatchRegistry.Contract.contract.Transact(opts, method, params...)
}

// STATEMANAGER is a free data retrieval call binding the contract method 0x2450a4a2.
//
// Solidity: function STATE_MANAGER() view returns(address)
func (_BatchRegistry *BatchRegistryCaller) STATEMANAGER(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "STATE_MANAGER")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// STATEMANAGER is a free data retrieval call binding the contract method 0x2450a4a2.
//
// Solidity: function STATE_MANAGER() view returns(address)
func (_BatchRegistry *BatchRegistrySession) STATEMANAGER() (common.Address, error) {
	return _BatchRegistry.Contract.STATEMANAGER(&_BatchRegistry.CallOpts)
}

// STATEMANAGER is a free data retrieval call binding the contract method 0x2450a4a2.
//
// Solidity: function STATE_MANAGER() view returns(address)
func (_BatchRegistry *BatchRegistryCallerSession) STATEMANAGER() (common.Address, error) {
	return _BatchRegistry.Contract.STATEMANAGER(&_BatchRegistry.CallOpts)
}

// BatchExists is a free data retrieval call binding the contract method 0xecd144a7.
//
// Solidity: function batchExists(bytes32 batchHash) view returns(bool exists)
func (_BatchRegistry *BatchRegistryCaller) BatchExists(opts *bind.CallOpts, batchHash [32]byte) (bool, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "batchExists", batchHash)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// BatchExists is a free data retrieval call binding the contract method 0xecd144a7.
//
// Solidity: function batchExists(bytes32 batchHash) view returns(bool exists)
func (_BatchRegistry *BatchRegistrySession) BatchExists(batchHash [32]byte) (bool, error) {
	return _BatchRegistry.Contract.BatchExists(&_BatchRegistry.CallOpts, batchHash)
}

// BatchExists is a free data retrieval call binding the contract method 0xecd144a7.
//
// Solidity: function batchExists(bytes32 batchHash) view returns(bool exists)
func (_BatchRegistry *BatchRegistryCallerSession) BatchExists(batchHash [32]byte) (bool, error) {
	return _BatchRegistry.Contract.BatchExists(&_BatchRegistry.CallOpts, batchHash)
}

// BatchHashToNumber is a free data retrieval call binding the contract method 0x17ad3a48.
//
// Solidity: function batchHashToNumber(bytes32 ) view returns(uint256)
func (_BatchRegistry *BatchRegistryCaller) BatchHashToNumber(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "batchHashToNumber", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BatchHashToNumber is a free data retrieval call binding the contract method 0x17ad3a48.
//
// Solidity: function batchHashToNumber(bytes32 ) view returns(uint256)
func (_BatchRegistry *BatchRegistrySession) BatchHashToNumber(arg0 [32]byte) (*big.Int, error) {
	return _BatchRegistry.Contract.BatchHashToNumber(&_BatchRegistry.CallOpts, arg0)
}

// BatchHashToNumber is a free data retrieval call binding the contract method 0x17ad3a48.
//
// Solidity: function batchHashToNumber(bytes32 ) view returns(uint256)
func (_BatchRegistry *BatchRegistryCallerSession) BatchHashToNumber(arg0 [32]byte) (*big.Int, error) {
	return _BatchRegistry.Contract.BatchHashToNumber(&_BatchRegistry.CallOpts, arg0)
}

// Batches is a free data retrieval call binding the contract method 0xb32c4d8d.
//
// Solidity: function batches(uint256 ) view returns(bytes32 batchHash, bytes32 oldStateRoot, bytes32 newStateRoot, address submitter, uint256 timestamp, uint256 verifiedAt, uint8 status)
func (_BatchRegistry *BatchRegistryCaller) Batches(opts *bind.CallOpts, arg0 *big.Int) (struct {
	BatchHash    [32]byte
	OldStateRoot [32]byte
	NewStateRoot [32]byte
	Submitter    common.Address
	Timestamp    *big.Int
	VerifiedAt   *big.Int
	Status       uint8
}, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "batches", arg0)

	outstruct := new(struct {
		BatchHash    [32]byte
		OldStateRoot [32]byte
		NewStateRoot [32]byte
		Submitter    common.Address
		Timestamp    *big.Int
		VerifiedAt   *big.Int
		Status       uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.BatchHash = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.OldStateRoot = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.NewStateRoot = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)
	outstruct.Submitter = *abi.ConvertType(out[3], new(common.Address)).(*common.Address)
	outstruct.Timestamp = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.VerifiedAt = *abi.ConvertType(out[5], new(*big.Int)).(**big.Int)
	outstruct.Status = *abi.ConvertType(out[6], new(uint8)).(*uint8)

	return *outstruct, err

}

// Batches is a free data retrieval call binding the contract method 0xb32c4d8d.
//
// Solidity: function batches(uint256 ) view returns(bytes32 batchHash, bytes32 oldStateRoot, bytes32 newStateRoot, address submitter, uint256 timestamp, uint256 verifiedAt, uint8 status)
func (_BatchRegistry *BatchRegistrySession) Batches(arg0 *big.Int) (struct {
	BatchHash    [32]byte
	OldStateRoot [32]byte
	NewStateRoot [32]byte
	Submitter    common.Address
	Timestamp    *big.Int
	VerifiedAt   *big.Int
	Status       uint8
}, error) {
	return _BatchRegistry.Contract.Batches(&_BatchRegistry.CallOpts, arg0)
}

// Batches is a free data retrieval call binding the contract method 0xb32c4d8d.
//
// Solidity: function batches(uint256 ) view returns(bytes32 batchHash, bytes32 oldStateRoot, bytes32 newStateRoot, address submitter, uint256 timestamp, uint256 verifiedAt, uint8 status)
func (_BatchRegistry *BatchRegistryCallerSession) Batches(arg0 *big.Int) (struct {
	BatchHash    [32]byte
	OldStateRoot [32]byte
	NewStateRoot [32]byte
	Submitter    common.Address
	Timestamp    *big.Int
	VerifiedAt   *big.Int
	Status       uint8
}, error) {
	return _BatchRegistry.Contract.Batches(&_BatchRegistry.CallOpts, arg0)
}

// GetBatch is a free data retrieval call binding the contract method 0x5ac44282.
//
// Solidity: function getBatch(uint256 batchNumber) view returns((bytes32,bytes32,bytes32,address,uint256,uint256,uint8) batch)
func (_BatchRegistry *BatchRegistryCaller) GetBatch(opts *bind.CallOpts, batchNumber *big.Int) (BatchRegistryBatch, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "getBatch", batchNumber)

	if err != nil {
		return *new(BatchRegistryBatch), err
	}

	out0 := *abi.ConvertType(out[0], new(BatchRegistryBatch)).(*BatchRegistryBatch)

	return out0, err

}

// GetBatch is a free data retrieval call binding the contract method 0x5ac44282.
//
// Solidity: function getBatch(uint256 batchNumber) view returns((bytes32,bytes32,bytes32,address,uint256,uint256,uint8) batch)
func (_BatchRegistry *BatchRegistrySession) GetBatch(batchNumber *big.Int) (BatchRegistryBatch, error) {
	return _BatchRegistry.Contract.GetBatch(&_BatchRegistry.CallOpts, batchNumber)
}

// GetBatch is a free data retrieval call binding the contract method 0x5ac44282.
//
// Solidity: function getBatch(uint256 batchNumber) view returns((bytes32,bytes32,bytes32,address,uint256,uint256,uint8) batch)
func (_BatchRegistry *BatchRegistryCallerSession) GetBatch(batchNumber *big.Int) (BatchRegistryBatch, error) {
	return _BatchRegistry.Contract.GetBatch(&_BatchRegistry.CallOpts, batchNumber)
}

// GetBatchNumber is a free data retrieval call binding the contract method 0x98a875f5.
//
// Solidity: function getBatchNumber(bytes32 batchHash) view returns(uint256 batchNumber)
func (_BatchRegistry *BatchRegistryCaller) GetBatchNumber(opts *bind.CallOpts, batchHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "getBatchNumber", batchHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBatchNumber is a free data retrieval call binding the contract method 0x98a875f5.
//
// Solidity: function getBatchNumber(bytes32 batchHash) view returns(uint256 batchNumber)
func (_BatchRegistry *BatchRegistrySession) GetBatchNumber(batchHash [32]byte) (*big.Int, error) {
	return _BatchRegistry.Contract.GetBatchNumber(&_BatchRegistry.CallOpts, batchHash)
}

// GetBatchNumber is a free data retrieval call binding the contract method 0x98a875f5.
//
// Solidity: function getBatchNumber(bytes32 batchHash) view returns(uint256 batchNumber)
func (_BatchRegistry *BatchRegistryCallerSession) GetBatchNumber(batchHash [32]byte) (*big.Int, error) {
	return _BatchRegistry.Contract.GetBatchNumber(&_BatchRegistry.CallOpts, batchHash)
}

// GetCurrentStateRoot is a free data retrieval call binding the contract method 0x974af738.
//
// Solidity: function getCurrentStateRoot() view returns(bytes32 stateRoot)
func (_BatchRegistry *BatchRegistryCaller) GetCurrentStateRoot(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "getCurrentStateRoot")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetCurrentStateRoot is a free data retrieval call binding the contract method 0x974af738.
//
// Solidity: function getCurrentStateRoot() view returns(bytes32 stateRoot)
func (_BatchRegistry *BatchRegistrySession) GetCurrentStateRoot() ([32]byte, error) {
	return _BatchRegistry.Contract.GetCurrentStateRoot(&_BatchRegistry.CallOpts)
}

// GetCurrentStateRoot is a free data retrieval call binding the contract method 0x974af738.
//
// Solidity: function getCurrentStateRoot() view returns(bytes32 stateRoot)
func (_BatchRegistry *BatchRegistryCallerSession) GetCurrentStateRoot() ([32]byte, error) {
	return _BatchRegistry.Contract.GetCurrentStateRoot(&_BatchRegistry.CallOpts)
}

// NextBatchNumber is a free data retrieval call binding the contract method 0xc0decb36.
//
// Solidity: function nextBatchNumber() view returns(uint256)
func (_BatchRegistry *BatchRegistryCaller) NextBatchNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "nextBatchNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextBatchNumber is a free data retrieval call binding the contract method 0xc0decb36.
//
// Solidity: function nextBatchNumber() view returns(uint256)
func (_BatchRegistry *BatchRegistrySession) NextBatchNumber() (*big.Int, error) {
	return _BatchRegistry.Contract.NextBatchNumber(&_BatchRegistry.CallOpts)
}

// NextBatchNumber is a free data retrieval call binding the contract method 0xc0decb36.
//
// Solidity: function nextBatchNumber() view returns(uint256)
func (_BatchRegistry *BatchRegistryCallerSession) NextBatchNumber() (*big.Int, error) {
	return _BatchRegistry.Contract.NextBatchNumber(&_BatchRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BatchRegistry *BatchRegistryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BatchRegistry *BatchRegistrySession) Owner() (common.Address, error) {
	return _BatchRegistry.Contract.Owner(&_BatchRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BatchRegistry *BatchRegistryCallerSession) Owner() (common.Address, error) {
	return _BatchRegistry.Contract.Owner(&_BatchRegistry.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_BatchRegistry *BatchRegistryCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_BatchRegistry *BatchRegistrySession) Paused() (bool, error) {
	return _BatchRegistry.Contract.Paused(&_BatchRegistry.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_BatchRegistry *BatchRegistryCallerSession) Paused() (bool, error) {
	return _BatchRegistry.Contract.Paused(&_BatchRegistry.CallOpts)
}

// Indexer is a free data retrieval call binding the contract method 0x5c1bba38.
//
// Solidity: function indexer() view returns(address)
func (_BatchRegistry *BatchRegistryCaller) Indexer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "indexer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Indexer is a free data retrieval call binding the contract method 0x5c1bba38.
//
// Solidity: function indexer() view returns(address)
func (_BatchRegistry *BatchRegistrySession) Indexer() (common.Address, error) {
	return _BatchRegistry.Contract.Indexer(&_BatchRegistry.CallOpts)
}

// Indexer is a free data retrieval call binding the contract method 0x5c1bba38.
//
// Solidity: function indexer() view returns(address)
func (_BatchRegistry *BatchRegistryCallerSession) Indexer() (common.Address, error) {
	return _BatchRegistry.Contract.Indexer(&_BatchRegistry.CallOpts)
}

// TotalBatches is a free data retrieval call binding the contract method 0x69ff6abb.
//
// Solidity: function totalBatches() view returns(uint256)
func (_BatchRegistry *BatchRegistryCaller) TotalBatches(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "totalBatches")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalBatches is a free data retrieval call binding the contract method 0x69ff6abb.
//
// Solidity: function totalBatches() view returns(uint256)
func (_BatchRegistry *BatchRegistrySession) TotalBatches() (*big.Int, error) {
	return _BatchRegistry.Contract.TotalBatches(&_BatchRegistry.CallOpts)
}

// TotalBatches is a free data retrieval call binding the contract method 0x69ff6abb.
//
// Solidity: function totalBatches() view returns(uint256)
func (_BatchRegistry *BatchRegistryCallerSession) TotalBatches() (*big.Int, error) {
	return _BatchRegistry.Contract.TotalBatches(&_BatchRegistry.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_BatchRegistry *BatchRegistryCaller) Verifier(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BatchRegistry.contract.Call(opts, &out, "verifier")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_BatchRegistry *BatchRegistrySession) Verifier() (common.Address, error) {
	return _BatchRegistry.Contract.Verifier(&_BatchRegistry.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_BatchRegistry *BatchRegistryCallerSession) Verifier() (common.Address, error) {
	return _BatchRegistry.Contract.Verifier(&_BatchRegistry.CallOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_BatchRegistry *BatchRegistryTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BatchRegistry.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_BatchRegistry *BatchRegistrySession) Pause() (*types.Transaction, error) {
	return _BatchRegistry.Contract.Pause(&_BatchRegistry.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_BatchRegistry *BatchRegistryTransactorSession) Pause() (*types.Transaction, error) {
	return _BatchRegistry.Contract.Pause(&_BatchRegistry.TransactOpts)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x79606d29.
//
// Solidity: function registerBatch(bytes32 batchHash, bytes32 newStateRoot, bytes batchData, uint256[2] proofA, uint256[2][2] proofB, uint256[2] proofC, uint256[6] publicInputs) returns(uint256 batchNumber)
func (_BatchRegistry *BatchRegistryTransactor) RegisterBatch(opts *bind.TransactOpts, batchHash [32]byte, newStateRoot [32]byte, batchData []byte, proofA [2]*big.Int, proofB [2][2]*big.Int, proofC [2]*big.Int, publicInputs [6]*big.Int) (*types.Transaction, error) {
	return _BatchRegistry.contract.Transact(opts, "registerBatch", batchHash, newStateRoot, batchData, proofA, proofB, proofC, publicInputs)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x79606d29.
//
// Solidity: function registerBatch(bytes32 batchHash, bytes32 newStateRoot, bytes batchData, uint256[2] proofA, uint256[2][2] proofB, uint256[2] proofC, uint256[6] publicInputs) returns(uint256 batchNumber)
func (_BatchRegistry *BatchRegistrySession) RegisterBatch(batchHash [32]byte, newStateRoot [32]byte, batchData []byte, proofA [2]*big.Int, proofB [2][2]*big.Int, proofC [2]*big.Int, publicInputs [6]*big.Int) (*types.Transaction, error) {
	return _BatchRegistry.Contract.RegisterBatch(&_BatchRegistry.TransactOpts, batchHash, newStateRoot, batchData, proofA, proofB, proofC, publicInputs)
}

// RegisterBatch is a paid mutator transaction binding the contract method 0x79606d29.
//
// Solidity: function registerBatch(bytes32 batchHash, bytes32 newStateRoot, bytes batchData, uint256[2] proofA, uint256[2][2] proofB, uint256[2] proofC, uint256[6] publicInputs) returns(uint256 batchNumber)
func (_BatchRegistry *BatchRegistryTransactorSession) RegisterBatch(batchHash [32]byte, newStateRoot [32]byte, batchData []byte, proofA [2]*big.Int, proofB [2][2]*big.Int, proofC [2]*big.Int, publicInputs [6]*big.Int) (*types.Transaction, error) {
	return _BatchRegistry.Contract.RegisterBatch(&_BatchRegistry.TransactOpts, batchHash, newStateRoot, batchData, proofA, proofB, proofC, publicInputs)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BatchRegistry *BatchRegistryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _BatchRegistry.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BatchRegistry *BatchRegistrySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BatchRegistry.Contract.TransferOwnership(&_BatchRegistry.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BatchRegistry *BatchRegistryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BatchRegistry.Contract.TransferOwnership(&_BatchRegistry.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_BatchRegistry *BatchRegistryTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BatchRegistry.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_BatchRegistry *BatchRegistrySession) Unpause() (*types.Transaction, error) {
	return _BatchRegistry.Contract.Unpause(&_BatchRegistry.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_BatchRegistry *BatchRegistryTransactorSession) Unpause() (*types.Transaction, error) {
	return _BatchRegistry.Contract.Unpause(&_BatchRegistry.TransactOpts)
}

// UpdateIndexer is a paid mutator transaction binding the contract method 0x43ae20a3.
//
// Solidity: function updateIndexer(address newIndexer) returns()
func (_BatchRegistry *BatchRegistryTransactor) UpdateIndexer(opts *bind.TransactOpts, newIndexer common.Address) (*types.Transaction, error) {
	return _BatchRegistry.contract.Transact(opts, "updateIndexer", newIndexer)
}

// UpdateIndexer is a paid mutator transaction binding the contract method 0x43ae20a3.
//
// Solidity: function updateIndexer(address newIndexer) returns()
func (_BatchRegistry *BatchRegistrySession) UpdateIndexer(newIndexer common.Address) (*types.Transaction, error) {
	return _BatchRegistry.Contract.UpdateIndexer(&_BatchRegistry.TransactOpts, newIndexer)
}

// UpdateIndexer is a paid mutator transaction binding the contract method 0x43ae20a3.
//
// Solidity: function updateIndexer(address newIndexer) returns()
func (_BatchRegistry *BatchRegistryTransactorSession) UpdateIndexer(newIndexer common.Address) (*types.Transaction, error) {
	return _BatchRegistry.Contract.UpdateIndexer(&_BatchRegistry.TransactOpts, newIndexer)
}

// UpdateVerifier is a paid mutator transaction binding the contract method 0x97fc007c.
//
// Solidity: function updateVerifier(address newVerifier) returns()
func (_BatchRegistry *BatchRegistryTransactor) UpdateVerifier(opts *bind.TransactOpts, newVerifier common.Address) (*types.Transaction, error) {
	return _BatchRegistry.contract.Transact(opts, "updateVerifier", newVerifier)
}

// UpdateVerifier is a paid mutator transaction binding the contract method 0x97fc007c.
//
// Solidity: function updateVerifier(address newVerifier) returns()
func (_BatchRegistry *BatchRegistrySession) UpdateVerifier(newVerifier common.Address) (*types.Transaction, error) {
	return _BatchRegistry.Contract.UpdateVerifier(&_BatchRegistry.TransactOpts, newVerifier)
}

// UpdateVerifier is a paid mutator transaction binding the contract method 0x97fc007c.
//
// Solidity: function updateVerifier(address newVerifier) returns()
func (_BatchRegistry *BatchRegistryTransactorSession) UpdateVerifier(newVerifier common.Address) (*types.Transaction, error) {
	return _BatchRegistry.Contract.UpdateVerifier(&_BatchRegistry.TransactOpts, newVerifier)
}

// BatchRegistryBatchFinalizedIterator is returned from FilterBatchFinalized and is used to iterate over the raw logs and unpacked data for BatchFinalized events raised by the BatchRegistry contract.
type BatchRegistryBatchFinalizedIterator struct {
	Event *BatchRegistryBatchFinalized // Event containing the contract specifics and raw log

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
func (it *BatchRegistryBatchFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryBatchFinalized)
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
		it.Event = new(BatchRegistryBatchFinalized)
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
func (it *BatchRegistryBatchFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryBatchFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryBatchFinalized represents a BatchFinalized event raised by the BatchRegistry contract.
type BatchRegistryBatchFinalized struct {
	BatchNumber  *big.Int
	NewStateRoot [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBatchFinalized is a free log retrieval operation binding the contract event 0xa3ea0288fb824a30cae7f7d72a9e887bd0926c40c8438c43b49cfca687b0df52.
//
// Solidity: event BatchFinalized(uint256 indexed batchNumber, bytes32 indexed newStateRoot)
func (_BatchRegistry *BatchRegistryFilterer) FilterBatchFinalized(opts *bind.FilterOpts, batchNumber []*big.Int, newStateRoot [][32]byte) (*BatchRegistryBatchFinalizedIterator, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var newStateRootRule []interface{}
	for _, newStateRootItem := range newStateRoot {
		newStateRootRule = append(newStateRootRule, newStateRootItem)
	}

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "BatchFinalized", batchNumberRule, newStateRootRule)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryBatchFinalizedIterator{contract: _BatchRegistry.contract, event: "BatchFinalized", logs: logs, sub: sub}, nil
}

// WatchBatchFinalized is a free log subscription operation binding the contract event 0xa3ea0288fb824a30cae7f7d72a9e887bd0926c40c8438c43b49cfca687b0df52.
//
// Solidity: event BatchFinalized(uint256 indexed batchNumber, bytes32 indexed newStateRoot)
func (_BatchRegistry *BatchRegistryFilterer) WatchBatchFinalized(opts *bind.WatchOpts, sink chan<- *BatchRegistryBatchFinalized, batchNumber []*big.Int, newStateRoot [][32]byte) (event.Subscription, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var newStateRootRule []interface{}
	for _, newStateRootItem := range newStateRoot {
		newStateRootRule = append(newStateRootRule, newStateRootItem)
	}

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "BatchFinalized", batchNumberRule, newStateRootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryBatchFinalized)
				if err := _BatchRegistry.contract.UnpackLog(event, "BatchFinalized", log); err != nil {
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

// ParseBatchFinalized is a log parse operation binding the contract event 0xa3ea0288fb824a30cae7f7d72a9e887bd0926c40c8438c43b49cfca687b0df52.
//
// Solidity: event BatchFinalized(uint256 indexed batchNumber, bytes32 indexed newStateRoot)
func (_BatchRegistry *BatchRegistryFilterer) ParseBatchFinalized(log types.Log) (*BatchRegistryBatchFinalized, error) {
	event := new(BatchRegistryBatchFinalized)
	if err := _BatchRegistry.contract.UnpackLog(event, "BatchFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryBatchRegisteredIterator is returned from FilterBatchRegistered and is used to iterate over the raw logs and unpacked data for BatchRegistered events raised by the BatchRegistry contract.
type BatchRegistryBatchRegisteredIterator struct {
	Event *BatchRegistryBatchRegistered // Event containing the contract specifics and raw log

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
func (it *BatchRegistryBatchRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryBatchRegistered)
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
		it.Event = new(BatchRegistryBatchRegistered)
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
func (it *BatchRegistryBatchRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryBatchRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryBatchRegistered represents a BatchRegistered event raised by the BatchRegistry contract.
type BatchRegistryBatchRegistered struct {
	BatchNumber  *big.Int
	BatchHash    [32]byte
	OldStateRoot [32]byte
	NewStateRoot [32]byte
	Submitter    common.Address
	Timestamp    *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterBatchRegistered is a free log retrieval operation binding the contract event 0x36f38af8b825f8e8eaf3597076e8d73633e224ace519e83b5b79071b10dabb5d.
//
// Solidity: event BatchRegistered(uint256 indexed batchNumber, bytes32 indexed batchHash, bytes32 oldStateRoot, bytes32 newStateRoot, address indexed submitter, uint256 timestamp)
func (_BatchRegistry *BatchRegistryFilterer) FilterBatchRegistered(opts *bind.FilterOpts, batchNumber []*big.Int, batchHash [][32]byte, submitter []common.Address) (*BatchRegistryBatchRegisteredIterator, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var batchHashRule []interface{}
	for _, batchHashItem := range batchHash {
		batchHashRule = append(batchHashRule, batchHashItem)
	}

	var submitterRule []interface{}
	for _, submitterItem := range submitter {
		submitterRule = append(submitterRule, submitterItem)
	}

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "BatchRegistered", batchNumberRule, batchHashRule, submitterRule)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryBatchRegisteredIterator{contract: _BatchRegistry.contract, event: "BatchRegistered", logs: logs, sub: sub}, nil
}

// WatchBatchRegistered is a free log subscription operation binding the contract event 0x36f38af8b825f8e8eaf3597076e8d73633e224ace519e83b5b79071b10dabb5d.
//
// Solidity: event BatchRegistered(uint256 indexed batchNumber, bytes32 indexed batchHash, bytes32 oldStateRoot, bytes32 newStateRoot, address indexed submitter, uint256 timestamp)
func (_BatchRegistry *BatchRegistryFilterer) WatchBatchRegistered(opts *bind.WatchOpts, sink chan<- *BatchRegistryBatchRegistered, batchNumber []*big.Int, batchHash [][32]byte, submitter []common.Address) (event.Subscription, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var batchHashRule []interface{}
	for _, batchHashItem := range batchHash {
		batchHashRule = append(batchHashRule, batchHashItem)
	}

	var submitterRule []interface{}
	for _, submitterItem := range submitter {
		submitterRule = append(submitterRule, submitterItem)
	}

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "BatchRegistered", batchNumberRule, batchHashRule, submitterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryBatchRegistered)
				if err := _BatchRegistry.contract.UnpackLog(event, "BatchRegistered", log); err != nil {
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

// ParseBatchRegistered is a log parse operation binding the contract event 0x36f38af8b825f8e8eaf3597076e8d73633e224ace519e83b5b79071b10dabb5d.
//
// Solidity: event BatchRegistered(uint256 indexed batchNumber, bytes32 indexed batchHash, bytes32 oldStateRoot, bytes32 newStateRoot, address indexed submitter, uint256 timestamp)
func (_BatchRegistry *BatchRegistryFilterer) ParseBatchRegistered(log types.Log) (*BatchRegistryBatchRegistered, error) {
	event := new(BatchRegistryBatchRegistered)
	if err := _BatchRegistry.contract.UnpackLog(event, "BatchRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryFinalizationDelayUpdatedIterator is returned from FilterFinalizationDelayUpdated and is used to iterate over the raw logs and unpacked data for FinalizationDelayUpdated events raised by the BatchRegistry contract.
type BatchRegistryFinalizationDelayUpdatedIterator struct {
	Event *BatchRegistryFinalizationDelayUpdated // Event containing the contract specifics and raw log

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
func (it *BatchRegistryFinalizationDelayUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryFinalizationDelayUpdated)
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
		it.Event = new(BatchRegistryFinalizationDelayUpdated)
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
func (it *BatchRegistryFinalizationDelayUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryFinalizationDelayUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryFinalizationDelayUpdated represents a FinalizationDelayUpdated event raised by the BatchRegistry contract.
type BatchRegistryFinalizationDelayUpdated struct {
	OldDelay *big.Int
	NewDelay *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterFinalizationDelayUpdated is a free log retrieval operation binding the contract event 0x7173eb35e518994352ac74bad54f9568372e65efdff1f65e3dc3fc3d330e1994.
//
// Solidity: event FinalizationDelayUpdated(uint256 oldDelay, uint256 newDelay)
func (_BatchRegistry *BatchRegistryFilterer) FilterFinalizationDelayUpdated(opts *bind.FilterOpts) (*BatchRegistryFinalizationDelayUpdatedIterator, error) {

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "FinalizationDelayUpdated")
	if err != nil {
		return nil, err
	}
	return &BatchRegistryFinalizationDelayUpdatedIterator{contract: _BatchRegistry.contract, event: "FinalizationDelayUpdated", logs: logs, sub: sub}, nil
}

// WatchFinalizationDelayUpdated is a free log subscription operation binding the contract event 0x7173eb35e518994352ac74bad54f9568372e65efdff1f65e3dc3fc3d330e1994.
//
// Solidity: event FinalizationDelayUpdated(uint256 oldDelay, uint256 newDelay)
func (_BatchRegistry *BatchRegistryFilterer) WatchFinalizationDelayUpdated(opts *bind.WatchOpts, sink chan<- *BatchRegistryFinalizationDelayUpdated) (event.Subscription, error) {

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "FinalizationDelayUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryFinalizationDelayUpdated)
				if err := _BatchRegistry.contract.UnpackLog(event, "FinalizationDelayUpdated", log); err != nil {
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

// ParseFinalizationDelayUpdated is a log parse operation binding the contract event 0x7173eb35e518994352ac74bad54f9568372e65efdff1f65e3dc3fc3d330e1994.
//
// Solidity: event FinalizationDelayUpdated(uint256 oldDelay, uint256 newDelay)
func (_BatchRegistry *BatchRegistryFilterer) ParseFinalizationDelayUpdated(log types.Log) (*BatchRegistryFinalizationDelayUpdated, error) {
	event := new(BatchRegistryFinalizationDelayUpdated)
	if err := _BatchRegistry.contract.UnpackLog(event, "FinalizationDelayUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the BatchRegistry contract.
type BatchRegistryOwnershipTransferredIterator struct {
	Event *BatchRegistryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *BatchRegistryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryOwnershipTransferred)
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
		it.Event = new(BatchRegistryOwnershipTransferred)
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
func (it *BatchRegistryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryOwnershipTransferred represents a OwnershipTransferred event raised by the BatchRegistry contract.
type BatchRegistryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BatchRegistry *BatchRegistryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*BatchRegistryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryOwnershipTransferredIterator{contract: _BatchRegistry.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BatchRegistry *BatchRegistryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BatchRegistryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryOwnershipTransferred)
				if err := _BatchRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BatchRegistry *BatchRegistryFilterer) ParseOwnershipTransferred(log types.Log) (*BatchRegistryOwnershipTransferred, error) {
	event := new(BatchRegistryOwnershipTransferred)
	if err := _BatchRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the BatchRegistry contract.
type BatchRegistryPausedIterator struct {
	Event *BatchRegistryPaused // Event containing the contract specifics and raw log

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
func (it *BatchRegistryPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryPaused)
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
		it.Event = new(BatchRegistryPaused)
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
func (it *BatchRegistryPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryPaused represents a Paused event raised by the BatchRegistry contract.
type BatchRegistryPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_BatchRegistry *BatchRegistryFilterer) FilterPaused(opts *bind.FilterOpts) (*BatchRegistryPausedIterator, error) {

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &BatchRegistryPausedIterator{contract: _BatchRegistry.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_BatchRegistry *BatchRegistryFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *BatchRegistryPaused) (event.Subscription, error) {

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryPaused)
				if err := _BatchRegistry.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_BatchRegistry *BatchRegistryFilterer) ParsePaused(log types.Log) (*BatchRegistryPaused, error) {
	event := new(BatchRegistryPaused)
	if err := _BatchRegistry.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryIndexerUpdatedIterator is returned from FilterIndexerUpdated and is used to iterate over the raw logs and unpacked data for IndexerUpdated events raised by the BatchRegistry contract.
type BatchRegistryIndexerUpdatedIterator struct {
	Event *BatchRegistryIndexerUpdated // Event containing the contract specifics and raw log

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
func (it *BatchRegistryIndexerUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryIndexerUpdated)
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
		it.Event = new(BatchRegistryIndexerUpdated)
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
func (it *BatchRegistryIndexerUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryIndexerUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryIndexerUpdated represents a IndexerUpdated event raised by the BatchRegistry contract.
type BatchRegistryIndexerUpdated struct {
	OldIndexer common.Address
	NewIndexer common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterIndexerUpdated is a free log retrieval operation binding the contract event 0xcd58b762453bd126b48db83f2cecd464f5281dd7e5e6824b528c09d0482984d6.
//
// Solidity: event IndexerUpdated(address indexed oldIndexer, address indexed newIndexer)
func (_BatchRegistry *BatchRegistryFilterer) FilterIndexerUpdated(opts *bind.FilterOpts, oldIndexer []common.Address, newIndexer []common.Address) (*BatchRegistryIndexerUpdatedIterator, error) {

	var oldIndexerRule []interface{}
	for _, oldIndexerItem := range oldIndexer {
		oldIndexerRule = append(oldIndexerRule, oldIndexerItem)
	}
	var newIndexerRule []interface{}
	for _, newIndexerItem := range newIndexer {
		newIndexerRule = append(newIndexerRule, newIndexerItem)
	}

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "IndexerUpdated", oldIndexerRule, newIndexerRule)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryIndexerUpdatedIterator{contract: _BatchRegistry.contract, event: "IndexerUpdated", logs: logs, sub: sub}, nil
}

// WatchIndexerUpdated is a free log subscription operation binding the contract event 0xcd58b762453bd126b48db83f2cecd464f5281dd7e5e6824b528c09d0482984d6.
//
// Solidity: event IndexerUpdated(address indexed oldIndexer, address indexed newIndexer)
func (_BatchRegistry *BatchRegistryFilterer) WatchIndexerUpdated(opts *bind.WatchOpts, sink chan<- *BatchRegistryIndexerUpdated, oldIndexer []common.Address, newIndexer []common.Address) (event.Subscription, error) {

	var oldIndexerRule []interface{}
	for _, oldIndexerItem := range oldIndexer {
		oldIndexerRule = append(oldIndexerRule, oldIndexerItem)
	}
	var newIndexerRule []interface{}
	for _, newIndexerItem := range newIndexer {
		newIndexerRule = append(newIndexerRule, newIndexerItem)
	}

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "IndexerUpdated", oldIndexerRule, newIndexerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryIndexerUpdated)
				if err := _BatchRegistry.contract.UnpackLog(event, "IndexerUpdated", log); err != nil {
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

// ParseIndexerUpdated is a log parse operation binding the contract event 0xcd58b762453bd126b48db83f2cecd464f5281dd7e5e6824b528c09d0482984d6.
//
// Solidity: event IndexerUpdated(address indexed oldIndexer, address indexed newIndexer)
func (_BatchRegistry *BatchRegistryFilterer) ParseIndexerUpdated(log types.Log) (*BatchRegistryIndexerUpdated, error) {
	event := new(BatchRegistryIndexerUpdated)
	if err := _BatchRegistry.contract.UnpackLog(event, "IndexerUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the BatchRegistry contract.
type BatchRegistryUnpausedIterator struct {
	Event *BatchRegistryUnpaused // Event containing the contract specifics and raw log

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
func (it *BatchRegistryUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryUnpaused)
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
		it.Event = new(BatchRegistryUnpaused)
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
func (it *BatchRegistryUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryUnpaused represents a Unpaused event raised by the BatchRegistry contract.
type BatchRegistryUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_BatchRegistry *BatchRegistryFilterer) FilterUnpaused(opts *bind.FilterOpts) (*BatchRegistryUnpausedIterator, error) {

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &BatchRegistryUnpausedIterator{contract: _BatchRegistry.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_BatchRegistry *BatchRegistryFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *BatchRegistryUnpaused) (event.Subscription, error) {

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryUnpaused)
				if err := _BatchRegistry.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_BatchRegistry *BatchRegistryFilterer) ParseUnpaused(log types.Log) (*BatchRegistryUnpaused, error) {
	event := new(BatchRegistryUnpaused)
	if err := _BatchRegistry.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BatchRegistryVerifierUpdatedIterator is returned from FilterVerifierUpdated and is used to iterate over the raw logs and unpacked data for VerifierUpdated events raised by the BatchRegistry contract.
type BatchRegistryVerifierUpdatedIterator struct {
	Event *BatchRegistryVerifierUpdated // Event containing the contract specifics and raw log

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
func (it *BatchRegistryVerifierUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BatchRegistryVerifierUpdated)
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
		it.Event = new(BatchRegistryVerifierUpdated)
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
func (it *BatchRegistryVerifierUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BatchRegistryVerifierUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BatchRegistryVerifierUpdated represents a VerifierUpdated event raised by the BatchRegistry contract.
type BatchRegistryVerifierUpdated struct {
	OldVerifier common.Address
	NewVerifier common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterVerifierUpdated is a free log retrieval operation binding the contract event 0x0243549a92b2412f7a3caf7a2e56d65b8821b91345363faa5f57195384065fcc.
//
// Solidity: event VerifierUpdated(address indexed oldVerifier, address indexed newVerifier)
func (_BatchRegistry *BatchRegistryFilterer) FilterVerifierUpdated(opts *bind.FilterOpts, oldVerifier []common.Address, newVerifier []common.Address) (*BatchRegistryVerifierUpdatedIterator, error) {

	var oldVerifierRule []interface{}
	for _, oldVerifierItem := range oldVerifier {
		oldVerifierRule = append(oldVerifierRule, oldVerifierItem)
	}
	var newVerifierRule []interface{}
	for _, newVerifierItem := range newVerifier {
		newVerifierRule = append(newVerifierRule, newVerifierItem)
	}

	logs, sub, err := _BatchRegistry.contract.FilterLogs(opts, "VerifierUpdated", oldVerifierRule, newVerifierRule)
	if err != nil {
		return nil, err
	}
	return &BatchRegistryVerifierUpdatedIterator{contract: _BatchRegistry.contract, event: "VerifierUpdated", logs: logs, sub: sub}, nil
}

// WatchVerifierUpdated is a free log subscription operation binding the contract event 0x0243549a92b2412f7a3caf7a2e56d65b8821b91345363faa5f57195384065fcc.
//
// Solidity: event VerifierUpdated(address indexed oldVerifier, address indexed newVerifier)
func (_BatchRegistry *BatchRegistryFilterer) WatchVerifierUpdated(opts *bind.WatchOpts, sink chan<- *BatchRegistryVerifierUpdated, oldVerifier []common.Address, newVerifier []common.Address) (event.Subscription, error) {

	var oldVerifierRule []interface{}
	for _, oldVerifierItem := range oldVerifier {
		oldVerifierRule = append(oldVerifierRule, oldVerifierItem)
	}
	var newVerifierRule []interface{}
	for _, newVerifierItem := range newVerifier {
		newVerifierRule = append(newVerifierRule, newVerifierItem)
	}

	logs, sub, err := _BatchRegistry.contract.WatchLogs(opts, "VerifierUpdated", oldVerifierRule, newVerifierRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BatchRegistryVerifierUpdated)
				if err := _BatchRegistry.contract.UnpackLog(event, "VerifierUpdated", log); err != nil {
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

// ParseVerifierUpdated is a log parse operation binding the contract event 0x0243549a92b2412f7a3caf7a2e56d65b8821b91345363faa5f57195384065fcc.
//
// Solidity: event VerifierUpdated(address indexed oldVerifier, address indexed newVerifier)
func (_BatchRegistry *BatchRegistryFilterer) ParseVerifierUpdated(log types.Log) (*BatchRegistryVerifierUpdated, error) {
	event := new(BatchRegistryVerifierUpdated)
	if err := _BatchRegistry.contract.UnpackLog(event, "VerifierUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
