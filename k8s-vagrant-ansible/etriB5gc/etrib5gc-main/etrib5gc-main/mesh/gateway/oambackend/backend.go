package oambackend

import (
	"github.com/reogac/utils/oam"
)

const (
	GW_CTX_ID string = "6ggw"
)

type OamApi interface {
	GetEndpointInfos() []EndpointInfo
}

func NewOamHandler(api OamApi) *oam.OamHandler {
	name := "6G-Gateway"
	rootId := "6ggw"

	getter := func(ctxId string) *oam.HandlerContext {
		if ctxId == GW_CTX_ID {
			return oam.NewHandlerContext(ctxId, &GwHandler{
				api: api,
			}, cmds, nil)
		}
		return nil
	}
	return oam.NewOamHandler(name, rootId, getter)
}
