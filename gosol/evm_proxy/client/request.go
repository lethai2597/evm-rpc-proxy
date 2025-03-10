package client

/* This file adds support for requests on higher level, without error processing */

import (
	"bytes"
	"fmt"
	"time"

	"encoding/json"
	"strconv"
	"strings"
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
		fmt.Println("Error in response for " + method + ": " + string(ret))
	}
	return 0, R_ERROR
}

func (this *EVMClient) GetFirstAvailableBlock() (int, ResponseType) {
	// For EVM, we use eth_getBlockByNumber with "earliest" parameter
	ret, r_type := this.RequestBasic(`{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["earliest", false],"id":1}`)
	if ret == nil {
		return 0, r_type
	}

	r := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&r)

	if result, ok := r["result"].(map[string]interface{}); ok {
		if number, ok := result["number"].(string); ok {
			// Convert hex string to int
			if strings.HasPrefix(number, "0x") {
				number = number[2:]
			}
			blockNum, err := strconv.ParseInt(number, 16, 64)
			if err == nil {
				_ts := time.Now().UnixMilli()
				this.mu.Lock()
				this.available_block_first = int(blockNum)
				this.available_block_first_ts = _ts
				this.mu.Unlock()
				return int(blockNum), R_OK
			}
		}
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

func (this *EVMClient) GetVersion() (int, int, string, ResponseType) {
	// For EVM, we use web3_clientVersion
	ret, r_type := this.RequestBasic(`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}`)
	if ret == nil {
		return 0, 0, "", r_type
	}

	r := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&r)

	if result, ok := r["result"].(string); ok {
		// Parse version string, format varies by client
		// Example: "Geth/v1.10.23-stable/linux-amd64/go1.18.5"
		parts := strings.Split(result, "/")
		clientName := parts[0]
		version := result
		versionMajor := 0
		versionMinor := 0

		if len(parts) > 1 && strings.HasPrefix(parts[1], "v") {
			versionStr := parts[1][1:] // Remove 'v' prefix
			versionParts := strings.Split(versionStr, ".")
			if len(versionParts) >= 2 {
				versionMajor, _ = strconv.Atoi(versionParts[0])
				versionMinor, _ = strconv.Atoi(versionParts[1])
			}
		}

		this.mu.Lock()
		this.version_major = versionMajor
		this.version_minor = versionMinor
		this.version = clientName + " " + version
		this.version_ts = time.Now().UnixMilli()
		this.mu.Unlock()

		return versionMajor, versionMinor, clientName + " " + version, R_OK
	}
	return 0, 0, "", R_ERROR
}
