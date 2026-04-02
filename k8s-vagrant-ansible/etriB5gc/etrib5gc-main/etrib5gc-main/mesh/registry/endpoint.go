package registry

import (
	"etrib5gc/mesh/models"
	"fmt"
	sbimodels "github.com/reogac/sbi/models"
	"github.com/reogac/utils/httpw"
	"time"
)

type Endpoints []*Endpoint //endpoint list

type RouteInfo struct {
	EpId string
	GwId string
}

type EpClient struct {
	Cli   *httpw.Client
	Addr  string
	Route *RouteInfo
}

type Endpoint struct {
	id        string //generated id
	gwId      string
	sbiUrl    string //registered sbi URL
	sbiPrefix string
	insId     string //instance identity
	labels    Labels //pod labels
	services  map[string]*Service

	cli *EpClient

	created time.Time
}

func (reg *Registry) createEndpoint(def models.Endpoint) *Endpoint {
	ep := &Endpoint{
		id:        def.Id,
		gwId:      def.GwId,
		sbiUrl:    def.SbiUrl,
		sbiPrefix: def.SbiPrefix,
		insId:     def.InsId,
		services:  make(map[string]*Service),
		labels:    Labels(def.Labels),
		created:   time.Now(),
	}

	if len(ep.gwId) == 0 {
		log.Errorf("Endpoint info has no gateway")
		//endpoint has no attached gateway
		return nil
	}

	//determine the address as endpoint identity
	useGw := true
	if ep.gwId == reg.gwId { //same gateway
		if reg.isReachable(ep.sbiUrl) {
			addr := ep.sbiUrl
			if len(ep.sbiPrefix) > 0 {
				addr = fmt.Sprintf("%s/%s", ep.sbiUrl, ep.sbiPrefix)
			}
			ep.cli = &EpClient{
				Addr: addr,
				Cli:  httpw.NewClient(reg.cert, reg.caPool, def.Name),
			}
			useGw = false
		} else {
			log.Tracef("%s is not reachable", ep.sbiUrl)
		}
	}
	if useGw { //always use the agent's gateway to send requests to remote endpoint (on another gateway)
		log.Tracef("Add remote endpoint %s[%s]", ep.id, ep.gwId)
		ep.cli = &EpClient{
			Addr: fmt.Sprintf("%s/sbi", reg.gwAddr),
			Cli:  reg.cli,
			Route: &RouteInfo{
				EpId: ep.id,
				GwId: ep.gwId,
			},
		}
	} else { //send requests directly to local endpoint
		log.Tracef("Add local endpoint %s[%s]", ep.id, ep.gwId)
	}
	return ep
}

func (ep *Endpoint) Id() string {
	return ep.id
}

func (ep *Endpoint) addService(s *Service) {
	ep.services[s.id] = s
}

func (ep *Endpoint) EndpointInfo() (ret sbimodels.EndpointInfo) {
	ret = sbimodels.EndpointInfo{
		Uuid: ep.id,
	}
	return
}

func (ep *Endpoint) unbindServices() {
	//unbind ep from services
	for _, s := range ep.services {
		s.removeEndpoint(ep)
	}
	//clear services
	ep.services = make(map[string]*Service)
}

// create endpoint with callback information
// NOTE: currently we don't have a mechanism to remove the endpoint if it is dead
func (reg *Registry) newEndpoint(uuid string, gwId string) (ep *Endpoint) {
	ep = &Endpoint{
		id:      uuid,
		gwId:    gwId,
		created: time.Now(),
		cli: &EpClient{
			Addr: fmt.Sprintf("%s/sbi", reg.gwAddr),
			Cli:  reg.cli,
			Route: &RouteInfo{
				EpId: uuid,
				GwId: gwId,
			},
		},
	}
	//TODO:initialize needed statistics
	return
}

func (ep *Endpoint) Client() *EpClient {
	return ep.cli
}
