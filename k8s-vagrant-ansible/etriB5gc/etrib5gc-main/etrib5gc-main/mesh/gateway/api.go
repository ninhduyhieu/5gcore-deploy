package gateway

import (
	"etrib5gc/mesh/gateway/oambackend"
	"time"
)

type oamApi struct {
	gw *Gateway
}

func (ep *Endpoint) ToEndpointInfo() oambackend.EndpointInfo {
	return oambackend.EndpointInfo{
		Uuid:     ep.id,
		AgentUrl: ep.agentUrl,
		SbiUrl:   ep.sbiUrl,
		Labels:   ep.labels,
		RegTime:  ep.regTime.Format(time.RFC1123),
		LastSeen: ep.lastSeen.Format(time.RFC1123),
	}
}

func (api *oamApi) GetEndpointInfos() []oambackend.EndpointInfo {
	epman := api.gw.epman
	endpoints := epman.getEndpoints()
	infos := []oambackend.EndpointInfo{}
	for _, ep := range endpoints {
		infos = append(infos, ep.ToEndpointInfo())
	}
	return infos
}
