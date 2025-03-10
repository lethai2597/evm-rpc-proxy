package throttle

type stat struct {
	stat_done           int
	stat_request_by_fn  map[string]int
	stat_bytes_received int
}

type ThrottleScore struct {
	Score        int
	Throttled    bool
	CapacityUsed int
}

func (this *Throttle) OnRequest(function_name string) bool {
	if this.status_throttled {
		return false
	}

	to_mod := &this.stats[this.stats_pos]
	to_mod.stat_done++
	to_mod.stat_request_by_fn[function_name]++

	// update statistics
	tmp := this._getThrottleScore()
	this.status_throttled = tmp.Throttled
	this.status_score = tmp.Score
	this.status_capacity_used = tmp.CapacityUsed
	return true
}

func (this *Throttle) OnReceive(data_bytes int) {
	to_mod := &this.stats[this.stats_pos]
	to_mod.stat_bytes_received += data_bytes
}

func (this *Throttle) _getThrottleStatus(l *Limiter) (int, int) {
	_max := l.maximum
	_in_windows := l.in_time_windows
	_used := 0

	_pos := this.stats_pos
	for i := 0; i < _in_windows; i++ {
		_pos--
		if _pos < 0 {
			_pos = len(this.stats) - 1
		}

		switch l.t {
		case L_REQUESTS:
			_used += this.stats[_pos].stat_done
		case L_DATA_RECEIVED:
			_used += this.stats[_pos].stat_bytes_received
		case L_REQUESTS_PER_FN:
			for _, v := range this.stats[_pos].stat_request_by_fn {
				_used += v
			}
		}
	}

	_score := 0
	if _max > 0 {
		_score = (_used * 100) / _max
	}

	return _used, _score
}

func (this *Throttle) _getThrottleScore() ThrottleScore {
	ret := ThrottleScore{}
	ret.Score = 0
	ret.Throttled = false
	ret.CapacityUsed = 0

	for _, limiter := range this.limiters {
		_used, _score := this._getThrottleStatus(&limiter)
		_max := limiter.maximum

		ret.Score += _score + this.score_modifier
		if _score > ret.CapacityUsed {
			ret.CapacityUsed = _score
		}

		if _used >= _max {
			ret.Throttled = true
		}
	}

	return ret
}
