package gateway

import (
	"sync"
)

type EndpointManager struct {
	k8s        bool //is running in K8s mode?
	endpoints  map[string]*Endpoint
	sbiIndexes map[string]*Endpoint
	wg         sync.WaitGroup //wait for all endpoint's loop
	gw         *Gateway
	mutex      sync.RWMutex
}

func newEndpointManager(c *Gateway) (epman *EndpointManager) {
	epman = &EndpointManager{
		gw:         c,
		endpoints:  make(map[string]*Endpoint),
		sbiIndexes: make(map[string]*Endpoint),
	}
	return
}
func (epman *EndpointManager) findByUuid(id string) *Endpoint {
	epman.mutex.RLock()
	defer epman.mutex.RUnlock()
	ep, _ := epman.endpoints[id]
	return ep
}

func (epman *EndpointManager) findBySbiId(id string) *Endpoint {
	epman.mutex.RLock()
	defer epman.mutex.RUnlock()
	ep, _ := epman.sbiIndexes[id]
	return ep
}

// update endpoint information then add to the list
func (epman *EndpointManager) registerEndpoint(ep *Endpoint, id string) {
	//update information for endpoint
	ep.id = id
	ep.quit = make(chan bool)

	//add to map
	epman.mutex.Lock()
	epman.endpoints[ep.id] = ep
	epman.sbiIndexes[ep.sbiUrl] = ep
	epman.mutex.Unlock()

	//start a goroutine to send pings
	epman.wg.Add(1)
	go ep.loop(&epman.wg)
	return
}

func (epman *EndpointManager) getEndpoints() (endpoints []*Endpoint) {
	epman.mutex.RLock()
	defer epman.mutex.RUnlock()
	if l := len(epman.endpoints); l > 0 {
		endpoints = make([]*Endpoint, l)
		i := 0
		for _, ep := range epman.endpoints {
			endpoints[i] = ep
			i++
		}
	}
	return
}

func (epman *EndpointManager) removeEp(ep *Endpoint) {
	epman.mutex.Lock()
	defer epman.mutex.Unlock()
	delete(epman.endpoints, ep.id)
	delete(epman.sbiIndexes, ep.sbiUrl)
}

func (epman *EndpointManager) close() {
	//get endpoint list
	endpoints := epman.getEndpoints()

	//close (remove) each of them
	for _, ep := range endpoints {
		ep.close()
	}
	epman.wg.Wait()
}
