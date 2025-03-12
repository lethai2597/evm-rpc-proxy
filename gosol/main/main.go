package main

import (
	"fmt"
	"gosol/evm/handle_ethereum_raw"
	"gosol/evm/handle_evm_admin"
	"gosol/handle_kvstore"
	handle_passthrough "gosol/passthrough"
	plugin_manager "gosol/plugins"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/handle_echo"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/handle_profiler"

	// "gosol/solana/handle_solana_01"
	// "gosol/solana/handle_solana_info"
	// "gosol/solana/handle_solana_admin"
	"os"
	"runtime"
	"strings"
)

// func _read_node_config() {

// 	fmt.Println("\nCấu hình node RPC cố định...")

// 	// RPC 1: RPC công khai (public)
// 	// rpc1 := map[string]interface{}{
// 	// 	"url":    "https://eth.llamarpc.com",
// 	// 	"public": true,
// 	// }
// 	// handle_evm_admin.NodeRegisterFromConfig(rpc1)
// 	// fmt.Println("Đã đăng ký RPC công khai: https://eth.llamarpc.com")

// 	// RPC 2: RPC riêng (private)
// 	rpc2 := map[string]interface{}{
// 		"url":    "https://node-eth.pinksale.com", // Ethereum Mainnet RPC
// 		"public": false,
// 	}
// 	handle_evm_admin.NodeRegisterFromConfig(rpc2)
// 	fmt.Println("Đã đăng ký RPC riêng: https://node-eth.pinksale.com")

// 	fmt.Println("Đã cấu hình xong node Ethereum RPC.")
// 	fmt.Println("")
// }

func _read_node_config() {

	fmt.Println("\nReading node config...")
	nodes := (config.Config().GetRawData("EVM_NODES", "")).([]interface{})
	if len(nodes) <= 0 {
		fmt.Println("ERROR: No nodes defined, please define at least one solana node to connect to")
		os.Exit(10)
		return
	}

	for _, v := range nodes {
		handle_evm_admin.NodeRegisterFromConfig(v.(map[string]interface{}))
	}
	fmt.Println("")
}

func main() {

	plugin_manager.RegisterAll()
	_read_node_config()

	num_cpu := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(num_cpu) // register handlers
	handlers := []handler_socket2.ActionHandler{}
	handlers = append(handlers, &handle_echo.HandleEcho{})
	handlers = append(handlers, &handle_profiler.HandleProfiler{})
	handlers = append(handlers, &handle_ethereum_raw.Handle_ethereum_raw{})
	// handlers = append(handlers, &handle_solana_01.Handle_solana_01{})
	// handlers = append(handlers, &handle_solana_info.Handle_solana_info{})
	handlers = append(handlers, &handle_passthrough.Handle_passthrough{})
	handlers = append(handlers, &handle_evm_admin.Handle_evm_admin{})
	handlers = append(handlers, &handle_kvstore.Handle_kvstore{})

	if len(config.Config().Get("RUN_SERVICES", "")) > 0 && config.Config().Get("RUN_SERVICES", "") != "*" {
		_h_modified := []handler_socket2.ActionHandler{}
		_tmp := strings.Split(config.Config().Get("RUN_SERVICES", ""), ",")
		supported := make(map[string]bool)
		for _, v := range _tmp {
			supported[strings.Trim(v, "\r\n \t")] = true
		}

		for _, v := range handlers {
			should_enable := false
			for _, action := range v.GetActions() {
				if supported[action] {
					should_enable = true
					break
				}
			}

			if should_enable {
				_h_modified = append(_h_modified, v)
			}
		}

		handlers = _h_modified
	}

	// start the server
	handler_socket2.RegisterHandler(handlers...)
	handler_socket2.StartServer(strings.Split(config.Config().Get("BIND_TO", ""), ","))
}
