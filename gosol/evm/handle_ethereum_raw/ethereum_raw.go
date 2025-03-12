package handle_ethereum_raw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gosol/evm_proxy"
	"gosol/evm_proxy/client"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
)

type Handle_ethereum_raw struct {
}

func (this *Handle_ethereum_raw) Initialize() {
}

func (this *Handle_ethereum_raw) Info() string {
	return "This plugin will allow to issue raw ethereum requests"
}

func (this *Handle_ethereum_raw) GetActions() []string {
	return []string{"ethereumRaw"}
}

func (this *Handle_ethereum_raw) HandleAction(action string, data *handler_socket2.HSParams) string {

	// get first client!
	sch := evm_proxy.MakeScheduler()
	if data.GetParamI("public", 0) == 1 {
		sch.ForcePublic(true)
	}
	if data.GetParamI("private", 0) == 1 {
		sch.ForcePrivate(true)
	}
	cl := sch.GetAnyClient()
	if cl == nil {
		return `{"error":"can't find appropriate client"}`
	}

	// Hiển thị thông tin về node đang được sử dụng
	fmt.Printf("=== EVM RPC Request ===\n")
	fmt.Printf("Action: %s\n", action)
	fmt.Printf("Forwarding to RPC: %s\n", cl.GetEndpoint())
	fmt.Printf("Method: %s\n", data.GetParam("method", ""))
	fmt.Printf("=====================\n")

	// run the request
	is_req_ok := func(data []byte) bool {
		v := make(map[string]interface{})
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		dec.Decode(&v)

		switch v["result"].(type) {
		case nil:
			return false
		}
		return true
	}

	method := data.GetParam("method", "")
	params := data.GetParam("params", "")
	if len(method) == 0 {
		return `{"error":"provide transaction &method=eth_blockNumber or &method=eth_getBalance and optionally &params=[\"0x...\"] add &public=1 if you want to force the request to be run on public node"}`
	}

	// Try first client (private by default)
	ret, result := cl.RequestBasic(method, params)
	if ret != nil && result == client.R_OK && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// Try public client, if private failed
	cl = sch.GetPublicClient()
	if cl != nil {
		fmt.Printf("=== EVM RPC Retry ===\n")
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("Forwarding to public RPC: %s\n", cl.GetEndpoint())
		fmt.Printf("=====================\n")
		ret, result = cl.RequestBasic(method, params)
	}
	if ret != nil && result == client.R_OK && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// last result, try anything which is not throttled!
	for result == client.R_THROTTLED {
		cl = sch.GetAnyClient()
		if cl == nil {
			break
		}
		fmt.Printf("=== EVM RPC Retry (throttled) ===\n")
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("Forwarding to RPC: %s\n", cl.GetEndpoint())
		fmt.Printf("=====================\n")
		ret, result = cl.RequestBasic(method, params)
	}
	if ret != nil && result == client.R_OK && is_req_ok(ret) {
		data.FastReturnBNocopy(ret)
		return ""
	}

	// return error, if we were not able to process the request correctly
	if ret != nil {
		data.FastReturnBNocopy(ret)
		return ""
	}
	return `{"error":"unknown issue"}`
}
