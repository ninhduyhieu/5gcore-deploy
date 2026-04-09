package lbs

import (
	"math/rand"
	"sync"
)

type randLbInstance struct {
	ep Endpoint
}

type randomLB struct {
	instances map[string]*randLbInstance
	rwmutex   sync.RWMutex
}

func (lb *randomLB) AddEndpoint(ep Endpoint) {
	lb.rwmutex.Lock()
	defer lb.rwmutex.Unlock()
	if _, ok := lb.instances[ep.Id()]; !ok {
		ins := &randLbInstance{
			ep: ep,
		}
		lb.instances[ep.Id()] = ins
	}
}

func (lb *randomLB) RemoveEndpoint(ep Endpoint) {
	lb.rwmutex.Lock()
	defer lb.rwmutex.Unlock()
	delete(lb.instances, ep.Id())
}

func (lb *randomLB) PickEndpoint() Endpoint {
	lb.rwmutex.RLock()
	defer lb.rwmutex.RUnlock()
	list := []*randLbInstance{}
	for _, ins := range lb.instances {
		list = append(list, ins)
	}
	if len(list) > 0 {
		index := rand.Intn(len(list))
		return list[index].ep
	}
	return nil
}

func (lb *randomLB) NumEndpoints() int {
	lb.rwmutex.RLock()
	defer lb.rwmutex.RUnlock()
	return len(lb.instances)
}

func createRandomLB() LoadBalancer {
	lb := &randomLB{
		instances: make(map[string]*randLbInstance),
	}
	return lb
}
