package controller

import (
	"etrib5gc/mesh/lbs"
	"etrib5gc/mesh/models"
	"fmt"
	"github.com/reogac/utils/idgen"
	"math"
	"sync"
)

type Destination struct {
	groupId string
	weight  uint16
	lb      lbs.LoadBalancerType
}

type RouteMatch map[string]string

type Route struct {
	id      uint16 //generated
	idStr   string
	service string
	match   RouteMatch
	targets []Destination
}

func (r *Route) setId(id uint16) {
	r.id = id
	r.idStr = fmt.Sprintf("%s-route-%d", r.service, r.id)
}

// an Routing binds a service to an ordered list of Routes
type Routing struct {
	service string
	routes  []*Route
}

type RoutingList struct {
	bindings   map[string]*Routing //indexed by service id
	routes     map[uint16]*Route
	routeIdGen idgen.Generator[uint16]
	mutex      sync.RWMutex
}

func newRoutingList() RoutingList {
	return RoutingList{
		bindings:   make(map[string]*Routing),
		routes:     make(map[uint16]*Route),
		routeIdGen: idgen.NewGenerator[uint16](0, math.MaxUint16),
	}
}

func (rl *RoutingList) addRoute(r *Route) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	r.setId(rl.routeIdGen.Allocate())
	rl.routes[r.id] = r
	if routing, ok := rl.bindings[r.service]; ok {
		//add to the end of route list
		routing.routes = append(routing.routes, r)
	} else {
		//create new binding then add the route to the first of the route list
		rl.bindings[r.service] = &Routing{
			service: r.service,
			routes:  []*Route{r},
		}
	}
}

func (rl *RoutingList) getRouteIds(service string) (ids []int) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	if routing, ok := rl.bindings[service]; ok {
		for _, r := range routing.routes {
			ids = append(ids, int(r.id))
		}
	}
	return
}

// build Route information for a service
func (rl *RoutingList) buildRouteInfo(service string) (list []models.Route) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	if routing, ok := rl.bindings[service]; ok {
		for _, route := range routing.routes {
			routeInfo := models.Route{
				Match: make(map[string]string),
			}
			for k, v := range route.match {
				routeInfo.Match[k] = v
			}
			for _, d := range route.targets {
				routeInfo.Destinations = append(routeInfo.Destinations, models.Destination{
					GroupId: d.groupId,
					Weight:  int(d.weight),
					Lb:      d.lb.String(),
				})
			}
			list = append(list, routeInfo)
		}
	}
	return
}

func (rl *RoutingList) listRoutes() (all []*Route) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	for _, r := range rl.routes {
		all = append(all, r)
	}
	return
}
