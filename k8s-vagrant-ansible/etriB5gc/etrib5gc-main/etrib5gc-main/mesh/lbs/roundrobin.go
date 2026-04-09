package lbs

import (
	"sync"
)

// instance for round robin loadbalancer implementation
type rrLbInstance struct {
	ep    Endpoint
	index int //instance index
}

// TODO: implement round robin load balancer
type roundRobinLB struct {
	instances map[string]*rrLbInstance
	rwmutex   sync.RWMutex
	curIndex  int //current picked index
}

func createRoundRobinLB() LoadBalancer {
	lb := &roundRobinLB{
		instances: make(map[string]*rrLbInstance),
	}
	return lb
}

func (lb *roundRobinLB) AddEndpoint(ep Endpoint) {
	lb.rwmutex.Lock()
	defer lb.rwmutex.Unlock()
	//TODO:
}

func (lb *roundRobinLB) RemoveEndpoint(ep Endpoint) {
	lb.rwmutex.Lock()
	defer lb.rwmutex.Unlock()
	//TODO
}

func (lb *roundRobinLB) PickEndpoint() Endpoint {
	lb.rwmutex.RLock()
	defer lb.rwmutex.RUnlock()
	//TODO:
	return nil
}

func (lb *roundRobinLB) NumEndpoints() int {
	lb.rwmutex.RLock()
	defer lb.rwmutex.RUnlock()
	return len(lb.instances)
}
