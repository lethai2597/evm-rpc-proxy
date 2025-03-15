package evm_proxy

import (
	"fmt"
	"goevm/evm_proxy/client"
	"strings"
	"sync"
	"time"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
)

type custom_health_checker struct {
	mu sync.Mutex

	run_every       int64
	max_data_age_ms int64
	max_block_lag   int

	_log string
}

var cc custom_health_checker

func init() {

	cfg := config.Config()
	has_custom_checker, err := cfg.ValidateAttribs("CUSTOM_HEALTH_CHECKER", []string{"run_every", "max_block_lag", "max_data_age_ms"})
	if err != nil {
		panic("Custom chealth checker config error. " + err.Error())
	}
	if !has_custom_checker {
		return
	}

	run_every, err := config.Config().GetSubattrInt("CUSTOM_HEALTH_CHECKER", "run_every")
	if err != nil {
		panic(err)
	}
	max_block_lag, err := config.Config().GetSubattrInt("CUSTOM_HEALTH_CHECKER", "max_block_lag")
	if err != nil {
		panic(err)
	}
	max_data_age_ms, err := config.Config().GetSubattrInt("CUSTOM_HEALTH_CHECKER", "max_data_age_ms")
	if err != nil {
		panic(err)
	}

	cc = custom_health_checker{}
	cc.run_every = int64(run_every)
	cc.max_block_lag = max_block_lag
	cc.max_data_age_ms = int64(max_data_age_ms)

	go func() {
		last := time.Now().Unix()
		for {
			time.Sleep(750 * time.Millisecond)
			if t := time.Now().Unix(); t-last < cc.run_every {
				continue
			} else {
				last = t
			}
			_run_custom_check()
		}
	}()

	handler_socket2.StatusPluginRegister(func() (string, string) {
		ret := "Custom health plugin will pause nodes when they start lagging\n"
		ret += fmt.Sprintf("run_every: %d - run the check every X seconds\n", cc.run_every)
		ret += fmt.Sprintf("max_block_lag: %d - maximum number of blocks which a node can lag behind, before being paused\n", cc.max_block_lag)
		ret += fmt.Sprintf("max_data_age_ms: %d - maximum age of highest block data (in milliseconds), if max block data is older, it'll be re-fetched\n", cc.max_data_age_ms)

		ret += "--------\n"
		ret += cc._log
		if len(cc._log) == 0 {
			ret += "Waiting for data"
		}

		return "EVM Proxy - Custom Health Plugin", "<pre>" + ret + "</pre>"
	})
}

func _run_custom_check() {
	infos := []*client.EVMClientinfo{}
	status := []string{}

	mu.RLock()
	for _, client := range clients {
		info := client.GetInfo()
		infos = append(infos, info)
		status = append(status, client.GetStatus())
	}
	mu.RUnlock()

	log := ""
	for num, info := range infos {
		is_ok := true
		_is_ok := "OK     "
		if !is_ok {
			_is_ok = "LAGGING"
		}

		for _, client := range clients {
			if strings.Compare(client.GetEndpoint(), info.Endpoint) == 0 {
				client.SetPaused(!is_ok, "")
			}
		}

		log += fmt.Sprintf("Node #%d %s Score: %d %s\n",
			num, _is_ok, info.Score, status[num])
	}

	cc.mu.Lock()
	cc._log = log
	cc.mu.Unlock()
}
