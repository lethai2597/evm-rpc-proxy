package client

import (
	"fmt"
	node_status "goevm/evm_proxy/client/status"
	"goevm/evm_proxy/client/throttle"
	"html"
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

	out.AddBadge(fmt.Sprintf("%d Requests Running", this.stat_running), node_status.Gray, "Number of requests currently being processed.")
	if this._probe_time >= 10 {
		out.AddBadge("Conserve Requests", node_status.Green, "Health checks are limited for\nthis node to conserve requests.\n\nIf you're paying per-request\nit's good to enable this mode.")
	}

	// Add throttling badges
	throttle.ThrottleGoup(this.throttle).GetStatusBadges(out, node_status.Purple)

	// show last error if we have any
	if this._last_error.counter > 0 {
		last_error_header, last_error_details := this._last_error.Info()
		_comment := html.EscapeString(last_error_header) + "\n" + html.EscapeString(last_error_details)
		out.AddBadge(fmt.Sprintf("Has Errors: %d", this._last_error.counter), node_status.Orange, _comment)
	}

	// Next health badge
	{
		_dead, r, e, _comment := this._statsIsDead()
		_comment = "Node status which will be applied during the next update:\n" + _comment
		if _dead {
			out.AddBadge(fmt.Sprintf("Predicted Not Healthy (%dR/%dE)", r, e), node_status.Red, _comment)
		} else {
			out.AddBadge(fmt.Sprintf("Predicted Healthy (%dR/%dE)", r, e), node_status.Green, _comment)
		}
	}

	// Paused status
	{
		if this.is_paused {
			_p := "Node is paused"
			if len(this.is_paused_comment) > 0 {
				_p += ", reason:\n" + this.is_paused_comment
			} else {
				_p += ", no additional info present"
			}
			out.AddBadge("Paused", node_status.Gray, _p)
		}
	}

	// Generate content (throttle settings)
	{
		content := ""
		for _, throttle := range this.throttle {
			content += throttle.GetStatus()
		}
		out.AddContent(content)
	}

	// Requests statistics
	{
		_get_row := func(label string, s stat, time_running int) []string {
			_req := fmt.Sprintf("%d", s.stat_done)
			_req_s := fmt.Sprintf("%.02f", float64(s.stat_done)/float64(time_running))
			_req_avg := fmt.Sprintf("%.02f ms", (float64(s.stat_ns_total)/float64(s.stat_done))/1000.0)

			_r := make([]string, 0, 10)
			_r = append(_r, label, _req, _req_s, _req_avg)

			_r = append(_r, fmt.Sprintf("%d", s.stat_error_json_marshal))
			_r = append(_r, fmt.Sprintf("%d", s.stat_error_req))
			_r = append(_r, fmt.Sprintf("%d", s.stat_error_resp))
			_r = append(_r, fmt.Sprintf("%d", s.stat_error_resp_read))
			_r = append(_r, fmt.Sprintf("%d", s.stat_error_json_decode))

			_r = append(_r, fmt.Sprintf("%.02fMB", float64(s.stat_bytes_sent)/1000/1000))
			_r = append(_r, fmt.Sprintf("%.02fMB", float64(s.stat_bytes_received)/1000/1000))
			return _r
		}

		// Get current stats
		this.mu.Lock()
		r := this.stat_total
		this.mu.Unlock()

		// Statistics
		table := hscommon.NewTableGen("Time", "Requests", "Req/s", "Avg Time",
			"Err JM", "Err Req", "Err Resp", "Err RResp", "Err Decode", "Sent", "Received")
		table.SetClass("tab evm")

		time_running := time.Now().Unix() - start_time
		table.AddRow(_get_row("Total", r, int(time_running))...)
		out.AddContent(table.Render())
	}

	return "\n" + out.GetHTML()
}
