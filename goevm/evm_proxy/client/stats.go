package client

import (
	"fmt"
)

type stat struct {
	stat_error_req          int
	stat_error_resp         int
	stat_error_resp_read    int
	stat_error_json_decode  int
	stat_error_json_marshal int
	stat_done               int
	stat_ns_total           uint64

	stat_request_by_fn  map[string]int
	stat_bytes_received int
	stat_bytes_sent     int
}

func (this *EVMClient) _statsIsDead() (bool, int, int, string) {
	probe_time := this._probe_time
	if probe_time < 30 {
		probe_time = 30
	}

	stat_requests := this.stat_total.stat_done
	stat_errors := this.stat_total.stat_error_resp +
		this.stat_total.stat_error_resp_read +
		this.stat_total.stat_error_json_decode

	// Node is considered dead if:
	// 1. No probing and errors are more than 20% of requests
	// 2. With probing and errors are more than or equal to 20% of requests
	dead := this._probe_time == 0 && stat_errors*5 > stat_requests
	dead = dead || this._probe_time > 0 && stat_errors*5 >= stat_requests

	log := fmt.Sprintf("Health probing time %ds, Total Requests: %d, Total Errors: %d",
		probe_time, stat_requests, stat_errors)
	return dead, stat_requests, stat_errors, log
}
