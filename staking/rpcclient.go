package staking

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/smartbch/smartbch/staking/types"
)

const (
	ReqStrBlockCount = "{\"jsonrpc\": \"1.0\", \"id\":\"smartbch\", \"method\": \"getblockcount\", \"params\": [] }"
	ReqStrBlockHash  = "{\"jsonrpc\": \"1.0\", \"id\":\"smartbch\", \"method\": \"getblockhash\", \"params\": [%d] }"
	ReqStrBlock      = "{\"jsonrpc\": \"1.0\", \"id\":\"smartbch\", \"method\": \"getblock\", \"params\": [\"%s\"] }"
	ReqStrTx         = "{\"jsonrpc\": \"1.0\", \"id\":\"smartbch\", \"method\": \"getrawtransaction\", \"params\": [\"%s\", true] }"
	Identifier       = "73424348"
	Version          = "00"
)

type JsonRpcError struct {
	Code    int `json:"code"`
	Message int `json:"messsage"`
}

type BlockCountResp struct {
	Result int64         `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type BlockHashResp struct {
	Result string        `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type BlockInfo struct {
	Hash              string   `json:"hash"`
	Confirmations     int      `json:"confirmations"`
	Size              int      `json:"size"`
	Height            int64    `json:"height"`
	Version           int      `json:"version"`
	VersionHex        string   `json:"versionHex"`
	Merkleroot        string   `json:"merkleroot"`
	Tx                []string `json:"tx"`
	Time              int64    `json:"time"`
	MedianTime        int64    `json:"mediantime"`
	Nonce             int      `json:"nonce"`
	Bits              string   `json:"bits"`
	Difficulty        float64  `json:"difficulty"`
	Chainwork         string   `json:"chainwork"`
	NumTx             int      `json:"nTx"`
	PreviousBlockhash string   `json:"previousblockhash"`
}

type BlockInfoResp struct {
	Result BlockInfo     `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type CoinbaseVin struct {
	Coinbase string `json:"coinbase"`
	Sequence int    `json:"sequence"`
}

type Vout struct {
	Value        float64                `json:"value"`
	N            int                    `json:"n"`
	ScriptPubKey map[string]interface{} `json:"scriptPubKey"`
}

type TxInfo struct {
	TxID          string                   `json:"txid"`
	Hash          string                   `json:"hash"`
	Version       int                      `json:"version"`
	Size          int                      `json:"size"`
	Locktime      int                      `json:"locktime"`
	VinList       []map[string]interface{} `json:"vin"`
	VoutList      []Vout                   `json:"vout"`
	Hex           string                   `json:"hex"`
	Blockhash     string                   `json:"blockhash"`
	Confirmations int                      `json:"confirmations"`
	Time          int64                    `json:"time"`
	BlockTime     int64                    `json:"blocktime"`
}

func (ti TxInfo) GetValidatorPubKey() (pubKey [32]byte, ok bool) {
	for _, vout := range ti.VoutList {
		asm, ok := vout.ScriptPubKey["asm"]
		if !ok || asm == nil {
			continue
		}
		script, ok := asm.(string)
		if !ok {
			continue
		}
		prefix := "OP_RETURN " + Identifier + Version
		if !strings.HasPrefix(script, prefix) {
			continue
		}
		script = script[len(prefix):]
		if len(script) != 64 {
			continue
		}
		bz, err := hex.DecodeString(script)
		if err != nil {
			continue
		}
		copy(pubKey[:], bz)
		break
	}
	return
}

type TxInfoResp struct {
	Result TxInfo        `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type RpcClient struct {
	url      string
	user     string
	password string
	err      error
}

var _ types.RpcClient = (*RpcClient)(nil)

func NewRpcClient(url, user, password string) *RpcClient {
	return &RpcClient{
		url:      url,
		user:     user,
		password: password,
	}
}

func (client *RpcClient) sendRequest(reqStr string) ([]byte, error) {
	body := strings.NewReader(reqStr)
	req, err := http.NewRequest("POST", "http://127.0.0.1:8332/", body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(client.user, client.password)
	req.Header.Set("Content-Type", "text/plain;")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return respData, nil
}

func (client *RpcClient) GetLatestHeight() (height int64) {
	height, client.err = client.getCurrHeight()
	return
}

func (client *RpcClient) GetBlockByHeight(height int64) *types.BCHBlock {
	var hash string
	hash, client.err = client.getBlockHashOfHeight(height)
	if client.err != nil {
		return nil
	}
	return client.getBCHBlock(hash)
}

func (client *RpcClient) GetBlockByHash(hash [32]byte) *types.BCHBlock {
	return client.getBCHBlock(hex.EncodeToString(hash[:]))
}

func (client *RpcClient) getBCHBlock(hash string) *types.BCHBlock {
	var bi *BlockInfo
	bi, client.err = client.getBlock(hash)
	if client.err != nil {
		return nil
	}
	bchBlock := &types.BCHBlock{
		Height:    bi.Height,
		Timestamp: bi.Time,
	}
	var bz []byte
	bz, client.err = hex.DecodeString(bi.Hash)
	copy(bchBlock.HashId[:], bz)
	if client.err != nil {
		return nil
	}
	bz, client.err = hex.DecodeString(bi.PreviousBlockhash)
	copy(bchBlock.ParentBlk[:], bz)
	if client.err != nil {
		return nil
	}
	var coinbase *TxInfo
	coinbase, client.err = client.getTx(bi.Tx[0])
	if client.err != nil {
		return nil
	}
	pubKey, ok := coinbase.GetValidatorPubKey()
	if ok {
		nomination := types.Nomination{
			Pubkey:         pubKey,
			NominatedCount: 1,
		}
		bchBlock.Nominations = append(bchBlock.Nominations, nomination)
	}
	return bchBlock
}

func (client *RpcClient) getCurrHeight() (int64, error) {
	respData, err := client.sendRequest(ReqStrBlockCount)
	if err != nil {
		return -1, err
	}
	var blockCountResp BlockCountResp
	err = json.Unmarshal(respData, &blockCountResp)
	if err != nil {
		return -1, err
	}
	return blockCountResp.Result, nil
}

func (client *RpcClient) getBlockHashOfHeight(height int64) (string, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrBlockHash, height))
	if err != nil {
		return "", err
	}
	var blockHashResp BlockHashResp
	err = json.Unmarshal(respData, &blockHashResp)
	if err != nil {
		return "", err
	}
	return blockHashResp.Result, nil
}

func (client *RpcClient) getBlock(hash string) (*BlockInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrBlock, hash))
	if err != nil {
		return nil, err
	}
	//fmt.Printf("BLOCK %s\n", string(respData))
	var blockInfoResp BlockInfoResp
	err = json.Unmarshal(respData, &blockInfoResp)
	if err != nil {
		return nil, err
	}
	return &blockInfoResp.Result, nil
}

