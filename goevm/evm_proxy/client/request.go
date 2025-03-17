package client

/* This file adds support for requests on higher level, without error processing */

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func (this *EVMClient) GetVersion() (int, int, string, ResponseType) {
	// For EVM, we use eth_chainId instead of web3_clientVersion
	// since web3_clientVersion is not supported by some nodes
	ret, r_type := this.RequestBasic(`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`)
	if ret == nil {
		return 0, 0, "", r_type
	}

	r := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&r)

	if result, ok := r["result"].(string); ok {
		// Parse chain ID string
		chainName := "Unknown Chain"
		if strings.HasPrefix(result, "0x") {
			chainIdInt, err := strconv.ParseInt(result[2:], 16, 64)
			if err == nil {
				// Map common chain IDs to names
				switch chainIdInt {
				case 1:
					chainName = "Ethereum Mainnet"
				case 5:
					chainName = "Goerli Testnet"
				case 11155111:
					chainName = "Sepolia Testnet"
				case 56:
					chainName = "Binance Smart Chain"
				case 97:
					chainName = "BSC Testnet"
				case 137:
					chainName = "Polygon Mainnet"
				case 80001:
					chainName = "Polygon Mumbai"
				default:
					chainName = fmt.Sprintf("EVM Chain %d", chainIdInt)
				}
			}
		}

		// We don't have real version info, so use chainId as identifier
		versionMajor := 1    // Default major version
		versionMinor := 0    // Default minor version
		version := chainName // Use chain name as version string

		this.mu.Lock()
		this.version_major = versionMajor
		this.version_minor = versionMinor
		this.version = version
		this.version_ts = time.Now().UnixMilli()
		this.mu.Unlock()

		return versionMajor, versionMinor, version, R_OK
	}
	return 0, 0, "", R_ERROR
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
