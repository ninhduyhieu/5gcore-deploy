package registry

import (
	"etrib5gc/mesh/models"
	"sync"
)

type ServiceManager struct {
	services map[string]*Service
	mutex    sync.RWMutex
	epMan    *EpManager
}

func newServiceManager(epMan *EpManager) (sman ServiceManager) {
	sman = ServiceManager{
		epMan:    epMan,
		services: make(map[string]*Service),
	}
	return
}

func (sman *ServiceManager) findService(id string) *Service {
	sman.mutex.RLock()
	defer sman.mutex.RUnlock()
	s, _ := sman.services[id]
	return s
}

// add newlysubscribed services
// add existing endpoints
func (sman *ServiceManager) addServices(items map[string]models.Service) {
	sman.mutex.Lock()
	defer sman.mutex.Unlock()

	//existing endpoints
	endpoints := sman.epMan.getEndpoints()

	//create and add services
	for _, info := range items {
		if len(info.Id) == 0 {
			log.Warnf("A service with empty name")
			continue
		}
		if len(info.Selectors) == 0 {
			log.Warnf("Service %s has no selector", info.Id)
			continue
		}
		if _, ok := sman.services[info.Id]; ok {
			//service already existed, do nothing
			continue
		}

		log.Infof("Subscribed to service %s", info.Id)
		service := newService(info)
		sman.services[info.Id] = service
		//add existing endpoints and build groups
		service.addEndpoints(endpoints)
	}
}

// match enpoint labels against services then return the matched services
func (sman *ServiceManager) getServices(labels Labels) (services []*Service) {
	sman.mutex.Lock()
	defer sman.mutex.Unlock()
	for _, s := range sman.services {
		if s.selectors.match(labels) {
			services = append(services, s)
		}
	}
	return
}
