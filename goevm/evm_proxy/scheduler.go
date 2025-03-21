package evm_proxy

import (
	"goevm/evm_proxy/client"
	"sort"
)

type scheduler struct {
	min_block_no  int
	clients       []*client.EVMClient
	force_public  bool
	force_private bool
}

func MakeScheduler() *scheduler {

	mu.RLock()
	tmp := make([]*client.EVMClient, len(clients))
	copy(tmp, clients)
	mu.RUnlock()

	ret := &scheduler{min_block_no: -1, clients: tmp}
	ret.force_public = false
	ret.force_private = false
	return ret
}

func (this *scheduler) SetMinBlock(min_block_no int) {
	this.min_block_no = min_block_no
}

func (this *scheduler) ForcePublic(f bool) {
	this.force_public = f
	if this.force_public && this.force_private {
		this.force_public = false
		this.force_private = false
	}
}
func (this *scheduler) ForcePrivate(f bool) {
	this.force_private = f
	if this.force_public && this.force_private {
		this.force_public = false
		this.force_private = false
	}
}

/* Gets client, prioritize private client */
func (this *scheduler) GetAnyClient() *client.EVMClient {
	return this._pick_next()
}

/* Get public client only */
func (this *scheduler) GetPublicClient() *client.EVMClient {

	// we forced something, so override the client returned
	if this.force_public || this.force_private {
		return this._pick_next()
	}

	this.force_public = true
	ret := this._pick_next()
	this.force_public = false
	return ret
}

func (this *scheduler) GetAll(is_public bool, include_deactivated bool) []*client.EVMClient {

	ret := make([]*client.EVMClient, 0, len(this.clients))
	for _, v := range this.clients {
		info := v.GetInfo()
		if (info.Is_disabled || info.Is_throttled || info.Is_paused) && include_deactivated == false {
			continue
		}

		if is_public != info.Is_public_node {
			continue
		}
		ret = append(ret, v)
	}
	return ret
}

func (this *scheduler) GetAllSorted(is_public bool, include_disabled bool) []*client.EVMClient {

	ret := this.GetAll(is_public, include_disabled)
	type r_sort struct {
		c     *client.EVMClient
		score int
		block int
	}
	s := make([]r_sort, 0, len(ret))
	for _, v := range ret {
		info := v.GetInfo()
		s = append(s, r_sort{v, info.Score, info.Available_block_last})
	}
	sort.Slice(s, func(a, b int) bool {
		if s[a].score == s[b].score {
			return s[a].block > s[b].block
		}
		return s[a].score < s[b].score
	})
	for k, v := range s {
		ret[k] = v.c
	}

	return ret
}
