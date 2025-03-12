package throttle

import (
	"fmt"
	"strconv"
	"strings"
)

func MakeFromConfig(config string) ([]*Throttle, []string) {
	logs := make([]string, 0)
	error := func(a ...interface{}) ([]*Throttle, []string) {
		logs = append(logs, fmt.Sprint(a...))
		return nil, logs
	}
	log := func(a ...interface{}) {
		logs = append(logs, fmt.Sprint(a...))
	}

	type tw struct {
		time_window_size  int
		time_window_count int
	}
	_get_timewindow_len := func(stat_time int) tw {
		if stat_time <= 120 {
			return tw{1, 120}
		}
		if stat_time <= 1200 {
			if stat_time%10 == 0 {
				return tw{10, 120}
			}
		}
		if stat_time <= 3600 {
			if stat_time%30 == 0 {
				return tw{30, 120}
			}
		}
		if stat_time <= 86400 {
			if stat_time%60 == 0 {
				return tw{60, 1440}
			}
		}
		return tw{1, 120}
	}

	ret := make([]*Throttle, 0)
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) < 3 {
			return error("Invalid throttle config, expected at least 3 parts: ", line)
		}

		_type := strings.TrimSpace(parts[0])
		_max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return error("Invalid throttle config, max is not a number: ", line)
		}
		_time, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return error("Invalid throttle config, time is not a number: ", line)
		}

		_score_modifier := 0
		if len(parts) > 3 {
			_score_modifier, err = strconv.Atoi(strings.TrimSpace(parts[3]))
			if err != nil {
				return error("Invalid throttle config, score modifier is not a number: ", line)
			}
		}

		_tw := _get_timewindow_len(_time)
		t := MakeCustom(_tw.time_window_count, _tw.time_window_size)
		t.SetScoreModifier(_score_modifier)

		switch _type {
		case "requests":
			t.AddLimiter(L_REQUESTS, _max, _time)
			log("Added requests throttle: ", _max, " in ", _time, "s")
		case "requests_per_fn":
			t.AddLimiter(L_REQUESTS_PER_FN, _max, _time)
			log("Added requests_per_fn throttle: ", _max, " in ", _time, "s")
		case "data_received":
			t.AddLimiter(L_DATA_RECEIVED, _max, _time)
			log("Added data_received throttle: ", _max, " in ", _time, "s")
		default:
			return error("Invalid throttle config, unknown type: ", _type)
		}

		ret = append(ret, t)
	}

	return ret, logs
}

func MakeForPublic() ([]*Throttle, []string) {
	return MakeFromConfig(`
# Type; Max; Time in seconds; Score modifier
requests; 100; 60; 0
requests_per_fn; 200; 60; 0
data_received; 1048576; 60; 0
`)
}
