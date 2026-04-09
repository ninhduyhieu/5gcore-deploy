package backend

import (
	"etrib5gc/nfs/amf/uecontext"
	"github.com/reogac/utils/oam"
	"strconv"
	"strings"
)

const (
	AMF_CTX_ID       string = "amf"
	UE_CTX_ID_PREFIX string = "ue"
)

func CreateOamHandler(ext *oam.HandlerContext) *oam.OamHandler {
	getter := func(contextId string) *oam.HandlerContext {
		if contextId == AMF_CTX_ID { //AMF context
			return oam.NewHandlerContext(contextId, new(OamAmfHandler), amfCmds, ext) //NOTE: add extention for root context
		} else { //Ue context [ue:tmsi]
			parts := strings.Split(contextId, ":")
			if len(parts) == 2 && parts[0] == UE_CTX_ID_PREFIX {
				//get TMSI value:
				if tmsi, err := strconv.Atoi(parts[1]); err == nil {
					//find UeContext
					if ueCtx := uecontext.FindUeByTmsi(uint32(tmsi)); ueCtx != nil {
						return oam.NewHandlerContext(contextId, &OamUeHandler{
							ueCtx: ueCtx,
						}, ueCmds, nil)
					}
				}
			}
		}
		return nil
	}

	return oam.NewOamHandler("AMF", AMF_CTX_ID, getter)
}
