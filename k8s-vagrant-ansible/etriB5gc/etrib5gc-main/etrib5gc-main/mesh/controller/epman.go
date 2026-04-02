package controller

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/models"
	"github.com/sirupsen/logrus"
	"sync"
)

type EndpointManager struct {
	k8s       bool //is running in K8s mode?
	endpoints map[string]*Endpoint
	mutex     sync.RWMutex
	ctrl      *Controller
}

func newEndpointManager(c *Controller) (epman *EndpointManager) {
	epman = &EndpointManager{
		ctrl:      c,
		endpoints: make(map[string]*Endpoint),
	}
	return
}

func (epman *EndpointManager) findByUuid(id string) *Endpoint {
	epman.mutex.RLock()
	defer epman.mutex.RUnlock()
	ep, _ := epman.endpoints[id]
	return ep
}

func (epman *EndpointManager) registerEndpoint(msg *models.RegistrationFwRequest, gw *Gateway) (rsp *models.RegistrationResponse, err error) {

	var ep *Endpoint

	ep = epman.ctrl.createEndpoint(gw)
	for k, v := range msg.Labels {
		ep.labels[k] = v
	}
	ep.Entry = logctx.Entry(logrus.Fields{
		"endpoint": ep.id,
	})
	//update endpoint information
	ep.sbiActive = msg.Active
	ep.servings = epman.ctrl.sman.getServedServices(ep)
	for _, s := range ep.servings {
		s.addEndpoint(ep)
	}
	ep.config = msg.Config
	//add to map
	epman.mutex.Lock()
	epman.endpoints[ep.id] = ep
	epman.mutex.Unlock()
	//add to gateway
	ep.gw.addEp(ep)
	//send notification to subscribers
	if ep.sbiActive {
		epman.ctrl.sendEpEvent(EP_JOIN, ep)
	}
	log.Infof("Endpoint %s registered", ep.id)
	rsp = &models.RegistrationResponse{
		Id: ep.id,
	}
	return
}

func (epman *EndpointManager) activateEndpoint(uuid string, msg *models.UpdateNotification) (err error) {
	var ep *Endpoint
	//find the endpoint with its identity
	if ep = epman.findByUuid(uuid); ep == nil {
		log.Errorf("Endpoint %s not found", uuid)
		return
	}

	ep.insId = msg.InsId
	ep.sbiActive = true

	ep.Infof("Endpoint activated")
	epman.ctrl.sendEpEvent(EP_JOIN, ep)
	return
}
func (epman *EndpointManager) removeEp(ep *Endpoint) {
	epman.mutex.Lock()
	ep.Infof("Endpoint removed")
	delete(epman.endpoints, ep.id)
	epman.mutex.Unlock()

	//remove endpoint as a subscriber
	ep.unsubscribeServices()

	//remove endpoint from services
	ep.detachServices()

	//detach from gateway
	ep.gw.removeEp(ep)
	//send notification to subscribers
	epman.ctrl.sendEpEvent(EP_LEFT, ep)
}

// remove endpoints of a gateway that is dead
func (epman *EndpointManager) removeEndpoints(endpoints []*Endpoint) {
	//remove endpoints from the main pool and the services
	epman.mutex.Lock()
	for _, ep := range endpoints {
		ep.Infof("Endpoint removed")
		delete(epman.endpoints, ep.id)
		ep.unsubscribeServices()
	}
	epman.mutex.Unlock()
	//notify subscribers of the left endpoints
	//TODO: push to a worker pool
	for _, ep := range endpoints {
		ep.Tracef("Notify subscribers")
		for _, sub := range epman.ctrl.sman.getSubscribers(ep.servings) {
			ep.Tracef("Notify %s", sub.id)
			sub.notifyEpLeft(ep)
		}

	}
}

// get endpoints serving subscribed services (for an endpoint)
func (epman *EndpointManager) getEndpoints(services []*Service) (endpoints []*Endpoint) {
	//get all endpoints
	all := epman.getAllEndpoints()
	for _, ep := range all {
		//for each subscribed service
		for _, s := range services {
			if ep.sbiActive && ep.isServing(s) {
				//add endpoint if it serve a subscribed service
				endpoints = append(endpoints, ep)
				break
			}
		}
	}
	return
}

func (epman *EndpointManager) getAllEndpoints() (endpoints []*Endpoint) {
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
