package handler_socket2

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
)

type HTTPPlugin func(http.ResponseWriter, http.Header, map[string]string, []byte) bool

var HTTPPlugins = make([]HTTPPlugin, 0)

func HTTPPluginRegister(f HTTPPlugin) {
	HTTPPlugins = append(HTTPPlugins, f)
}

func startServiceHTTP(bindTo string, handler handlerFunc) {

	fmt.Printf("HTTP Service starting : %s\n", bindTo)
	boundMutex.Lock()
	boundTo = append(boundTo, "http://"+bindTo)
	boundMutex.Unlock()

	handle_hunc := func(w http.ResponseWriter, r *http.Request) {
		req_len := 0
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Access-Control-Allow-Methods", "*")
		// w.Header().Add("Access-Control-Allow-Methods", "*")
		// if r.Method == "OPTIONS" {
		// 	// w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// 	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		// 	w.Header().Add("Content-Length", "0")
		// 	w.Header().Add("Access-Control-Allow-Credentials", "true")
		// 	w.Header().Add("Access-Control-Max-Age", "1728000")
		// }
		params := make(map[string]string)
		for k, v := range r.URL.Query() {
			v_ := strings.Join(v, ",")
			params[k] = v_
			req_len += len(k) + len(v)
		}

		for k, v := range r.Header {
			req_len += len(k) + len(v)
		}

		// build request representation from GET and POST
		str_req_id := r.URL.RawQuery
		r_body := make([]byte, 0)
		if r.Method == "POST" {
			tmp, err := ioutil.ReadAll(r.Body)
			if err == nil {
				r.Body.Close()
				r_body = tmp

				if len(r_body) > 40 {
					str_req_id += " P:" + string(r_body[0:40]) + "..."
				} else {
					str_req_id += " P:" + string(r_body)
				}
			}
		}

		_req_status := &httpRequest{str_req_id, time.Now().UnixNano(), 0, "R"}
		httpStatMutex.Lock()
		httpRequestId++
		_my_reqid := httpRequestId
		httpRequestStatus[_my_reqid] = _req_status
		httpStatMutex.Unlock()

		// plugins support, now we can process raw HTTP request
		for _, plugin := range HTTPPlugins {
			if plugin(w, r.Header, params, r_body) {

				_end := time.Now().UnixNano()
				go func(_my_reqid uint64, _end int64) {
					httpStatMutex.Lock()
					_req_status.status = "F"
					_req_status.end_time = _end
					httpStatMutex.Unlock()

					time.Sleep(5000 * time.Millisecond)
					httpStatMutex.Lock()
					delete(httpRequestStatus, _my_reqid)
					httpStatMutex.Unlock()
				}(_my_reqid, _end)

				return
			}
		}

		hsparams := CreateHSParamsFromMap(params)

		ret := []byte(handleRequest(hsparams))
		if hsparams.fastreturn != nil {
			ret = hsparams.fastreturn
		}

		// w.Header().Add("Server", version)
		w.Header().Add("Content-type", "text/html")
		w.Header().Add("Content-length", strconv.Itoa(len(ret)))

		// w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
		// w.Header().Add("Pragma", "no-cache")
		// w.Header().Add("Expires", "0")

		for _, v := range hsparams.additional_resp_headers {
			_pos := strings.IndexByte(v, ':')
			if _pos < 0 {
				continue
			}
			w.Header().Add(v[0:_pos], v[_pos+1:])
		}

		httpStatMutex.Lock()
		_req_status.status = "W"
		httpStatMutex.Unlock()

		w.Write(ret)

		httpStatMutex.Lock()
		_req_status.status = "F"
		_req_status.end_time = time.Now().UnixNano()
		httpStatMutex.Unlock()

		hsparams.Cleanup()

		go func(_my_reqid uint64) {
			time.Sleep(5000 * time.Millisecond)
			httpStatMutex.Lock()
			delete(httpRequestStatus, _my_reqid)
			httpStatMutex.Unlock()
		}(_my_reqid)

	}

	err := http.ListenAndServe(bindTo, http.HandlerFunc(handle_hunc))
	if err != nil {
		fmt.Println("HTTP Error listening on TCP address: ", bindTo)
	}

}

func GetStatusHTTP() string {
	httpStatMutex.Lock()
	now := time.Now().UnixNano()
	scored_items := make([]hscommon.ScoredItems, len(httpRequestStatus))
	i := 0
	for k, rs := range httpRequestStatus {

		took_str := "??"
		took := float64(0)
		if rs.start_time != 0 {
			if rs.end_time == 0 {
				took = float64(now-rs.start_time) / float64(1000000)
			} else {
				took = float64(rs.end_time-rs.start_time) / float64(1000000)
			}
			took_str = fmt.Sprintf("%.3fms", took)
		}

		tmp := fmt.Sprintf("<div class='thread_list'><span>Num. %d</span> - <span>[%s]</span> <span>%s</span> <span>%s</span></div>", k, rs.status, took_str, rs.req)

		scored_items[i] = hscommon.ScoredItems{Item: tmp, Score: int64(k)}
		i++
	}

	httpStatMutex.Unlock()

	sort.Sort(hscommon.SIArr(scored_items))
	tmp := ""
	for _, v := range scored_items {
		tmp += v.Item
	}

	return tmp
}
