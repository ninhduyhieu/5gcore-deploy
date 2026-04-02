package controller

import (
	"etrib5gc/mesh/controller/oambackend"
	"time"
)

type oamApi struct {
	ctrl *Controller
}

func (ep *Endpoint) toEndpointInfo() (info oambackend.EndpointInfo) {
	info = oambackend.EndpointInfo{
		Uuid:    ep.id,
		GwId:    ep.gw.id,
		Labels:  ep.labels,
		RegTime: ep.regTime.Format(time.RFC1123),
	}
	for _, s := range ep.servings {
		info.OfferingServices = append(info.OfferingServices, s.name)
	}
	for _, s := range ep.services {
		info.SubscribedServices = append(info.SubscribedServices, s.name)
	}

	return
}

func (gw *Gateway) toGatewayInfo() (info oambackend.GatewayInfo) {
	info = oambackend.GatewayInfo{
		Uuid:     gw.id,
		Labels:   gw.labels,
		Name:     gw.name,
		RegTime:  gw.regTime.Format(time.RFC1123),
		LastSeen: gw.lastSeen.Format(time.RFC1123),
		Url:      gw.url,
	}

	for _, ep := range gw.getEndpoints() {
		info.Endpoints = append(info.Endpoints, ep.id)
	}
	return
}

func (s *Service) toServiceInfo(c *Controller) (info oambackend.ServiceInfo) {
	info = oambackend.ServiceInfo{
		Name:           s.name,
		Selectors:      s.selectors,
		NumEndpoints:   s.numEndpoints(),
		NumSubscribers: s.numSubscribers(),
		Stateful:       s.stateful,
		Routes:         c.routingList.getRouteIds(s.name),
		Groups:         c.groupList.getGroupNames(s.name),
	}
	return
}

func (api *oamApi) GetEndpointInfos() (infos []oambackend.EndpointInfo) {
	epman := api.ctrl.epman
	endpoints := epman.getAllEndpoints()
	for _, ep := range endpoints {
		infos = append(infos, ep.toEndpointInfo())
	}
	return
}

func (api *oamApi) GetGatewayInfos() (infos []oambackend.GatewayInfo) {
	gwman := api.ctrl.gwman
	gateways := gwman.getGateways()
	for _, gw := range gateways {
		infos = append(infos, gw.toGatewayInfo())
	}
	return
}

func (api *oamApi) GetServiceInfos() (infos []oambackend.ServiceInfo) {
	sman := api.ctrl.sman
	services := sman.listAll()
	for _, s := range services {
		infos = append(infos, s.toServiceInfo(api.ctrl))
	}
	return
}

func (r *Route) toRouteInfo() (info oambackend.RouteInfo) {
	info = oambackend.RouteInfo{
		Id:      int(r.id),
		IdStr:   r.idStr,
		Service: r.service,
		Match:   r.match,
	}
	for _, d := range r.targets {
		info.Destinations = append(info.Destinations, oambackend.DestinationInfo{
			GroupId:      d.groupId,
			Weight:       int(d.weight),
			LoadBalancer: d.lb.String(),
		})
	}
	return
}

func (api *oamApi) GetRouteInfos() (infos []oambackend.RouteInfo) {
	for _, s := range api.ctrl.routingList.listRoutes() {
		infos = append(infos, s.toRouteInfo())
	}
	return
}

func (sg *ServiceGroup) toGroupInfo() oambackend.GroupInfo {
	return oambackend.GroupInfo{
		Id:        int(sg.id),
		IdStr:     sg.idStr,
		Name:      sg.name,
		Service:   sg.service,
		Selectors: sg.selectors,
	}
}

func (api *oamApi) GetGroupInfos() (infos []oambackend.GroupInfo) {
	for _, g := range api.ctrl.groupList.listGroups() {
		infos = append(infos, g.toGroupInfo())
	}
	return
}
