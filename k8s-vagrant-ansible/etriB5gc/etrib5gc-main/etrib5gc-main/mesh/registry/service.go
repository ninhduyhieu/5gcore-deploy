package registry

import (
	"etrib5gc/mesh/lbs"
	"etrib5gc/mesh/models"
	"fmt"
	"sync"
)

type Service struct {
	id        string //service identity
	selectors Selectors
	stateful  bool
	groups    map[string]*EpGroup
	routes    []*Route
	lb        lbs.LoadBalancer //default load balancer

	endpoints    map[string]*Endpoint //Ep's uuid to endpoint
	epInsIndexes map[string]*Endpoint //Ep's instance Id to endpoint
	mutex        sync.RWMutex
}

func newService(def models.Service) (s *Service) {
	s = &Service{
		id:           def.Id,
		selectors:    Selectors(def.Selectors),
		stateful:     def.Stateful,
		endpoints:    make(map[string]*Endpoint),
		epInsIndexes: make(map[string]*Endpoint),
		groups:       make(map[string]*EpGroup),
		lb:           lbs.CreateLoadBalancer(lbs.LB_TYPE_RANDOM),
	}
	for gname, selectors := range def.Groups {
		s.groups[gname] = newEpGroup(s, Selectors(selectors))
	}

	//add routes
	for _, info := range def.Routes {
		log.Infof("Add route: %v for service %s", info, s.id)
		if route := s.createRoute(info); route != nil {
			s.routes = append(s.routes, route)
		}
	}

	return
}

func getServiceId(s *Service) string {
	return s.id
}

//find then add matched endpoints
func (s *Service) addEndpoints(endpoints []*Endpoint) {
	for _, ep := range endpoints {
		if s.selectors.match(ep.labels) {
			s.addEndpoint(ep)
		}
	}
}

// add a matched endpoint
func (s *Service) addEndpoint(ep *Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Tracef("Add an endpoint[%s] to service %s", ep.id, s.id)
	//add service to endpoint
	ep.addService(s)
	//add endpoint to service
	s.endpoints[ep.id] = ep
	if len(ep.insId) > 0 {
		s.epInsIndexes[ep.insId] = ep
	}
	//include endpoint to groups if matching
	for _, g := range s.groups {
		g.add(ep)
	}
	//add to default load balancer
	s.lb.AddEndpoint(ep)
}

func (s *Service) removeEndpoint(ep *Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	//remove from endpoint list
	delete(s.endpoints, ep.id)
	if len(ep.insId) > 0 {
		delete(s.epInsIndexes, ep.insId)
	}
	//remove from groups
	for _, g := range s.groups {
		g.remove(ep)
	}
	s.lb.RemoveEndpoint(ep)
}

func (s *Service) searchInstance(id string) *Endpoint {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	ep, _ := s.epInsIndexes[id]
	return ep
}

// find a routing destination
func (s *Service) match(options map[string]string) (m *MatchedGroup, err error) {
	if len(options) > 0 {
		log.Tracef("Search service %s for endpoints with options: %v", s.id, options)
	}
	if s.stateful {
		var ep *Endpoint
		if ep = s.selectEndpoint(options); ep != nil {
			m = &MatchedGroup{
				endpoint: ep,
			}
		} else {
			err = fmt.Errorf("No endpoint")
		}
	} else { //move selection to future
		m = &MatchedGroup{
			service: s,
			options: options,
		}
	}
	return
}

func (s *Service) selectEndpoint(options map[string]string) (ep *Endpoint) {
	for _, route := range s.routes {
		log.Tracef("Match agains %v", route.match)
		if route.isMatched(options) {
			if ep = route.selectEndpoint(); ep != nil {
				log.Tracef("%s picked from a route", ep.id)
				return
			}
		}
	}

	if picked := s.lb.PickEndpoint(); picked != nil {
		ep, _ = picked.(*Endpoint)
		log.Tracef("%s picked from default group for service %s", ep.id, s.id)
	} else {
		log.Errorf("Can't pick any from default group for service %s[%d endpoints]", s.id, s.lb.NumEndpoints())
	}
	return
}
