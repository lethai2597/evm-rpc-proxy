package client

import (
	"fmt"
	node_status "gosol/evm_proxy/client/status"
	"gosol/evm_proxy/client/throttle"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
)

var start_time = int64(0)

func init() {
	start_time = time.Now().Unix()
}

func (this *EVMClient) GetStatus() string {

	status_throttle := throttle.ThrottleGoup(this.throttle).GetThrottleScore()
	out, status_description := node_status.Create(this.is_paused, status_throttle.Throttled, this.is_disabled)

	// Node name and status description
	{
		header := ""
		_t := "Private"
		if this.is_public_node {
			_t = "Public"
		}
		_e := this.endpoint
		_util := fmt.Sprintf("%.02f%%", float64(status_throttle.CapacityUsed)/100.0)
		header += fmt.Sprintf("<b>%s Node #%d</b>, Score: %d, Utilization: %s, %s\n", _t, this.id, status_throttle.Score, _util, _e)
		header += status_description
		header += "\n"
		header += this._probe_log
		out.SetHeader(header)
	}

	// Add basic badges
	if len(this.version) > 0 {
		out.AddBadge("Version: "+this.version, node_status.Gray, "Version number was updated on: "+time.UnixMilli(this.version_ts).Format("2006-01-02 15:04:05"))
	}
	if this.header != nil && len(this.header) > 0 {
		h_ := ""
		for k, v := range this.header {
			vv := strings.Join(v, ", ")

			out := vv
			if strings.Index(strings.ToLower(k), "authorization") != -1 && len(vv) > 5 {
				out = ""
				if strings.Index(strings.ToLower(vv), "bearer") == 0 {
					out += vv[0:6] + " "
				}
				out += "****" + vv[len(vv)-5:]
			}
			h_ += k + ": " + out + "<br>"
		}
		out.AddBadge(fmt.Sprintf("%d Header(s) defined", len(this.header)), node_status.Gray, h_)
	}

	// Add throttling badges
	throttle.ThrottleGoup(this.throttle).GetStatusBadges(out, node_status.Purple)

	// Add throttling badges
	r := this._statsGetAggr(60)

	// throttling badges for stats
	if r.stat_done > 0 {
		out.AddBadge(fmt.Sprintf("Last 60s: %d reqs, %d err, Avg. %.3fms", r.stat_done, r.stat_error_resp+r.stat_error_resp_read, float64(r.stat_ns_total)/float64(r.stat_done)/1000000), node_status.Blue, "")
	}

	// throttling badges for status
	{
		_err_rate := 0.0
		if r.stat_done > 0 {
			_err_rate = 100 * float64(r.stat_error_resp+r.stat_error_resp_read) / float64(r.stat_done)
		}

		reqerrs := fmt.Sprintf("Error Rate: %.2f%% (%d/%d)", _err_rate, r.stat_error_resp+r.stat_error_resp_read, r.stat_done)
		if _err_rate >= 5 {
			out.AddBadge(reqerrs, node_status.Red, fmt.Sprintf("High Error Rate"))
		} else {
			out.AddBadge(reqerrs, node_status.Blue, "")
		}
	}

	// throttling badges for detailed errors
	if r.stat_error_req > 0 || r.stat_error_resp > 0 || r.stat_error_resp_read > 0 || r.stat_error_json_decode > 0 || r.stat_error_json_marshal > 0 {
		d := "Detailed Errors in last 60s\n"
		if r.stat_error_req > 0 {
			d += "\nRequest errors: " + strconv.Itoa(r.stat_error_req)
		}
		if r.stat_error_resp > 0 {
			d += "\nResponse errors: " + strconv.Itoa(r.stat_error_resp)
		}
		if r.stat_error_resp_read > 0 {
			d += "\nResponse Read errors: " + strconv.Itoa(r.stat_error_resp_read)
		}
		if r.stat_error_json_decode > 0 {
			d += "\nJSON Decode errors: " + strconv.Itoa(r.stat_error_json_decode)
		}
		if r.stat_error_json_marshal > 0 {
			d += "\nJSON Marshall errors: " + strconv.Itoa(r.stat_error_json_marshal)
		}

		out.AddBadge(fmt.Sprintf("Err JM: %d, Req: %d, Resp: %d, RResp: %d, Decode: %d",
			r.stat_error_json_marshal, r.stat_error_req, r.stat_error_resp, r.stat_error_resp_read, r.stat_error_json_decode),
			node_status.Orange, html.EscapeString(d))
	}

	if r.stat_bytes_received != 0 || r.stat_bytes_sent != 0 {
		out.AddBadge(hscommon.FormatBytes(uint64(r.stat_bytes_received))+" / "+hscommon.FormatBytes(uint64(r.stat_bytes_sent)), node_status.Blue,
			fmt.Sprintf("First number is received data, second is sent.\nTotal bytes received: %d\nTotal bytes sent: %d", r.stat_bytes_received, r.stat_bytes_sent))
	}

	// throttling badges for function calls
	if len(r.stat_request_by_fn) > 0 {
		bd := ""
		for k, v := range r.stat_request_by_fn {
			bd += k + ": " + fmt.Sprintf("%d", v) + " calls\n"
		}
		out.AddBadge(fmt.Sprintf("Function calls: %d", len(r.stat_request_by_fn)), node_status.Blue, bd)
	}

	// last error badge
	{
		var le LastError
		this.mu.Lock()
		le = this._last_error
		this.mu.Unlock()

		if le.counter > 0 {
			_h, _d := le.Info()
			_a := time.UnixMicro(le.call_ts).Format("15:04:05")
			out.AddBadge("Last Error at "+_a, node_status.Red, html.EscapeString(_h+"\n"+_d))
		}
	}

	return out.GetHTML()
}
