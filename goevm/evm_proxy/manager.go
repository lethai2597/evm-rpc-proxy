package evm_proxy

import (
	"goevm/evm_proxy/client"
	"math"
	"sync"
)

var mu sync.RWMutex
var clients []*client.EVMClient

func init() {
	clients = make([]*client.EVMClient, 0, 10)
}

func ClientRegister(c *client.EVMClient) {
	ClientManage(c, math.MaxUint64)
}

func ClientRemove(id uint64) bool {
	return ClientManage(nil, id)
}

func ClientManage(add *client.EVMClient, removeClientID uint64) bool {
	acted := false

	mu.Lock()
	defer mu.Unlock()

	if add != nil && removeClientID == math.MaxUint64 {
		clients = append(clients, add)
		return true
	}

	tmp := make([]*client.EVMClient, 0, len(clients))
	for _, client := range clients {
		if client.GetInfo().ID == removeClientID {
			acted = true
			if add != nil {
				tmp = append(tmp, add)
				add = nil
			}
			continue
		}
		tmp = append(tmp, client)
	}
	if add != nil {
		tmp = append(tmp, add)
	}

	clients = tmp
	return acted
}
