package registry

import (
	"etrib5gc/mesh/models"
	sbimodels "github.com/reogac/sbi/models"
	"sync"
)

type EpManager struct {
	endpoints    map[string]*Endpoint
	epInsIndexes map[string]*Endpoint
	reg          *Registry
	onEpLeft     func(string)
	onEpJoin     func(sbimodels.EndpointInfo, []string, []byte)
	mutex        sync.RWMutex
}

func newEpManager(reg *Registry) EpManager {
	return EpManager{
		reg:          reg,
		endpoints:    make(map[string]*Endpoint),
		epInsIndexes: make(map[string]*Endpoint),
	}
}

// find and endpoint with address, or create a new one if not found
func (man *EpManager) createEndpointWithUuid(uuid string) *Endpoint {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	if ep, ok := man.endpoints[uuid]; ok {
		log.Infof("Existing endpoint found for a callback")
		return ep
	} else {
		return man.reg.discoverEndpoint(uuid)
	}
}

func (man *EpManager) getEndpoints() (list []*Endpoint) {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	for _, ep := range man.endpoints {
		list = append(list, ep)
	}
	return
}

// add new endpoints and update services
func (man *EpManager) addEndpoints(list map[string]models.Endpoint) {
	man.mutex.Lock()
	defer man.mutex.Unlock()
	for id, info := range list {
		//get relevant services
		services := man.reg.sman.getServices(Labels(info.Labels))
		if len(services) == 0 {
			continue //Endpoint belongs no service
		}

		sIds := []string{} //list of service ids
		for _, s := range services {
			sIds = append(sIds, s.id)
		}
		//create new endpoint if it is not in existing list
		if _, ok := man.endpoints[id]; !ok {
			ep := man.reg.createEndpoint(info)
			if ep == nil {
				//endpoint can not be created due to invalid information
				log.Warnf("Failed to create endpoint %s[%s]", info.Id, info.GwId)
				continue
			}

			if len(info.InsId) > 0 {
				log.Infof("Endpoint discovered [%s:%s]", id, info.InsId)
			} else {
				log.Infof("Endpoint discovered [%s]", id)
			}
			if man.onEpJoin != nil {
				man.onEpJoin(ep.EndpointInfo(), sIds, info.Config)
			}

			//add endpoint to list
			if len(ep.id) > 0 {
				man.endpoints[ep.id] = ep
			}
			if len(ep.insId) > 0 {
				man.endpoints[ep.insId] = ep
			}

			//binding relevant services and the enpoint
			for _, s := range services {
				s.addEndpoint(ep)
			}
		}
		//NOTE: nothing change with existing endpoints
	}
}

func (man *EpManager) updateEndpoints(msg *models.EndpointUpdates) {

	//delete endpoints
	man.mutex.Lock()
	for _, epId := range msg.Left {
		log.Tracef("Endpoint %s is dead", epId)
		if ep, ok := man.endpoints[epId]; ok {
			ep.unbindServices()
			delete(man.endpoints, epId)
			delete(man.epInsIndexes, ep.insId)

			log.Infof("Endpoint %s is removed", ep.id)
			if man.onEpLeft != nil {
				man.onEpLeft(epId)
			}
		}
	}
	man.mutex.Unlock()

	//add endpoints
	newEndpoints := make(map[string]models.Endpoint)
	for _, item := range msg.Join {
		newEndpoints[item.Id] = item
	}
	man.addEndpoints(newEndpoints)
}

func (man *EpManager) findEndpointByUuid(uuid string) *Endpoint {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	ep, _ := man.endpoints[uuid]
	return ep
}

func (man *EpManager) listEndpoints() (list []*Endpoint) {
	man.mutex.RLock()
	defer man.mutex.RUnlock()
	for _, ep := range man.endpoints {
		list = append(list, ep)
	}
	return
}
