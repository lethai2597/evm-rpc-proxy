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
			time.Sleep(1 * time.Second) // Increased sleep time since EVM doesn't need frequent updates
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			now = _t
			_maint_stat(now)
		}
	}()
}
