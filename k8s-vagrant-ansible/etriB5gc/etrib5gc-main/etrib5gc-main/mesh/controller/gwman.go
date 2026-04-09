package controller

import (
	"etrib5gc/mesh/models"
	"sync"
)

type GatewayManager struct {
	ctrl        *Controller
	gateways    map[string]*Gateway
	addrIndexes map[string]*Gateway
	mutex       sync.RWMutex
	wg          sync.WaitGroup
}

func newGatewayManager(ctrl *Controller) *GatewayManager {
	return &GatewayManager{
		ctrl:        ctrl,
		gateways:    make(map[string]*Gateway),
		addrIndexes: make(map[string]*Gateway),
	}
}

func (man *GatewayManager) findByUuid(id string) *Gateway {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	gw, _ := man.gateways[id]
	return gw
}

func (man *GatewayManager) registerGw(req *models.GwRegistrationRequest) (rsp *models.GwRegistrationResponse, err error) {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	var gw *Gateway
	var ok bool
	if gw, ok = man.addrIndexes[req.Url]; !ok {
		gw = man.ctrl.newGateway(req)
		man.wg.Add(1)
		go gw.loop(&man.wg)

		man.gateways[gw.id] = gw
		man.addrIndexes[gw.url] = gw
	}
	log.Infof("Gateway %s registered", gw.url)
	rsp = &models.GwRegistrationResponse{
		Id: gw.id,
	}

	return
}

func (man *GatewayManager) removeGw(gw *Gateway) {
	man.mutex.Lock()
	defer man.mutex.Unlock()
	log.Infof("Gateway %s is removed", gw.url)
	delete(man.gateways, gw.id)
	delete(man.addrIndexes, gw.url)
	endpoints := gw.getEndpoints()
	man.ctrl.epman.removeEndpoints(endpoints)
}

func (man *GatewayManager) close() {
	//get gateway list
	gateways := man.getGateways()

	//close (remove) each of them
	for _, gw := range gateways {
		gw.close()
	}
	man.wg.Wait()
}

func (man *GatewayManager) getGateways() (gateways []*Gateway) {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	if l := len(man.gateways); l > 0 {
		gateways = make([]*Gateway, l)
		i := 0
		for _, gw := range man.gateways {
			gateways[i] = gw
			i++
		}
	}
	return
}
