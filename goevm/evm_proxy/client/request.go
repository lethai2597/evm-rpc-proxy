package client

/* This file adds support for requests on higher level, without error processing */

import (
	"bytes"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

// ResponseType represents the type of response from the EVM node
type ResponseType int

const (
	R_OK        ResponseType = 0
	R_ERROR     ResponseType = 1
	R_THROTTLED ResponseType = 2
)

func (this *EVMClient) _intcall(method string) (int, ResponseType) {
	ret, r_type := this.RequestBasic(method)
	if ret == nil {
		return 0, r_type
	}

	r := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&r)

	switch v := r["result"].(type) {
	case json.Number:
		_ret, err := v.Int64()
		if err != nil {
			break
		}
		return int(_ret), r_type
	default:
		log.Printf("Error in response for %s: %s", method, string(ret))
	}
	return 0, R_ERROR
}

func (this *EVMClient) GetLastAvailableBlock() (int, ResponseType) {
	// For EVM, we use eth_blockNumber
	ret, r_type := this.RequestBasic(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`)
	if ret == nil {
		return 0, r_type
	}

	r := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&r)

	if result, ok := r["result"].(string); ok {
		// Convert hex string to int
		if strings.HasPrefix(result, "0x") {
			result = result[2:]
		}
		blockNum, err := strconv.ParseInt(result, 16, 64)
		if err == nil {
			_ts := time.Now().UnixMilli()
			this.mu.Lock()
			this.available_block_last = int(blockNum)
			this.available_block_last_ts = _ts
			this.mu.Unlock()
			return int(blockNum), R_OK
		}
	}
	return 0, R_ERROR
}