func (client *RpcClient) getTx(hash string) (*TxInfo, error) {
	respData, err := client.sendRequest(fmt.Sprintf(ReqStrTx, hash))
	if err != nil {
		return nil, err
	}
	//fmt.Printf("TX %s\n", string(respData))
	var txInfoResp TxInfoResp
	err = json.Unmarshal(respData, &txInfoResp)
	if err != nil {
		return nil, err
	}
	return &txInfoResp.Result, nil
}

func (client *RpcClient) PrintAllOpReturn(startHeight, endHeight int64) {
	for h := startHeight; h < endHeight; h++ {
		fmt.Printf("Height: %d\n", h)
		hash, err := client.getBlockHashOfHeight(h)
		if err != nil {
			fmt.Printf("Error when getBlockHashOfHeight %d %s\n", h, err.Error())
			continue
		}
		bi, err := client.getBlock(hash)
		if err != nil {
			fmt.Printf("Error when getBlock %d %s\n", h, err.Error())
			continue
		}
		for _, txid := range bi.Tx {
			tx, err := client.getTx(txid)
			if err != nil {
				fmt.Printf("Error when getTx %s %s\n", txid, err.Error())
				continue
			}
			for _, vout := range tx.VoutList {
				asm, ok := vout.ScriptPubKey["asm"]
				if !ok || asm == nil {
					continue
				}
				script, ok := asm.(string)
				if !ok {
					continue
				}
				if strings.HasPrefix(script, "OP_RETURN") {
					fmt.Println(script)
				}
			}
		}
	}
}
