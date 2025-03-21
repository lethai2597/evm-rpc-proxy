package throttle

import (
	"time"
)

type LimiterType uint8

const (
	L_REQUESTS        LimiterType = 0
	L_REQUESTS_PER_FN             = 1
	L_DATA_RECEIVED               = 2
)

type Limiter struct {
	t               LimiterType
	maximum         int
	in_time_windows int
}

type Throttle struct {
	limiters []Limiter

	stats_pos                 int
	stats                     []stat
	stats_window_size_seconds int

	score_modifier int

	status_score         int
	status_throttled     bool
	status_capacity_used int
}

func Make() *Throttle {
	return MakeCustom(120, 1)
}

func MakeCustom(window_count, window_size_seconds int) *Throttle {
	ret := &Throttle{}
	ret.limiters = make([]Limiter, 0, 10)

	ret.stats = make([]stat, window_count)
	ret.stats_window_size_seconds = window_size_seconds
	ret.stats_pos = (int(time.Now().Unix()) / ret.stats_window_size_seconds) % len(ret.stats)

	for i := 0; i < len(ret.stats); i++ {
		ret.stats[i].stat_request_by_fn = make(map[string]int)
	}
	return ret
}

func (this *Throttle) AddLimiter(t LimiterType, maximum, time_seconds int) {
	this.limiters = append(this.limiters, Limiter{t, maximum, time_seconds / this.stats_window_size_seconds})
}

func (this *Throttle) SetScoreModifier(m int) {
	this.score_modifier = m
}

func (this *Throttle) IsThrottled(fn string) bool {
	this.OnMaintenance(int(time.Now().Unix()))
	return this.status_throttled
}

func (this *Throttle) GetThrottleScore() ThrottleScore {
	this.OnMaintenance(int(time.Now().Unix()))
	return ThrottleScore{this.status_score, this.status_throttled, this.status_capacity_used}
}

func (this *Throttle) OnMaintenance(now int) {
	_pos := (now / this.stats_window_size_seconds) % len(this.stats)
	if _pos == this.stats_pos {
		return
	}

	// Clear stats for the new position
	this.stats[_pos].stat_done = 0
	this.stats[_pos].stat_bytes_received = 0
	this.stats[_pos].stat_request_by_fn = make(map[string]int)
	this.stats_pos = _pos

	// Check if we're throttled
	this.status_throttled = false
	this.status_score = 0
	this.status_capacity_used = 0

	for _, limiter := range this.limiters {
		_max := limiter.maximum
		_in_windows := limiter.in_time_windows
		_used := 0

		_pos := this.stats_pos
		for i := 0; i < _in_windows; i++ {
			_pos--
			if _pos < 0 {
				_pos = len(this.stats) - 1
			}

			switch limiter.t {
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
		if _score > this.status_capacity_used {
			this.status_capacity_used = _score
		}
		if _score > this.status_score {
			this.status_score = _score
		}

		if _used >= _max {
			this.status_throttled = true
		}
	}

	this.status_score += this.score_modifier
}
