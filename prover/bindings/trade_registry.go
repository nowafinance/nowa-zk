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

// TradeRegistryTrade is an auto generated low-level Go binding around an user-defined struct.
type TradeRegistryTrade struct {
	MessageHash [32]byte
	PubKeyX     *big.Int
	PubKeyY     *big.Int
}

// TradeRegistryMetaData contains all meta data concerning the TradeRegistry contract.
var TradeRegistryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_verifier\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"chunkBatchRoot\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isChunkVerified\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerTrades\",\"inputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"chunkIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"proof\",\"type\":\"uint256[8]\",\"internalType\":\"uint256[8]\"},{\"name\":\"commitments\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"commitmentPok\",\"type\":\"uint256[2]\",\"internalType\":\"uint256[2]\"},{\"name\":\"publicInputs\",\"type\":\"uint256[1]\",\"internalType\":\"uint256[1]\"},{\"name\":\"trades\",\"type\":\"tuple[]\",\"internalType\":\"structTradeRegistry.Trade[]\",\"components\":[{\"name\":\"messageHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"pubKeyX\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"pubKeyY\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"verifier\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractITradeVerifier\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"TradesSettled\",\"inputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"chunkIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"batchRoot\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"trades\",\"type\":\"tuple[]\",\"indexed\":false,\"internalType\":\"structTradeRegistry.Trade[]\",\"components\":[{\"name\":\"messageHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"pubKeyX\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"pubKeyY\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"TradesVerified\",\"inputs\":[{\"name\":\"batchNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"chunkIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
}

// TradeRegistryABI is the input ABI used to generate the binding from.
// Deprecated: Use TradeRegistryMetaData.ABI instead.
var TradeRegistryABI = TradeRegistryMetaData.ABI

// TradeRegistry is an auto generated Go binding around an Ethereum contract.
type TradeRegistry struct {
	TradeRegistryCaller     // Read-only binding to the contract
	TradeRegistryTransactor // Write-only binding to the contract
	TradeRegistryFilterer   // Log filterer for contract events
}

// TradeRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type TradeRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TradeRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TradeRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TradeRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TradeRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TradeRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TradeRegistrySession struct {
	Contract     *TradeRegistry    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TradeRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TradeRegistryCallerSession struct {
	Contract *TradeRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// TradeRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TradeRegistryTransactorSession struct {
	Contract     *TradeRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// TradeRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type TradeRegistryRaw struct {
	Contract *TradeRegistry // Generic contract binding to access the raw methods on
}

// TradeRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TradeRegistryCallerRaw struct {
	Contract *TradeRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// TradeRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TradeRegistryTransactorRaw struct {
	Contract *TradeRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTradeRegistry creates a new instance of TradeRegistry, bound to a specific deployed contract.
func NewTradeRegistry(address common.Address, backend bind.ContractBackend) (*TradeRegistry, error) {
	contract, err := bindTradeRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TradeRegistry{TradeRegistryCaller: TradeRegistryCaller{contract: contract}, TradeRegistryTransactor: TradeRegistryTransactor{contract: contract}, TradeRegistryFilterer: TradeRegistryFilterer{contract: contract}}, nil
}

// NewTradeRegistryCaller creates a new read-only instance of TradeRegistry, bound to a specific deployed contract.
func NewTradeRegistryCaller(address common.Address, caller bind.ContractCaller) (*TradeRegistryCaller, error) {
	contract, err := bindTradeRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TradeRegistryCaller{contract: contract}, nil
}

// NewTradeRegistryTransactor creates a new write-only instance of TradeRegistry, bound to a specific deployed contract.
func NewTradeRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*TradeRegistryTransactor, error) {
	contract, err := bindTradeRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TradeRegistryTransactor{contract: contract}, nil
}

// NewTradeRegistryFilterer creates a new log filterer instance of TradeRegistry, bound to a specific deployed contract.
func NewTradeRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*TradeRegistryFilterer, error) {
	contract, err := bindTradeRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TradeRegistryFilterer{contract: contract}, nil
}

// bindTradeRegistry binds a generic wrapper to an already deployed contract.
func bindTradeRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TradeRegistryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TradeRegistry *TradeRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TradeRegistry.Contract.TradeRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TradeRegistry *TradeRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TradeRegistry.Contract.TradeRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TradeRegistry *TradeRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TradeRegistry.Contract.TradeRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TradeRegistry *TradeRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TradeRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TradeRegistry *TradeRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TradeRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TradeRegistry *TradeRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TradeRegistry.Contract.contract.Transact(opts, method, params...)
}

