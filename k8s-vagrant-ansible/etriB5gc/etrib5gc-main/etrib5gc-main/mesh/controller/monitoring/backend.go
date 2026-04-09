package monitoring

import (
	"etrib5gc/mesh/controller/oambackend"
	"github.com/gin-gonic/gin"
	"github.com/reogac/utils/httpw"
	"net/http"
)

type MonitoringApi interface {
	GetEndpointInfos() []oambackend.EndpointInfo
	GetGatewayInfos() []oambackend.GatewayInfo
	GetServiceInfos() []oambackend.ServiceInfo
	GetRouteInfos() []oambackend.RouteInfo
	GetGroupInfos() []oambackend.GroupInfo
}

type backend struct {
	api MonitoringApi
}

func New(api MonitoringApi) *backend {
	return &backend{api: api}
}

func (b *backend) MakeHttpRoutes() []httpw.Route {
	return []httpw.Route{
		httpw.Route{ //endpoint registers
			Method:  http.MethodGet,
			Pattern: "/gateways",
			Handler: b.onGetGatewayInfos,
		},
		httpw.Route{ //endpoint deregisters
			Method:  http.MethodGet,
			Pattern: "/endpoints",
			Handler: b.onGetEndpointInfos,
		},
		httpw.Route{ //endpoint subscribes
			Method:  http.MethodGet,
			Pattern: "/services",
			Handler: b.onGetServiceInfos,
		},
		httpw.Route{ //endpoint activates
			Method:  http.MethodGet,
			Pattern: "/routes",
			Handler: b.onGetRouteInfos,
		},
		httpw.Route{ //controller sends updates to endpoint
			Method:  http.MethodGet,
			Pattern: "/groups",
			Handler: b.onGetGroupInfos,
		},
	}
}

func (b *backend) onGetGatewayInfos(ctx *gin.Context) {
	infos := b.api.GetGatewayInfos()
	ctx.JSON(http.StatusOK, &infos)
}

func (b *backend) onGetEndpointInfos(ctx *gin.Context) {
	infos := b.api.GetEndpointInfos()
	ctx.JSON(http.StatusOK, &infos)
}

func (b *backend) onGetGroupInfos(ctx *gin.Context) {
	infos := b.api.GetGroupInfos()
	ctx.JSON(http.StatusOK, &infos)
}

func (b *backend) onGetServiceInfos(ctx *gin.Context) {
	infos := b.api.GetServiceInfos()
	ctx.JSON(http.StatusOK, &infos)
}
func (b *backend) onGetRouteInfos(ctx *gin.Context) {
	infos := b.api.GetRouteInfos()
	ctx.JSON(http.StatusOK, &infos)
}
