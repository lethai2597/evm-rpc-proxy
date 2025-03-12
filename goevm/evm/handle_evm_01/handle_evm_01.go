package handle_evm_01

import (
	"goevm/evm_proxy"
	"goevm/evm_proxy/client"
	"strings"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
)

type Handle_evm_01 struct {
}

func (this *Handle_evm_01) Initialize() {
}

func (this *Handle_evm_01) Info() string {
	return "This plugin provides common Ethereum RPC methods"
}

func (this *Handle_evm_01) GetActions() []string {
	return []string{"getBlock", "getTransaction", "getBalance", "getTokenInfo"}
}

func (this *Handle_evm_01) HandleAction(action string, data *handler_socket2.HSParams) string {

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

	if action == "getBlock" {
		block_no := data.GetParam("block", "")
		if len(block_no) == 0 {
			return `{"error":"provide block number as &block=123 or block hash as &block=0x..."}`
		}

		full_tx := data.GetParamI("fullTx", 0) == 1
		ret, result := cl.GetBlock(block_no, full_tx)

		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}
			ret, result = cl.GetBlock(block_no, full_tx)
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	if action == "getTransaction" {
		hash := data.GetParam("hash", "")
		if len(hash) == 0 || !strings.HasPrefix(hash, "0x") {
			return `{"error":"provide transaction &hash=0x... (must start with 0x)"}`
		}

		ret, result := cl.GetTransaction(hash)
		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}

			ret, result = cl.GetTransaction(hash)
		}

		data.FastReturnBNocopy(ret)
		return ""
	}

	if action == "getBalance" {
		address := data.GetParam("address", "")
		if len(address) == 0 || !strings.HasPrefix(address, "0x") {
			return `{"error":"provide address &address=0x... (must start with 0x)"}`
		}

		blockParam := data.GetParam("block", "latest")

		ret, result := cl.GetBalance(address, blockParam)
		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}
			ret, result = cl.GetBalance(address, blockParam)
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	if action == "getTokenInfo" {
		token := data.GetParam("token", "")
		if len(token) == 0 || !strings.HasPrefix(token, "0x") {
			return `{"error":"provide token contract address &token=0x... (must start with 0x)"}`
		}

		ret, result := cl.GetTokenInfo(token)
		for result != client.R_OK {
			cl = sch.GetPublicClient()
			if cl == nil {
				cl = sch.GetAnyClient()
			}
			if cl == nil {
				return `{"error":"can't find appropriate client (2)"}`
			}
			ret, result = cl.GetTokenInfo(token)
		}
		data.FastReturnBNocopy(ret)
		return ""
	}

	return "No function?!"
}
