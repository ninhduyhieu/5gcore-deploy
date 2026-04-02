package registry

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServiceInfo struct {
	Id        string
	Selectors map[string]string
	Stateful  bool
	Endpoints []EndpointInfo
}

type EndpointInfo struct {
	Id       string
	Addr     string
	Labels   map[string]string
	Services []string
}

func (sman *ServiceManager) writeServiceInfos() (infos []ServiceInfo) {
	sman.mutex.RLock()
	defer sman.mutex.RUnlock()
	for _, s := range sman.services {
		infos = append(infos, s.toServiceInfo())
	}
	return
}
func (s *Service) toServiceInfo() (info ServiceInfo) {
	info = ServiceInfo{
		Id:        s.id,
		Selectors: s.selectors,
		Stateful:  s.stateful,
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, ep := range s.endpoints {
		info.Endpoints = append(info.Endpoints, ep.toEndpointInfo())
	}
	return
}

func (reg *Registry) onGetServices(ctx *gin.Context) {
	services := reg.sman.writeServiceInfos()
	ctx.JSON(http.StatusOK, services)
}
func (ep *Endpoint) toEndpointInfo() (info EndpointInfo) {
	info = EndpointInfo{
		Id:     ep.id,
		Addr:   ep.sbiUrl,
		Labels: ep.labels,
	}
	for s, _ := range ep.services {
		info.Services = append(info.Services, s)
	}
	return
}

func (reg *Registry) onGetEndpoints(ctx *gin.Context) {
	endpoints := []EndpointInfo{}
	for _, ep := range reg.epman.listEndpoints() {
		endpoints = append(endpoints, ep.toEndpointInfo())
	}
	ctx.JSON(http.StatusOK, endpoints)
}
