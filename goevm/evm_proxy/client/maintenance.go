package client

import (
	"fmt"
	"goevm/evm_proxy/client/throttle"
	"time"
)

func (this *EVMClient) _maintenance() {

	_maint_stat := func(now int64) {
		this.mu.Lock()

		throttle.ThrottleGoup(this.throttle).OnMaintenance(int(now))

		_p := int(now % 60)
		this.stat_last_60_pos = _p
		this.stat_last_60[_p].stat_done = 0
		this.stat_last_60[_p].stat_error_json_decode = 0
		this.stat_last_60[_p].stat_error_json_marshal = 0
		this.stat_last_60[_p].stat_error_req = 0
		this.stat_last_60[_p].stat_error_resp = 0
		this.stat_last_60[_p].stat_error_resp_read = 0
		this.stat_last_60[_p].stat_ns_total = 0

		this.stat_last_60[_p].stat_request_by_fn = make(map[string]int)
		this.stat_last_60[_p].stat_bytes_received = 0
		this.stat_last_60[_p].stat_bytes_sent = 0

		_d, _req_ok, _req_err, _log := this._statsIsDead()
		this.is_disabled = _d
		this._probe_log = _log

		// if we don't have at least 1 requests,
		// run a request to check if the node is alive
		// this._probe_time related
		if _req_ok+_req_err < 1 && this._probe_time > 0 {
			go func() {
				this.GetVersion()
			}()
		}
		this.mu.Unlock()
	}

	_update_version := func() {
		_a, _b, _c, ok := this.GetVersion()
		if ok != R_OK {
			fmt.Println("Health: Can't get version for: ", this.endpoint)
			return
		}
		this.mu.Lock()
		this.version_major, this.version_minor, this.version = _a, _b, _c
		this.mu.Unlock()
	}

	// run first update, get all data required for the node to work!
	_update_version()

	_maint_stat(time.Now().Unix())
	go func() {
		for {
			now := time.Now().Unix()
			time.Sleep(200 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			now = _t
			_maint_stat(now)
		}
	}()
}
