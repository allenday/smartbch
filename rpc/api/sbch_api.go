package api

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
	sbchapi "github.com/smartbch/smartbch/api"
	rpctypes "github.com/smartbch/smartbch/rpc/internal/ethapi"
)

var _ SbchAPI = (*sbchAPI)(nil)

type SbchAPI interface {
	GetStandbyTxQueue()
	QueryTxBySrc(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error)
	QueryTxByDst(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error)
	QueryTxByAddr(addr gethcmn.Address, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error)
	QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash, startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*gethtypes.Log, error)
	GetTxListByHeight(height gethrpc.BlockNumber) ([]map[string]interface{}, error)
	GetAddressCount(kind string, addr gethcmn.Address) hexutil.Uint64
	GetSep20AddressCount(kind string, contract, addr gethcmn.Address) hexutil.Uint64
}

type sbchAPI struct {
	backend sbchapi.BackendService
}

func newSbchAPI(backend sbchapi.BackendService) SbchAPI {
	return sbchAPI{backend: backend}
}

func (sbch sbchAPI) GetStandbyTxQueue() {
	panic("implement me")
}

func (sbch sbchAPI) GetTxListByHeight(height gethrpc.BlockNumber) ([]map[string]interface{}, error) {
	if height == gethrpc.LatestBlockNumber {
		height = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	txs, err := sbch.backend.GetTxListByHeight(uint32(height))
	if err != nil {
		return nil, err
	}
	return txsToReceiptRpcResp(txs), nil
}

func (sbch sbchAPI) QueryTxBySrc(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error) {

	_start, _end := sbch.prepareHeightRange(startHeight, endHeight)
	txs, err := sbch.backend.QueryTxBySrc(addr, _start, _end, uint32(limit))
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (sbch sbchAPI) QueryTxByDst(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error) {

	_start, _end := sbch.prepareHeightRange(startHeight, endHeight)
	txs, err := sbch.backend.QueryTxByDst(addr, _start, _end, uint32(limit))
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (sbch sbchAPI) QueryTxByAddr(addr gethcmn.Address,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*rpctypes.Transaction, error) {

	_start, _end := sbch.prepareHeightRange(startHeight, endHeight)
	txs, err := sbch.backend.QueryTxByAddr(addr, _start, _end, uint32(limit))
	if err != nil {
		return nil, err
	}
	return txsToRpcResp(txs), nil
}

func (sbch sbchAPI) prepareHeightRange(startHeight, endHeight gethrpc.BlockNumber) (uint32, uint32) {
	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	if startHeight > endHeight {
		startHeight++
	} else {
		endHeight++
	}
	return uint32(startHeight), uint32(endHeight)
}

func (sbch sbchAPI) QueryLogs(addr gethcmn.Address, topics []gethcmn.Hash,
	startHeight, endHeight gethrpc.BlockNumber, limit hexutil.Uint64) ([]*gethtypes.Log, error) {

	if startHeight == gethrpc.LatestBlockNumber {
		startHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}
	if endHeight == gethrpc.LatestBlockNumber {
		endHeight = gethrpc.BlockNumber(sbch.backend.LatestHeight())
	}

	logs, err := sbch.backend.SbchQueryLogs(addr, topics,
		uint32(startHeight), uint32(endHeight), uint32(limit))
	if err != nil {
		return nil, err
	}
	return motypes.ToGethLogs(logs), nil
}

func (sbch sbchAPI) GetAddressCount(kind string, addr gethcmn.Address) hexutil.Uint64 {
	fromCount, toCount := int64(0), int64(0)
	if kind == "from" || kind == "both" {
		fromCount = sbch.backend.GetFromAddressCount(addr)
	}
	if kind == "to" || kind == "both" {
		toCount = sbch.backend.GetToAddressCount(addr)
	}
	if kind == "from" {
		return hexutil.Uint64(fromCount)
	} else if kind == "to" {
		return hexutil.Uint64(toCount)
	} else if kind == "both" {
		return hexutil.Uint64(fromCount + toCount)
	}
	return hexutil.Uint64(0)
}

func (sbch sbchAPI) GetSep20AddressCount(kind string, contract, addr gethcmn.Address) hexutil.Uint64 {
	fromCount, toCount := int64(0), int64(0)
	if kind == "from" || kind == "both" {
		fromCount = sbch.backend.GetSep20FromAddressCount(contract, addr)
	}
	if kind == "to" || kind == "both" {
		toCount = sbch.backend.GetSep20ToAddressCount(contract, addr)
	}
	if kind == "from" {
		return hexutil.Uint64(fromCount)
	} else if kind == "to" {
		return hexutil.Uint64(toCount)
	} else if kind == "both" {
		return hexutil.Uint64(fromCount + toCount)
	}
	return hexutil.Uint64(0)
}
