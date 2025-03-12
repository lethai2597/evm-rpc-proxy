package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GetBlock fetches block information by number or hash
// blockIdentifier can be a block number or block hash (with 0x prefix)
// fullTx determines whether to include full transaction objects or just hashes
func (this *EVMClient) GetBlock(blockIdentifier string, fullTx bool) ([]byte, ResponseType) {
	method := "eth_getBlockByNumber"

	// If blockIdentifier starts with 0x, it's a block hash, otherwise it's a block number
	if strings.HasPrefix(blockIdentifier, "0x") {
		method = "eth_getBlockByHash"
	} else if blockIdentifier != "latest" && blockIdentifier != "earliest" && blockIdentifier != "pending" {
		// Convert to hex if it's a number
		blockIdentifier = fmt.Sprintf("0x%x", blockIdentifier)
	}

	params := fmt.Sprintf(`["%s", %t]`, blockIdentifier, fullTx)
	return this.RequestBasic(method, params)
}

// GetTransaction fetches transaction information by transaction hash
func (this *EVMClient) GetTransaction(txHash string) ([]byte, ResponseType) {
	params := fmt.Sprintf(`["%s"]`, txHash)
	return this.RequestBasic("eth_getTransactionByHash", params)
}

// GetBalance fetches the balance of an Ethereum address
// blockParam can be a block number, "latest", "earliest", or "pending"
func (this *EVMClient) GetBalance(address string, blockParam string) ([]byte, ResponseType) {
	if blockParam != "latest" && blockParam != "earliest" && blockParam != "pending" && !strings.HasPrefix(blockParam, "0x") {
		// Convert to hex if it's a number
		if _, err := fmt.Sscanf(blockParam, "%d", new(int)); err == nil {
			blockParam = fmt.Sprintf("0x%x", blockParam)
		} else {
			blockParam = "latest"
		}
	}

	params := fmt.Sprintf(`["%s", "%s"]`, address, blockParam)
	return this.RequestBasic("eth_getBalance", params)
}

// GetTokenInfo fetches basic information about an ERC20 token
// This combines several calls to get name, symbol, and totalSupply
func (this *EVMClient) GetTokenInfo(tokenAddress string) ([]byte, ResponseType) {
	// First check if contract exists
	codeParams := fmt.Sprintf(`["%s", "latest"]`, tokenAddress)
	code, codeResult := this.RequestBasic("eth_getCode", codeParams)

	if codeResult != R_OK {
		return code, codeResult
	}

	var codeResponse struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(code, &codeResponse); err != nil {
		return []byte(`{"error":"failed to parse response"}`), R_ERROR
	}

	// If code is empty (just "0x"), contract doesn't exist
	if codeResponse.Result == "0x" || codeResponse.Result == "" {
		return []byte(`{"error":"contract not found"}`), R_ERROR
	}

	// Call methods to get token info
	nameParam := fmt.Sprintf(`[{"to":"%s","data":"0x06fdde03"}, "latest"]`, tokenAddress)
	symbolParam := fmt.Sprintf(`[{"to":"%s","data":"0x95d89b41"}, "latest"]`, tokenAddress)
	supplyParam := fmt.Sprintf(`[{"to":"%s","data":"0x18160ddd"}, "latest"]`, tokenAddress)
	decimalsParam := fmt.Sprintf(`[{"to":"%s","data":"0x313ce567"}, "latest"]`, tokenAddress)

	// Get token name
	nameRes, nameResult := this.RequestBasic("eth_call", nameParam)
	if nameResult != R_OK {
		return nameRes, nameResult
	}

	// Get token symbol
	symbolRes, symbolResult := this.RequestBasic("eth_call", symbolParam)
	if symbolResult != R_OK {
		return symbolRes, symbolResult
	}

	// Get token total supply
	supplyRes, supplyResult := this.RequestBasic("eth_call", supplyParam)
	if supplyResult != R_OK {
		return supplyRes, supplyResult
	}

	// Get token decimals
	decimalsRes, decimalsResult := this.RequestBasic("eth_call", decimalsParam)
	if decimalsResult != R_OK {
		return decimalsRes, decimalsResult
	}

	// Parse responses
	var nameData, symbolData, supplyData, decimalsData struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(nameRes, &nameData); err != nil {
		nameData.Result = "0x"
	}

	if err := json.Unmarshal(symbolRes, &symbolData); err != nil {
		symbolData.Result = "0x"
	}

	if err := json.Unmarshal(supplyRes, &supplyData); err != nil {
		supplyData.Result = "0x0"
	}

	if err := json.Unmarshal(decimalsRes, &decimalsData); err != nil {
		decimalsData.Result = "0x0"
	}

	// Construct response
	result := fmt.Sprintf(`{
		"result": {
			"address": "%s",
			"name_hex": "%s",
			"symbol_hex": "%s",
			"totalSupply_hex": "%s",
			"decimals_hex": "%s"
		}
	}`, tokenAddress, nameData.Result, symbolData.Result, supplyData.Result, decimalsData.Result)

	return []byte(result), R_OK
}
