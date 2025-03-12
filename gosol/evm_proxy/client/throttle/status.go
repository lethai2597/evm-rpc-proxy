package throttle

import (
	"fmt"
	"gosol/evm_proxy/client/status"
	"strings"

	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
)

/* This has to hold mutex externally */
func (this *Throttle) GetStatus() string {

	_progress := func(p int) string {
		if p > 100 {
			p = 100
		}
		p = p / 10

		ret := strings.Repeat("◆", p)
		if p < 10 && 10-p > 0 {
			ret = ret + strings.Repeat("◇", 10-p)
		}
		return ret
	}

	status := "<span style='color: #449944; font-family: monospace'> <b>⬤</b> Throttling disabled (##layout##) ⏵︎⏵︎⏵︎</span>"
	if (len(this.limiters) > 0) && this.status_throttled {
		status = "<span style='color: #dd4444; font-family: monospace'> <b>⮿</b> Throttling group (##layout##), exhausted</span>"
	}
	if (len(this.limiters) > 0) && !this.status_throttled {
		status = "<span style='color: #449944; font-family: monospace'><b>⬤</b> Throttling group (##layout##)</span>"
	}

	layout := ""
	for _, limiter := range this.limiters {
		_used, _score := this._getThrottleStatus(&limiter)
		_max := limiter.maximum
		_in_windows := limiter.in_time_windows
		_left := _max - _used

		_type := "?"
		switch limiter.t {
		case L_REQUESTS:
			_type = "Req"
		case L_REQUESTS_PER_FN:
			_type = "Req/Fn"
		case L_DATA_RECEIVED:
			_type = "Data"
		}

		_time := _in_windows * this.stats_window_size_seconds
		_time_str := fmt.Sprintf("%ds", _time)
		if _time >= 60 {
			_time_str = fmt.Sprintf("%dm", _time/60)
		}
		if _time >= 3600 {
			_time_str = fmt.Sprintf("%dh", _time/3600)
		}

		_max_str := fmt.Sprintf("%d", _max)
		if limiter.t == L_DATA_RECEIVED {
			_max_str = hscommon.FormatBytes(uint64(_max))
		}

		_left_str := fmt.Sprintf("%d", _left)
		if limiter.t == L_DATA_RECEIVED {
			_left_str = hscommon.FormatBytes(uint64(_left))
		}

		layout += fmt.Sprintf("%s/%s %s %s %s %s<br>", _type, _time_str, _max_str, _progress(_score), _left_str, fmt.Sprintf("%.1f%%", float64(_score)))
	}

	status = strings.Replace(status, "##layout##", layout, 1)
	return status
}

func (this ThrottleGoup) GetStatusBadges(out *status.Status, color status.Color) {
	for _, throttle := range this {
		if len(throttle.limiters) == 0 {
			continue
		}

		for _, limiter := range throttle.limiters {
			_used, _score := throttle._getThrottleStatus(&limiter)
			_max := limiter.maximum
			_in_windows := limiter.in_time_windows
			_left := _max - _used

			_type := "?"
			switch limiter.t {
			case L_REQUESTS:
				_type = "Requests"
			case L_REQUESTS_PER_FN:
				_type = "Requests/Fn"
			case L_DATA_RECEIVED:
				_type = "Data"
			}

			_time := _in_windows * throttle.stats_window_size_seconds
			_time_str := fmt.Sprintf("%ds", _time)
			if _time >= 60 {
				_time_str = fmt.Sprintf("%dm", _time/60)
			}
			if _time >= 3600 {
				_time_str = fmt.Sprintf("%dh", _time/3600)
			}

			_max_str := fmt.Sprintf("%d", _max)
			if limiter.t == L_DATA_RECEIVED {
				_max_str = hscommon.FormatBytes(uint64(_max))
			}

			_left_str := fmt.Sprintf("%d", _left)
			if limiter.t == L_DATA_RECEIVED {
				_left_str = hscommon.FormatBytes(uint64(_left))
			}

			_title := fmt.Sprintf("Throttle %s/%s", _type, _time_str)
			_desc := fmt.Sprintf("Maximum: %s\nLeft: %s\nUsage: %.1f%%", _max_str, _left_str, float64(_score))
			out.AddBadge(_title, color, _desc)
		}
	}
}
