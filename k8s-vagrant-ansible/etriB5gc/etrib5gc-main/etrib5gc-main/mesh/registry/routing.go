package registry

import (
	"etrib5gc/mesh/lbs"
	"etrib5gc/mesh/models"
	"math/rand"
)

type Labels map[string]string

func newLabels() Labels {
	return Labels(make(map[string]string))
}

type Selectors map[string]string

func newSelectors() Selectors {
	return Selectors(make(map[string]string))
}

func (s Selectors) match(labels Labels) (ok bool) {
	for k, v1 := range s {
		if v2, ok := labels[k]; !ok || v1 != v2 {
			return false
		}
	}

	return true
}

type RouteMatch map[string]string

func (rm RouteMatch) isMatched(options map[string]string) bool {
	if len(rm) == 0 {
		return true
	}

	for k, v1 := range rm {
		if v2, ok := options[k]; !ok || v1 != v2 {
			return false
		}
	}
	return true
}

type Target struct {
	//group  *EpGroup
	weight int
	lb     lbs.LoadBalancer
}

func (t *Target) selectEndpoint() (ep *Endpoint) {
	if picked := t.lb.PickEndpoint(); picked != nil {
		ep, _ = picked.(*Endpoint)
	}
	return
}

type EpGroup struct {
	service   *Service
	selectors Selectors
	targets   []lbs.LoadBalancer
}

func (g *EpGroup) addTarget(lb lbs.LoadBalancer) {
	g.targets = append(g.targets, lb)
}

func (g *EpGroup) remove(ep *Endpoint) {
	for _, t := range g.targets {
		t.RemoveEndpoint(ep)
	}
}

func (g *EpGroup) add(ep *Endpoint) {
	if g.selectors.match(ep.labels) {
		for _, t := range g.targets {
			t.AddEndpoint(ep)
		}
	}
}

func newEpGroup(s *Service, selectors Selectors) *EpGroup {
	return &EpGroup{
		service:   s,
		selectors: selectors,
	}
}

type Route struct {
	service *Service
	match   RouteMatch
	targets []Target
}

func (r *Route) selectEndpoint() (ep *Endpoint) {
	//get targets having endpoint
	targets := []Target{}
	for _, t := range r.targets {
		if t.lb.NumEndpoints() > 0 {
			targets = append(targets, t)
		}
	}
	num := len(targets)
	if num == 0 {
		log.Warnf("skip route %v without target having endpoints", r.match)
		return
	}
	//now pick one target
	var selectedTarget Target
	if num == 1 {
		selectedTarget = targets[0]
	} else {
		wmarkers := make([]float64, num)
		wmarkers[0] = float64(targets[0].weight)
		for i := 1; i < num; i++ {
			wmarkers[i] = wmarkers[i-1] + float64(targets[i].weight)
		}

		if wmarkers[num-1] <= 0 {
			log.Errorf("Wrong target weights for route %v", r.match)
			return
		}

		for i := 0; i < num; i++ {
			wmarkers[i] /= wmarkers[num-1]
		}
		tmp := rand.Float64() //0 to 1

		var targetId int
		for targetId = 0; targetId < num; targetId++ {
			if wmarkers[targetId] > tmp {
				break
			}
		}
		selectedTarget = targets[targetId]
	}

	//pick one endpoint from selected target
	ep = selectedTarget.selectEndpoint()
	return
}

func (r *Route) isMatched(options map[string]string) bool {
	return r.match.isMatched(options)
}

func (s *Service) createRoute(def models.Route) *Route {
	r := &Route{
		service: s,
		match:   make(map[string]string),
	}
	for k, v := range def.Match {
		r.match[k] = v
	}
	for _, d := range def.Destinations {
		//add only target linked to a valid group
		if g, ok := s.groups[d.GroupId]; ok {
			//create a load balancer for the target
			lbType := lbs.ParseLoadBalancerType(d.Lb)
			lb := lbs.CreateLoadBalancer(lbType)
			r.targets = append(r.targets, Target{
				//group:  g,
				weight: d.Weight,
				lb:     lb,
			})
			//add loadbalancer to group for receiving updates of endpoint
			//events (left/join)
			g.addTarget(lb)
		}
	}
	if len(r.targets) == 0 {
		return nil
	}
	return r
}

type MatchedGroup struct {
	endpoint *Endpoint
	service  *Service
	options  map[string]string
}

func (m *MatchedGroup) IsStateful() bool {
	return m.service == nil
}

func (m *MatchedGroup) Select() *Endpoint {
	if m.service == nil {
		return m.endpoint
	}
	return m.service.selectEndpoint(m.options)
}
