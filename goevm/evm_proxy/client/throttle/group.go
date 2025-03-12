package throttle

import (
	"math"
)

type ThrottleGoup []*Throttle

func (this ThrottleGoup) OnRequest(function_name string) bool {

	for _, throttle := range this {
		if throttle.status_throttled {
			return false
		}
	}

	for _, throttle := range this {
		if !throttle.OnRequest(function_name) {
			return false
		}
	}
	return true
}

func (this ThrottleGoup) OnReceive(data_bytes int) {
	for _, throttle := range this {
		throttle.OnReceive(data_bytes)
	}
}

func (this ThrottleGoup) OnMaintenance(now int) {
	for _, throttle := range this {
		throttle.OnMaintenance(now)
	}
}

func (this ThrottleGoup) GetThrottleScore() ThrottleScore {
	ret := ThrottleScore{}
	ret.Score = 0
	ret.Throttled = false
	ret.CapacityUsed = 0

	for _, throttle := range this {
		tmp := throttle.GetThrottleScore()
		ret.Score += tmp.Score
		ret.CapacityUsed = int(math.Max(float64(ret.CapacityUsed), float64(tmp.CapacityUsed)))
		if tmp.Throttled {
			ret.Throttled = true
		}
	}

	return ret
}

func (this ThrottleGoup) GetLimitsLeft() (int, int, int, int) {
	ret_req := math.MaxInt32
	ret_req_fn := math.MaxInt32
	ret_data := math.MaxInt32
	ret_score := 0

	for _, throttle := range this {
		for _, limiter := range throttle.limiters {
			_used, _score := throttle._getThrottleStatus(&limiter)
			_max := limiter.maximum
			_left := _max - _used

			switch limiter.t {
			case L_REQUESTS:
				ret_req = int(math.Min(float64(ret_req), float64(_left)))
			case L_REQUESTS_PER_FN:
				ret_req_fn = int(math.Min(float64(ret_req_fn), float64(_left)))
			case L_DATA_RECEIVED:
				ret_data = int(math.Min(float64(ret_data), float64(_left)))
			}

			ret_score += _score
		}
	}

	return ret_req, ret_req_fn, ret_data, ret_score
}

func (this ThrottleGoup) SetScoreModifier(m int) {
	for _, throttle := range this {
		throttle.SetScoreModifier(m)
	}
}

func (this ThrottleGoup) IsThrottled(fn string) bool {
	for _, throttle := range this {
		if throttle.IsThrottled(fn) {
			return true
		}
	}
	return false
}
