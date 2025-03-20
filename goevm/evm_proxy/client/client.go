package client

import (
	"goevm/evm_proxy/client/throttle"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type EVMClientAttr int

const (
	CLIENT_CONSERVE_REQUESTS EVMClientAttr = 1 << 0
)

func (this *EVMClient) SetAttr(attrs EVMClientAttr) {
	this.attr = attrs
}

func (this *EVMClient) SetPaused(paused bool, comment string) {
	this.mu.Lock()
	this.is_paused = paused
	if this.is_paused {
		this.is_paused_comment = comment
	}
	this.mu.Unlock()
}

type EVMClient struct {
	id                      uint64
	client                  *http.Client
	endpoint                string
	header                  http.Header
	is_public_node          bool
	available_block_last    int
	available_block_last_ts int64

	is_disabled       bool
	is_paused         bool
	is_paused_comment string
	is_throttled      bool
	pause_comment     string
	throttle_comment  string
	disabled_comment  string

	stat_running int
	stat_total   stat

	mu        sync.Mutex
	serial_no uint64

	attr     EVMClientAttr
	throttle []*throttle.Throttle

	_probe_time       int
	_probe_time_until int64
	_probe_log        string

	_last_error LastError
}

type EVMClientinfo struct {
	ID                      uint64
	Endpoint                string
	Is_public_node          bool
	Available_block_last    int
	Available_block_last_ts int64
	Is_disabled             bool
	Is_throttled            bool
	Is_paused               bool

	Attr  EVMClientAttr
	Score int
}

func (this *EVMClient) GetEndpoint() string {
	this.mu.Lock()
	ret := this.endpoint
	this.mu.Unlock()

	return ret
}

func (this *EVMClient) GetInfo() *EVMClientinfo {
	ret := EVMClientinfo{}

	this.mu.Lock()
	ret.ID = this.id
	ret.Endpoint = this.endpoint
	ret.Is_public_node = this.is_public_node
	ret.Is_disabled = this.is_disabled
	ret.Is_paused = this.is_paused
	ret.Available_block_last = this.available_block_last
	ret.Available_block_last_ts = this.available_block_last_ts

	tmp := throttle.ThrottleGoup(this.throttle).GetThrottleScore()
	ret.Score = tmp.Score
	ret.Is_throttled = tmp.Throttled

	ret.Attr = this.attr
	this.mu.Unlock()

	return &ret
}

var new_client_id = uint64(0)

func MakeClient(endpoint string, header http.Header, is_public_node bool, probe_time int, max_conns int, throttle []*throttle.Throttle) *EVMClient {

	tr := &http.Transport{
		MaxIdleConns:       max_conns,
		MaxConnsPerHost:    max_conns,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: true}

	ret := EVMClient{}
	ret.client = &http.Client{Transport: tr, Timeout: 5 * time.Second}
	ret.endpoint = endpoint
	ret.header = header
	ret.is_public_node = is_public_node
	ret._probe_time = probe_time
	ret.stat_total.stat_request_by_fn = make(map[string]int)

	ret.throttle = throttle
	ret._maintenance()

	ret.id = atomic.AddUint64(&new_client_id, 1)
	return &ret
}

func (this *EVMClient) GetThrottleLimitsLeft() (int, int, int, int) {
	this.mu.Lock()
	defer this.mu.Unlock()
	return throttle.ThrottleGoup(this.throttle).GetLimitsLeft()
}
