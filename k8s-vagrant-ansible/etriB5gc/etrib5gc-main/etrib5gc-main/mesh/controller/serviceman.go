package controller

import (
	"etrib5gc/mesh/models"
	"sync"
)

type ServiceManager struct {
	epman    *EndpointManager
	services map[string]*Service //indexed by service name
	rwmutex  sync.RWMutex
}

func newServiceManager(epman *EndpointManager) (sman *ServiceManager) {
	sman = &ServiceManager{
		epman:    epman,
		services: make(map[string]*Service),
	}

	return
}

func (sman *ServiceManager) listAll() (all []*Service) {
	sman.rwmutex.RLock()
	defer sman.rwmutex.RUnlock()
	for _, s := range sman.services {
		all = append(all, s)
	}
	return
}

func (sman *ServiceManager) handleUpdateService(msg models.AddServiceRequest) []string {
	addedServices := []string{}
	for _, sDef := range msg {
		if len(sDef.Id) == 0 {
			continue
		}

		if len(sDef.Selectors) == 0 {
			//fmt.Errorf("Empty service labels")
			continue
		}

		if s := sman.findService(sDef.Id); s != nil {
			//fmt.Errorf("Service %s exists", sDef.Name)
			continue
		}

		s := newService(ServiceConfig{
			Id:        sDef.Id,
			Selectors: sDef.Selectors,
			//Stateful:  sDef.Stateful,
			Stateful: true,
		})
		sman.addService(s)
		addedServices = append(addedServices, sDef.Id)
	}
	return addedServices
}

func (sman *ServiceManager) addService(s *Service) {
	sman.rwmutex.Lock()
	defer sman.rwmutex.Unlock()
	sman.services[s.name] = s
}

func (sman *ServiceManager) findService(name string) (s *Service) {
	sman.rwmutex.RLock()
	defer sman.rwmutex.RUnlock()
	s, _ = sman.services[name]
	return
}

// an endpoint subscribes a list of services
func (sman *ServiceManager) addSubscriber(ep *Endpoint, services []string) (validServices []*Service) {
	sman.rwmutex.Lock()
	defer sman.rwmutex.Unlock()
	//make unique service list
	uServices := make(map[string]struct{})
	for _, s := range services {
		uServices[s] = struct{}{}
	}
	//only subscribe to existing services
	for sId, _ := range uServices {
		if service, ok := sman.services[sId]; ok {
			//subscribe the endpoint
			service.addSub(ep)
			//add the subscibed service
			ep.services[sId] = service
			validServices = append(validServices, service)
		} else {
			ep.Warnf("Service %s not exist to subscribe", sId)
		}
	}
	return
}

// get all subscribers of services served by the endpoint
func (sman *ServiceManager) getSubscribers(services []*Service) (endpoints []*Endpoint) {
	sman.rwmutex.RLock()
	defer sman.rwmutex.RUnlock()
	//use map to create set of unique endpoints
	list := make(map[string]*Endpoint)
	for _, service := range services { //for each service the endpoin serving
		for _, ep := range service.getSubscribers() { //for each endpoint that subscribes to the service
			list[ep.id] = ep
		}
	}
	//turn into an array
	for _, ep := range list {
		endpoints = append(endpoints, ep)
	}
	return
}

// return all services that the endpoint serves
func (sman *ServiceManager) getServedServices(ep *Endpoint) (services []*Service) {
	sman.rwmutex.RLock()
	defer sman.rwmutex.RUnlock()
	for _, service := range sman.services {
		if ep.isServing(service) {
			services = append(services, service)
		}
	}
	return
}
