package backend

import (
	"github.com/reogac/utils/oam"
)

const (
	SMF_CTX_ID string = "smf"
)

func CreateOamHandler(ext *oam.HandlerContext) *oam.OamHandler {
	getter := func(contextId string) *oam.HandlerContext {
		if contextId == SMF_CTX_ID { //SMF context
			return oam.NewHandlerContext(contextId, new(OamSmfHandler), smfCmds, ext)
		} else { //Pdu sessions context [session:id]
			//TODO: creat a handler for Pdu Sesion Context
		}
		return nil
	}

	return oam.NewOamHandler("SMF", SMF_CTX_ID, getter)
}
