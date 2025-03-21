package evm_proxy

import (
	"goevm/evm_proxy/client"
)

func (this *scheduler) _pick_next() *client.EVMClient {
	min, min_pos := -1, -1
	for num, v := range this.clients {
		if v == nil {
			continue
		}

		info := v.GetInfo()
		if info.Is_disabled || info.Is_throttled || info.Is_paused {
			this.clients[num] = nil
			continue
		}

		if info.Is_public_node && this.force_private {
			continue
		}
		if !info.Is_public_node && this.force_public {
			continue
		}

		_r := info.Score
		if min == -1 || _r < min {
			min = _r
			min_pos = num
		}
	}

	if min_pos == -1 {
		return nil
	}
	ret := this.clients[min_pos]
	this.clients[min_pos] = nil
	return ret
}
