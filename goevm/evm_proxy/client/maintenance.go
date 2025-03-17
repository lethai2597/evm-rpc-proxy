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
		_, _, _, _ok := this.GetVersion()
		if _ok != R_OK {
			fmt.Println("Health: Can't get version for: ", this.endpoint)
			return
		}
	}

	_update_last_block := func() {
		_, _ok := this.GetLastAvailableBlock()
		if _ok != R_OK {
			fmt.Println("Health: Can't get last block for: ", this.endpoint)
			return
		}
	}

	// run first update, get all data required for the node to work!
	_update_version()
	_update_last_block()

	go func() {
		for {
			now := time.Now().Unix()
			time.Sleep(500 * time.Millisecond)
			_t := time.Now().Unix()
			if now >= _t {
				continue
			}

			// update version and last block
			now = _t

			// if we have probing time set - use that
			if pt := int64(this._probe_time); pt > 0 {
				pt_by2 := pt * 2
				if now%pt_by2 == 0 {
					_update_version()
				}
				if now%pt_by2 == pt {
					_update_last_block()
				}
			}
		}
	}()

	_maint_stat(time.Now().Unix())
}
