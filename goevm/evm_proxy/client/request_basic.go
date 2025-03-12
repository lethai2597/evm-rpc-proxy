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

	// Check if client is throttled
	if throttle.ThrottleGoup(this.throttle).IsThrottled(method) {
		return R_THROTTLED, []byte(`{"error":"throttled"}`)
	}

	// Update stats
	this.mu.Lock()
	_pos := this.stat_last_60_pos
	this.stat_last_60[_pos].stat_request_by_fn[method]++
	this.stat_total.stat_request_by_fn[method]++
	this.stat_running++
	this.mu.Unlock()

	// Make the request
	ret, r_type := this.RequestBasic(string(body))
	if r_type != R_OK {
		return r_type, []byte(`{"error":"request failed"}`)
	}

	return R_OK, ret
}

func (this *EVMClient) RequestBasic(method_param ...string) ([]byte, ResponseType) {
	ts_started := time.Now().UnixNano()

	// Check if client is paused or disabled
	this.mu.Lock()
	if this.is_paused || this.is_disabled {
		this.mu.Unlock()
		return nil, R_ERROR
	}
	this.mu.Unlock()

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
			_pos := this.stat_last_60_pos
			this.stat_last_60[_pos].stat_error_json_marshal++
			this.stat_total.stat_error_json_marshal++
			this.mu.Unlock()
			return nil, R_ERROR
		}
	}

	// Check if throttled
	this.mu.Lock()
	_pos := this.stat_last_60_pos
	this.stat_last_60[_pos].stat_bytes_sent += len(post)
	this.stat_total.stat_bytes_sent += len(post)
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
		_pos := this.stat_last_60_pos
		this.stat_last_60[_pos].stat_error_req++
		this.stat_total.stat_error_req++
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
		_pos := this.stat_last_60_pos
		this.stat_last_60[_pos].stat_error_resp++
		this.stat_total.stat_error_resp++
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
		_pos := this.stat_last_60_pos
		this.stat_last_60[_pos].stat_error_resp_read++
		this.stat_total.stat_error_resp_read++
		this.stat_running--
		this._last_error = *isGenericError(err, post)
		this.mu.Unlock()
		return nil
	}

	// Update stats
	this.mu.Lock()
	_pos := this.stat_last_60_pos
	this.stat_last_60[_pos].stat_done++
	this.stat_total.stat_done++
	this.stat_last_60[_pos].stat_ns_total += uint64(time.Now().UnixNano() - ts_started)
	this.stat_total.stat_ns_total += uint64(time.Now().UnixNano() - ts_started)
	this.stat_last_60[_pos].stat_bytes_received += len(body)
	this.stat_total.stat_bytes_received += len(body)
	this.stat_running--
	this.mu.Unlock()

	return body
}
