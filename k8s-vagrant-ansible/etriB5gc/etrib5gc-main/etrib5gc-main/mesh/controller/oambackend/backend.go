package oambackend

import (
	"github.com/reogac/utils/oam"
)

const (
	CTRL_CTX_ID string = "6gctrl"
)

type OamApi interface {
	GetEndpointInfos() []EndpointInfo
	GetGatewayInfos() []GatewayInfo
	GetServiceInfos() []ServiceInfo
	GetRouteInfos() []RouteInfo
	GetGroupInfos() []GroupInfo
}

func NewOamHandler(api OamApi) *oam.OamHandler {
	name := "6G-Controller"
	rootId := "6gctrl"

	getter := func(ctxId string) *oam.HandlerContext {
		if ctxId == CTRL_CTX_ID {
			return oam.NewHandlerContext(ctxId, &CtrlHandler{
				api: api,
			}, cmds, nil)
		}
		return nil
	}
	return oam.NewOamHandler(name, rootId, getter)
}
