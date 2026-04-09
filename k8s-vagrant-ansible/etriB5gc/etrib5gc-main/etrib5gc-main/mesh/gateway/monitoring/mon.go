package monitoring

import (
	"etrib5gc/mesh/gateway/oambackend"
	"github.com/reogac/utils/httpw"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MonitoringApi interface {
	GetEndpointInfos() []oambackend.EndpointInfo
}

type backend struct {
	api MonitoringApi
}

func New(api MonitoringApi) *backend {
	return &backend{
		api: api,
	}
}

func (b *backend) onGetEndpointInfos(ctx *gin.Context) {
	infos := b.api.GetEndpointInfos()
	ctx.JSON(http.StatusOK, &infos)
}
func (b *backend) MakeHttpRoutes() []httpw.Route {
	return []httpw.Route{
		httpw.Route{ //monitoring: get all endponts
			Method:  http.MethodGet,
			Pattern: "/endpoints",
			Handler: b.onGetEndpointInfos,
		},
	}
}
