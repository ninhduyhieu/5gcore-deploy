package producer

import (
	"net/http"
	//"github.com/reogac/sbi/apis/pran/subs"
	"etrib5gc/nfs/pran/context"
	"github.com/reogac/sbi/models"
)

func (p *producerImpl) HandleAmfSubscribe(callback *models.EndpointInfo, body *models.AmfSubscribeRequest) (rsp *models.AmfSubscribeResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive AMF subsribe request")
	if err := context.SubscribeAmf(callback, body.Id); err != nil {
		p.Errorf("Fail to subcribe AMF: %+v", err)
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Fail to subscribe AMF",
		}
	} else {
		rsp = &models.AmfSubscribeResponse{
			Id:              body.Id,
			PlmnId:          *context.PlmnId(),
			SupportedTAList: context.GetSupportedTAList(),
		}
	}
	return
}

func (p *producerImpl) HandleSendPaging(body *models.PagingMessage) (prob *models.ProblemDetails) {
	p.Warnf("Receive Paging message from AMF, not handled yet")
	return
}