// ChunkBatchRoot is a free data retrieval call binding the contract method 0x9c3597ae.
//
// Solidity: function chunkBatchRoot(uint256 , uint256 ) view returns(bytes32)
func (_TradeRegistry *TradeRegistryCaller) ChunkBatchRoot(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _TradeRegistry.contract.Call(opts, &out, "chunkBatchRoot", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ChunkBatchRoot is a free data retrieval call binding the contract method 0x9c3597ae.
//
// Solidity: function chunkBatchRoot(uint256 , uint256 ) view returns(bytes32)
func (_TradeRegistry *TradeRegistrySession) ChunkBatchRoot(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _TradeRegistry.Contract.ChunkBatchRoot(&_TradeRegistry.CallOpts, arg0, arg1)
}

// ChunkBatchRoot is a free data retrieval call binding the contract method 0x9c3597ae.
//
// Solidity: function chunkBatchRoot(uint256 , uint256 ) view returns(bytes32)
func (_TradeRegistry *TradeRegistryCallerSession) ChunkBatchRoot(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _TradeRegistry.Contract.ChunkBatchRoot(&_TradeRegistry.CallOpts, arg0, arg1)
}

// IsChunkVerified is a free data retrieval call binding the contract method 0xfdfac173.
//
// Solidity: function isChunkVerified(uint256 , uint256 ) view returns(bool)
func (_TradeRegistry *TradeRegistryCaller) IsChunkVerified(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (bool, error) {
	var out []interface{}
	err := _TradeRegistry.contract.Call(opts, &out, "isChunkVerified", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsChunkVerified is a free data retrieval call binding the contract method 0xfdfac173.
//
// Solidity: function isChunkVerified(uint256 , uint256 ) view returns(bool)
func (_TradeRegistry *TradeRegistrySession) IsChunkVerified(arg0 *big.Int, arg1 *big.Int) (bool, error) {
	return _TradeRegistry.Contract.IsChunkVerified(&_TradeRegistry.CallOpts, arg0, arg1)
}

// IsChunkVerified is a free data retrieval call binding the contract method 0xfdfac173.
//
// Solidity: function isChunkVerified(uint256 , uint256 ) view returns(bool)
func (_TradeRegistry *TradeRegistryCallerSession) IsChunkVerified(arg0 *big.Int, arg1 *big.Int) (bool, error) {
	return _TradeRegistry.Contract.IsChunkVerified(&_TradeRegistry.CallOpts, arg0, arg1)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_TradeRegistry *TradeRegistryCaller) Verifier(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TradeRegistry.contract.Call(opts, &out, "verifier")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_TradeRegistry *TradeRegistrySession) Verifier() (common.Address, error) {
	return _TradeRegistry.Contract.Verifier(&_TradeRegistry.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_TradeRegistry *TradeRegistryCallerSession) Verifier() (common.Address, error) {
	return _TradeRegistry.Contract.Verifier(&_TradeRegistry.CallOpts)
}

// RegisterTrades is a paid mutator transaction binding the contract method 0xaebae4a1.
//
// Solidity: function registerTrades(uint256 batchNumber, uint256 chunkIndex, uint256[8] proof, uint256[2] commitments, uint256[2] commitmentPok, uint256[1] publicInputs, (bytes32,uint256,uint256)[] trades) returns()
func (_TradeRegistry *TradeRegistryTransactor) RegisterTrades(opts *bind.TransactOpts, batchNumber *big.Int, chunkIndex *big.Int, proof [8]*big.Int, commitments [2]*big.Int, commitmentPok [2]*big.Int, publicInputs [1]*big.Int, trades []TradeRegistryTrade) (*types.Transaction, error) {
	return _TradeRegistry.contract.Transact(opts, "registerTrades", batchNumber, chunkIndex, proof, commitments, commitmentPok, publicInputs, trades)
}

// RegisterTrades is a paid mutator transaction binding the contract method 0xaebae4a1.
//
// Solidity: function registerTrades(uint256 batchNumber, uint256 chunkIndex, uint256[8] proof, uint256[2] commitments, uint256[2] commitmentPok, uint256[1] publicInputs, (bytes32,uint256,uint256)[] trades) returns()
func (_TradeRegistry *TradeRegistrySession) RegisterTrades(batchNumber *big.Int, chunkIndex *big.Int, proof [8]*big.Int, commitments [2]*big.Int, commitmentPok [2]*big.Int, publicInputs [1]*big.Int, trades []TradeRegistryTrade) (*types.Transaction, error) {
	return _TradeRegistry.Contract.RegisterTrades(&_TradeRegistry.TransactOpts, batchNumber, chunkIndex, proof, commitments, commitmentPok, publicInputs, trades)
}

// RegisterTrades is a paid mutator transaction binding the contract method 0xaebae4a1.
//
// Solidity: function registerTrades(uint256 batchNumber, uint256 chunkIndex, uint256[8] proof, uint256[2] commitments, uint256[2] commitmentPok, uint256[1] publicInputs, (bytes32,uint256,uint256)[] trades) returns()
func (_TradeRegistry *TradeRegistryTransactorSession) RegisterTrades(batchNumber *big.Int, chunkIndex *big.Int, proof [8]*big.Int, commitments [2]*big.Int, commitmentPok [2]*big.Int, publicInputs [1]*big.Int, trades []TradeRegistryTrade) (*types.Transaction, error) {
	return _TradeRegistry.Contract.RegisterTrades(&_TradeRegistry.TransactOpts, batchNumber, chunkIndex, proof, commitments, commitmentPok, publicInputs, trades)
}

// TradeRegistryTradesSettledIterator is returned from FilterTradesSettled and is used to iterate over the raw logs and unpacked data for TradesSettled events raised by the TradeRegistry contract.
type TradeRegistryTradesSettledIterator struct {
	Event *TradeRegistryTradesSettled // Event containing the contract specifics and raw log

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
func (it *TradeRegistryTradesSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TradeRegistryTradesSettled)
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
		it.Event = new(TradeRegistryTradesSettled)
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
func (it *TradeRegistryTradesSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TradeRegistryTradesSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TradeRegistryTradesSettled represents a TradesSettled event raised by the TradeRegistry contract.
type TradeRegistryTradesSettled struct {
	BatchNumber *big.Int
	ChunkIndex  *big.Int
	BatchRoot   [32]byte
	Trades      []TradeRegistryTrade
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTradesSettled is a free log retrieval operation binding the contract event 0xd66d5184730ffc5b58571f567b324f2e19a6a2efc3defc3f92ef662580792fb1.
//
// Solidity: event TradesSettled(uint256 indexed batchNumber, uint256 indexed chunkIndex, bytes32 batchRoot, (bytes32,uint256,uint256)[] trades)
func (_TradeRegistry *TradeRegistryFilterer) FilterTradesSettled(opts *bind.FilterOpts, batchNumber []*big.Int, chunkIndex []*big.Int) (*TradeRegistryTradesSettledIterator, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var chunkIndexRule []interface{}
	for _, chunkIndexItem := range chunkIndex {
		chunkIndexRule = append(chunkIndexRule, chunkIndexItem)
	}

	logs, sub, err := _TradeRegistry.contract.FilterLogs(opts, "TradesSettled", batchNumberRule, chunkIndexRule)
	if err != nil {
		return nil, err
	}
	return &TradeRegistryTradesSettledIterator{contract: _TradeRegistry.contract, event: "TradesSettled", logs: logs, sub: sub}, nil
}

// WatchTradesSettled is a free log subscription operation binding the contract event 0xd66d5184730ffc5b58571f567b324f2e19a6a2efc3defc3f92ef662580792fb1.
//
// Solidity: event TradesSettled(uint256 indexed batchNumber, uint256 indexed chunkIndex, bytes32 batchRoot, (bytes32,uint256,uint256)[] trades)
func (_TradeRegistry *TradeRegistryFilterer) WatchTradesSettled(opts *bind.WatchOpts, sink chan<- *TradeRegistryTradesSettled, batchNumber []*big.Int, chunkIndex []*big.Int) (event.Subscription, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var chunkIndexRule []interface{}
	for _, chunkIndexItem := range chunkIndex {
		chunkIndexRule = append(chunkIndexRule, chunkIndexItem)
	}

	logs, sub, err := _TradeRegistry.contract.WatchLogs(opts, "TradesSettled", batchNumberRule, chunkIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TradeRegistryTradesSettled)
				if err := _TradeRegistry.contract.UnpackLog(event, "TradesSettled", log); err != nil {
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

// ParseTradesSettled is a log parse operation binding the contract event 0xd66d5184730ffc5b58571f567b324f2e19a6a2efc3defc3f92ef662580792fb1.
//
// Solidity: event TradesSettled(uint256 indexed batchNumber, uint256 indexed chunkIndex, bytes32 batchRoot, (bytes32,uint256,uint256)[] trades)
func (_TradeRegistry *TradeRegistryFilterer) ParseTradesSettled(log types.Log) (*TradeRegistryTradesSettled, error) {
	event := new(TradeRegistryTradesSettled)
	if err := _TradeRegistry.contract.UnpackLog(event, "TradesSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TradeRegistryTradesVerifiedIterator is returned from FilterTradesVerified and is used to iterate over the raw logs and unpacked data for TradesVerified events raised by the TradeRegistry contract.
type TradeRegistryTradesVerifiedIterator struct {
	Event *TradeRegistryTradesVerified // Event containing the contract specifics and raw log

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
func (it *TradeRegistryTradesVerifiedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TradeRegistryTradesVerified)
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
		it.Event = new(TradeRegistryTradesVerified)
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
func (it *TradeRegistryTradesVerifiedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TradeRegistryTradesVerifiedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TradeRegistryTradesVerified represents a TradesVerified event raised by the TradeRegistry contract.
type TradeRegistryTradesVerified struct {
	BatchNumber *big.Int
	ChunkIndex  *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTradesVerified is a free log retrieval operation binding the contract event 0xb6b6e54edd13a24686e0f7daee525bf7bb460fbdc9fe2ac49c8c46780e1153d9.
//
// Solidity: event TradesVerified(uint256 indexed batchNumber, uint256 indexed chunkIndex)
func (_TradeRegistry *TradeRegistryFilterer) FilterTradesVerified(opts *bind.FilterOpts, batchNumber []*big.Int, chunkIndex []*big.Int) (*TradeRegistryTradesVerifiedIterator, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var chunkIndexRule []interface{}
	for _, chunkIndexItem := range chunkIndex {
		chunkIndexRule = append(chunkIndexRule, chunkIndexItem)
	}

	logs, sub, err := _TradeRegistry.contract.FilterLogs(opts, "TradesVerified", batchNumberRule, chunkIndexRule)
	if err != nil {
		return nil, err
	}
	return &TradeRegistryTradesVerifiedIterator{contract: _TradeRegistry.contract, event: "TradesVerified", logs: logs, sub: sub}, nil
}

// WatchTradesVerified is a free log subscription operation binding the contract event 0xb6b6e54edd13a24686e0f7daee525bf7bb460fbdc9fe2ac49c8c46780e1153d9.
//
// Solidity: event TradesVerified(uint256 indexed batchNumber, uint256 indexed chunkIndex)
func (_TradeRegistry *TradeRegistryFilterer) WatchTradesVerified(opts *bind.WatchOpts, sink chan<- *TradeRegistryTradesVerified, batchNumber []*big.Int, chunkIndex []*big.Int) (event.Subscription, error) {

	var batchNumberRule []interface{}
	for _, batchNumberItem := range batchNumber {
		batchNumberRule = append(batchNumberRule, batchNumberItem)
	}
	var chunkIndexRule []interface{}
	for _, chunkIndexItem := range chunkIndex {
		chunkIndexRule = append(chunkIndexRule, chunkIndexItem)
	}

	logs, sub, err := _TradeRegistry.contract.WatchLogs(opts, "TradesVerified", batchNumberRule, chunkIndexRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TradeRegistryTradesVerified)
				if err := _TradeRegistry.contract.UnpackLog(event, "TradesVerified", log); err != nil {
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

// ParseTradesVerified is a log parse operation binding the contract event 0xb6b6e54edd13a24686e0f7daee525bf7bb460fbdc9fe2ac49c8c46780e1153d9.
//
// Solidity: event TradesVerified(uint256 indexed batchNumber, uint256 indexed chunkIndex)
func (_TradeRegistry *TradeRegistryFilterer) ParseTradesVerified(log types.Log) (*TradeRegistryTradesVerified, error) {
	event := new(TradeRegistryTradesVerified)
	if err := _TradeRegistry.contract.UnpackLog(event, "TradesVerified", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
