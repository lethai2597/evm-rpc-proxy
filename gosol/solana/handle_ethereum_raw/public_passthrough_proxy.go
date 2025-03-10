package handle_ethereum_raw

import (
	"encoding/json"
	"fmt"
	"gosol/solana_proxy"
	"gosol/solana_proxy/client"
	"net/http"
	"strings"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
)

func _passthrough_err(err string) []byte {
	out := make(map[string]interface{}, 0)
	out["message"] = err
	out["code"] = 111
	out["proxy_error"] = true
	b, e := json.Marshal(out)
	if e != nil {
		b = []byte("\"Unknown error\"")
	}
	return []byte("{\"error\":" + string(b) + "}")
}

func init() {

	handler_socket2.HTTPPluginRegister(func(w http.ResponseWriter, header http.Header, get map[string]string, post []byte) bool {

		// Kiểm tra xem có phải là yêu cầu Ethereum JSON-RPC không
		is_eth_rpc := strings.EqualFold("application/json", header.Get("Content-Type"))
		if !is_eth_rpc {
			return false
		}

		// Kiểm tra xem có phải là JSON không
		for i := 0; i < len(post); i++ {
			if post[i] == '{' {
				is_eth_rpc = true
				break
			}
			if post[i] == '\n' || post[i] == '\r' || post[i] == ' ' {
				continue
			}
			break // Không tìm thấy dấu ngoặc JSON, không phải Ethereum RPC
		}
		if !is_eth_rpc {
			return false
		}

		// In ra thông tin debug để kiểm tra
		fmt.Printf("Debug - Headers: %v\n", header)
		fmt.Printf("Debug - GET params: %v\n", get)
		fmt.Printf("Debug - POST body: %s\n", string(post))

		// Kiểm tra xem URL có phải là /action/ethereumRaw không
		isEthereumRawEndpoint := false

		// Kiểm tra từ tham số _path
		if path, ok := get["_path"]; ok {
			if path == "/action/ethereumRaw" {
				isEthereumRawEndpoint = true
			}
		}

		// Kiểm tra từ tham số action
		if !isEthereumRawEndpoint {
			if action, ok := get["action"]; ok && action == "ethereumRaw" {
				isEthereumRawEndpoint = true
			}
		}

		if !isEthereumRawEndpoint {
			return false
		}

		sch := solana_proxy.MakeScheduler()
		cl := sch.GetAnyClient()
		if cl == nil {
			w.Write(_passthrough_err("Can't find appropriate client"))
			return true
		}

		// Gửi yêu cầu đến Ethereum RPC
		resp_type, resp_data := cl.RequestForward(post)
		if resp_type == client.R_OK {
			w.Write(resp_data)
			return true
		}

		// Nếu client đầu tiên thất bại, thử client công khai
		cl = sch.GetPublicClient()
		if cl != nil {
			resp_type, resp_data = cl.RequestForward(post)
			if resp_type == client.R_OK {
				w.Write(resp_data)
				return true
			}
		}

		// Thử bất kỳ client nào không bị throttled
		for resp_type == client.R_THROTTLED {
			cl = sch.GetAnyClient()
			if cl == nil {
				break
			}
			resp_type, resp_data = cl.RequestForward(post)
			if resp_type == client.R_OK {
				w.Write(resp_data)
				return true
			}
		}

		w.Write(_passthrough_err("Request failed"))
		return true
	})
}
