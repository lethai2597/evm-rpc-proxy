package client

import (
	"bytes"
	"goevm/evm_proxy/client/throttle"
	"io/ioutil"
	"net/http"
	"time"

	"encoding/json"
)

func (this *EVMClient) RequestForward(body []byte) (ResponseType, []byte) {
	var method string

	// Attempt to unmarshal the body to an empty interface
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_json_decode++
		this.stat_last_60[this.stat_last_60_pos].stat_error_json_decode++
		this.mu.Unlock()
		return R_ERROR, []byte(`{"error":"json unmarshal error"}`)
	}

	// Check the type of the decoded value
	switch jsonData.(type) {
	case []interface{}:
		// JSON array - EVM batch requests
		jsonArray := jsonData.([]interface{})
		if len(jsonArray) > 0 {
			// Take the method from the first item of the array
			if obj, ok := jsonArray[0].(map[string]interface{}); ok {
				if m, ok := obj["method"].(string); ok {
					method = m
				}
			}
		}
	case map[string]interface{}:
		// JSON object - standard EVM request
		jsonObject := jsonData.(map[string]interface{})
		if m, ok := jsonObject["method"].(string); ok {
			method = m
		}
	default:
		// Neither object nor array, return error
		return R_ERROR, []byte(`{"error":"invalid json format"}`)
	}

	// If method is still empty, return error
	if len(method) == 0 {
		return R_ERROR, []byte(`{"error":"method not found in json"}`)
	}

	this.mu.Lock()
	// Check if client is paused or disabled
	if this.is_paused || this.is_disabled {
		this.mu.Unlock()
		return R_ERROR, []byte(`{"error":"node is paused or disabled"}`)
	}

	// Check if client is throttled
	if throttle.ThrottleGoup(this.throttle).GetThrottleScore().Throttled {
		this.mu.Unlock()
		return R_THROTTLED, []byte(`{"error":"throttled"}`)
	}
	throttle.ThrottleGoup(this.throttle).OnRequest(method)

	// Update stats
	this.stat_total.stat_done++
	this.stat_last_60[this.stat_last_60_pos].stat_done++
	this.stat_total.stat_request_by_fn[method]++
	this.stat_last_60[this.stat_last_60_pos].stat_request_by_fn[method]++
	this.stat_running++
	this.stat_total.stat_bytes_sent += len(body)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_sent += len(body)
	this.mu.Unlock()

	// Make the request
	now := time.Now().UnixNano()
	respBody := this._docall(now, body)
	if respBody == nil {
		return R_ERROR, []byte(`{"error":"request failed"}`)
	}

	return R_OK, respBody
}

func (this *EVMClient) RequestBasic(method_param ...string) ([]byte, ResponseType) {
	ts_started := time.Now().UnixNano()

	// Check if client is paused or disabled
	this.mu.Lock()
	if this.is_paused || this.is_disabled {
		this.mu.Unlock()
		return nil, R_ERROR
	}

	// THROTTLE BLOCK! Check if we're not throttled
	if throttle.ThrottleGoup(this.throttle).GetThrottleScore().Throttled {
		this.mu.Unlock()
		return nil, R_THROTTLED
	}

	// Prepare the request body
	var post []byte
	if len(method_param) == 1 {
		// If a full JSON-RPC request is provided
		post = []byte(method_param[0])
	} else {
		// If we need to construct a JSON-RPC request
		method := "eth_blockNumber" // Default method
		if len(method_param) > 0 {
			method = method_param[0]
		}

		params := []interface{}{}
		if len(method_param) > 1 {
			// Parse additional parameters
			for _, p := range method_param[1:] {
				params = append(params, p)
			}
		}

		// Create JSON-RPC request
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  method,
			"params":  params,
			"id":      1,
		}

		var err error
		post, err = json.Marshal(req)
		if err != nil {
			this.mu.Lock()
			this.stat_total.stat_error_json_marshal++
			this.stat_last_60[this.stat_last_60_pos].stat_error_json_marshal++
			this.mu.Unlock()
			return nil, R_ERROR
		}
	}

	// Update stats
	this.stat_total.stat_bytes_sent += len(post)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_sent += len(post)
	this.mu.Unlock()

	// Make the request
	ret := this._docall(ts_started, post)
	if ret == nil {
		return nil, R_ERROR
	}

	return ret, R_OK
}

func (this *EVMClient) _docall(ts_started int64, post []byte) []byte {
	// Create request
	req, err := http.NewRequest("POST", this.endpoint, bytes.NewBuffer(post))
	if err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_req++
		this.stat_last_60[this.stat_last_60_pos].stat_error_req++
		this.stat_running--
		this._last_error = *isGenericError(err, post)
		this.mu.Unlock()
		return nil
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if this.header != nil {
		for k, v := range this.header {
			for _, vv := range v {
				req.Header.Add(k, vv)
			}
		}
	}

	// Make the request
	resp, err := this.client.Do(req)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		this.mu.Lock()
		this.stat_total.stat_error_resp++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp++
		this.stat_running--
		this._last_error = *isHTTPError(resp, err, post)
		this.mu.Unlock()
		return nil
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		this.mu.Lock()
		this.stat_total.stat_error_resp_read++
		this.stat_last_60[this.stat_last_60_pos].stat_error_resp_read++
		this.stat_running--
		this._last_error = *isGenericError(err, post)
		this.mu.Unlock()
		return nil
	}

	// Update stats
	this.mu.Lock()
	elapsed := time.Now().UnixNano() - ts_started
	this.stat_total.stat_ns_total += uint64(elapsed / 1000)
	this.stat_last_60[this.stat_last_60_pos].stat_ns_total += uint64(elapsed / 1000)
	this.stat_total.stat_bytes_received += len(body)
	this.stat_last_60[this.stat_last_60_pos].stat_bytes_received += len(body)
	this.stat_running--
	throttle.ThrottleGoup(this.throttle).OnReceive(len(body))
	this.mu.Unlock()

	return body
}
