package producer

import (
	"etrib5gc/nfs/pran/ran"
	"etrib5gc/nfs/pran/ue"
	"fmt"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *producerImpl) HandleHandoverRequest(callback *models.EndpointInfo, msg *models.HandoverRequest) (rsp *models.HandoverRequestAcknowledge, errRsp *models.HandoverRequestFailure, prob *models.ProblemDetails) {
	p.Debugf("Receive HandoverRequest")
	//look for RAN:
	var gnb *ran.Ran
	ranId := new(models.GlobalRanNodeId) //TODO: from request
	if gnb := ran.FindRanWithId(ranId); gnb == nil {
		prob = &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Gnb not found"),
		}
		return
	}

	ctxInfo := &ue.HandoverRequestContext{
		Request: msg,
	}

	if err := ue.HandleHandoverRequest(callback, gnb, ctxInfo); err != nil {
		p.Errorf("Fail to create UeContext: %+v", err)
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Fail to handle HandoverRequest"),
		}
	} else {
		rsp, errRsp = ctxInfo.Response, ctxInfo.ErrResponse
	}
	return
}
