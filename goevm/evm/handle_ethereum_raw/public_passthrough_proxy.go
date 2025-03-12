package handle_ethereum_raw

import (
	"encoding/json"
	"fmt"
	"goevm/evm_proxy"
	"goevm/evm_proxy/client"
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

		is_sol_rpc := strings.EqualFold("application/json", header.Get("Content-Type"))
		if !is_sol_rpc {
			return false
		}

		for i := 0; i < len(post); i++ {
			if post[i] == '{' {
				is_sol_rpc = true
				break
			}
			if post[i] == '\n' || post[i] == '\r' || post[i] == ' ' {
				continue
			}
			break // we couldn't find JSON bracket, so it's not SOL RPC
		}
		if !is_sol_rpc {
			return false
		}

		// Identify method from JSON body
		var method string
		var jsonData interface{}
		if err := json.Unmarshal(post, &jsonData); err == nil {
			switch jsonData.(type) {
			case map[string]interface{}:
				if m, ok := jsonData.(map[string]interface{})["method"].(string); ok {
					method = m
				}
			case []interface{}:
				// Batch request - get the first method to display
				batch := jsonData.([]interface{})
				if len(batch) > 0 {
					if req, ok := batch[0].(map[string]interface{}); ok {
						if m, ok := req["method"].(string); ok {
							method = m + " (batch)"
						}
					}
				}
			}
		}

		sch := evm_proxy.MakeScheduler()
		clients := sch.GetAllSorted(false, false)
		if len(clients) == 0 {
			fmt.Println("Debug - No clients found")
			w.Write(_passthrough_err("Can't find any client"))
			return true
		}

		fmt.Printf("=== EVM RPC Direct Passthrough ===\n")
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("Available Clients: %d\n", len(clients))
		if len(clients) > 0 {
			fmt.Printf("First Client: %s\n", clients[0].GetEndpoint())
		}
		fmt.Printf("================================\n")

		// loop over workers, if we have "throttled" returned it'll try other workers
		errors := 0
		for i, cl := range clients {
			fmt.Printf("Trying client #%d: %s\n", i+1, cl.GetEndpoint())
			resp_type, resp_data := cl.RequestForward(post)
			if resp_type == client.R_OK {
				fmt.Printf("Success with client: %s\n", cl.GetEndpoint())
				w.Write(resp_data)
				return true
			}

			if resp_type == client.R_ERROR {
				fmt.Printf("Error with client: %s\n", cl.GetEndpoint())
				errors++
				if errors >= 2 {
					w.Write(_passthrough_err("Request failed (e)"))
					return true
				}
			}

			if resp_type == client.R_THROTTLED {
				fmt.Printf("Client throttled: %s\n", cl.GetEndpoint())
			}
		}

		w.Write(_passthrough_err("Request failed"))
		return true
	})
}
