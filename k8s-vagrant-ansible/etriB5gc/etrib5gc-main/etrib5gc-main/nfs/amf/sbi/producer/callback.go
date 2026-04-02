package producer

import (
	"etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/uecontext"
	"fmt"
	"github.com/reogac/sbi/apis/amf/callback"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *producerImpl) HandleSmContextStatusNotify(params *callback.SmContextStatusNotifyParams, body *models.SmContextStatusNotification) (prob *models.ProblemDetails) {
	var err error
	if ueCtx := uecontext.FindUeBySupi(params.Supi); ueCtx == nil {
		p.Errorf("UeContxt not found [supi = %s] to handle SmContextStatusNotify", params.Supi)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UeContext not found for SUPI=%s", params.Supi),
		}
	} else {
		ueCtx.Tracef("Receive SmContextStatusNotify from SMF")
		if err = ueCtx.HandleSmContextStatusNotify(uint8(params.SessionId), body); err != nil {
			p.Errorf("Fail to handle SmContextStatusNotify: %+v", err)
			prob = &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: fmt.Sprintf("Fail to handle request"),
			}
		}
	}
	return
}

func (p *producerImpl) HandleRanInfoUpdate(ranId string, body *models.RanInfoUpdateData) (prob *models.ProblemDetails) {
	context.UpdateRanInfo(ranId, body)
	return
}
