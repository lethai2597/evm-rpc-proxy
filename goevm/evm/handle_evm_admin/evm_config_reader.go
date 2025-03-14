package handle_evm_admin

import (
	"encoding/json"
	"fmt"
	"goevm/evm_proxy"
	"goevm/evm_proxy/client"
	"goevm/evm_proxy/client/throttle"
	"math"
	"net/http"
	"reflect"
	"strings"
)

func NodeRegister(endpoint string, header http.Header, public bool, probe_time int, throttle []*throttle.Throttle) *client.EVMClient {
	if len(endpoint) == 0 {
		return nil
	}
	max_conn := 50
	if public {
		max_conn = 10
	}
	endpoint = strings.Trim(endpoint, "\r\n\t ")

	if probe_time == -1 {
		if public {
			probe_time = 10
		} else {
			probe_time = 1
		}
	}

	cl := client.MakeClient(endpoint, header, public, probe_time, max_conn, throttle)
	evm_proxy.ClientManage(cl, math.MaxUint64)
	return cl
}

func _get_cfg_data[T any](node map[string]interface{}, attr string, def T) T {
	if val, ok := node[attr]; ok {
		switch val.(type) {
		case T:
			return val.(T)
		default:
			fmt.Println("Warning: type mismatch for", attr, "attribute is", reflect.TypeOf(val).Name(), ", needs to be ", reflect.TypeOf(new(T)).Name())
		}
	}
	return def
}

func NodeRegisterFromConfig(node map[string]interface{}) *client.EVMClient {

	url := _get_cfg_data(node, "url", "")
	public := _get_cfg_data(node, "public", false)
	score_modifier, _ := _get_cfg_data(node, "score_modifier", json.Number("0")).Int64()
	probe_time, _ := _get_cfg_data(node, "probe_time", json.Number("-1")).Int64()
	header := parseHeader(_get_cfg_data(node, "header", ""))

	if url == "" {
		fmt.Println("Cannot read node config (no url) ... skipping")
		return nil
	}

	thr := ([]*throttle.Throttle)(nil)
	logs := []string{}
	fmt.Printf("## Node: %s Public: %v, score modifier: %d\n", url, public, score_modifier)

	if val, ok := node["throttle"]; ok {
		switch val.(type) {
		case string:
			thr, logs = throttle.MakeFromConfig(val.(string))
		default:
			fmt.Println("Warning: Cannot read throttle settings, skipping throttling")
		}
	} else {
		if public {
			thr, logs = throttle.MakeForPublic()
		}
	}

	if thr == nil {
		thr = make([]*throttle.Throttle, 0, 1)
		thr = append(thr, throttle.Make())
		logs = append(logs, "Throttling disabled")
	}
	throttle.ThrottleGoup(thr).SetScoreModifier(int(score_modifier))

	for _, log := range logs {
		fmt.Println(" ", log)
	}

	return NodeRegister(url, header, public, int(probe_time), thr)
}
