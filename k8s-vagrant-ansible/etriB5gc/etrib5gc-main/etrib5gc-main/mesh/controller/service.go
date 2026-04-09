package controller

import (
	"sync"
)

type Service struct {
	name        string
	selectors   Selectors
	stateful    bool
	subscribers map[string]*Endpoint //endpoints subscribe to the service
	endpoints   map[string]*Endpoint //endpoins serving the service
	mutex       sync.Mutex
}

func newService(info ServiceConfig) (s *Service) {
	s = &Service{
		name:        info.Id,
		stateful:    info.Stateful,
		selectors:   info.Selectors,
		subscribers: make(map[string]*Endpoint),
		endpoints:   make(map[string]*Endpoint),
	}
	return
}

func (s *Service) numEndpoints() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.endpoints)
}

func (s *Service) numSubscribers() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.subscribers)
}

func (s *Service) addEndpoint(ep *Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.endpoints[ep.id] = ep
}

func (s *Service) removeEndpoint(ep *Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.endpoints, ep.id)
}

func (s *Service) addSub(ep *Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Debugf("Add %s as a subscriber to %s", ep.id, s.name)
	s.subscribers[ep.id] = ep
}

func (s *Service) removeSub(ep *Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.subscribers, ep.id)
	log.Debugf("Remove subscriber %s from %s", ep.id, s.name)
}

func (s *Service) getSubscribers() (endpoints []*Endpoint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, subs := range s.subscribers {
		endpoints = append(endpoints, subs)
	}
	return
}
