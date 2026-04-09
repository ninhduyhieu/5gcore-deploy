package producer

import (
	"context"
	"etrib5gc/nfs/amf/ranue"
	"github.com/reogac/sbi/apis/amf/handover"
	"github.com/reogac/sbi/models"
	"net/http"
)

var internalProblem *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusInternalServerError,
	Detail: "Fail to handle request",
}
var notFoundProblem *models.ProblemDetails = &models.ProblemDetails{
	Status: http.StatusNotFound,
	Detail: "UeContext not found",
}

func (p *producerImpl) HandleHandoverRequired(ueId int64, msg *models.HandoverRequired) (*models.HandoverCommand, *models.HandoverPreparationFailure, *models.ProblemDetails) {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle HandoverRequire", ueId)
		return nil, nil, notFoundProblem
	} else if rsp, ersp, err := ranUe.ReceiveHandoverRequired(context.TODO(), msg); err != nil {
		p.Errorf("Fail to handle HandoverRequired: %+v", err)
		return nil, nil, internalProblem
	} else {
		return rsp, ersp, nil
	}
}

func (p *producerImpl) HandleHandoverNotify(ueId int64, msg *models.HandoverNotify) *models.ProblemDetails {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle HandoverNotify", ueId)
		return notFoundProblem
	} else {
		if err := ranUe.ReceiveHandoverNotify(context.TODO(), msg); err != nil {
			p.Errorf("Fail to handle HandoverNotify: %+v", err)
			return internalProblem
		}
	}

	return nil
}

func (p *producerImpl) HandleHandoverCancel(ueId int64, msg *models.HandoverCancel) (*models.HandoverCancelAcknowledge, *models.ProblemDetails) {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle HandoverCancel", ueId)
		return nil, notFoundProblem
	} else {
		if rsp, err := ranUe.ReceiveHandoverCancel(context.TODO(), msg); err != nil {
			p.Errorf("Fail to handle HandoverCancel: %+v", err)
			return nil, internalProblem
		} else {
			return rsp, nil
		}
	}
}

func (p *producerImpl) HandlePathSwitch(params *handover.PathSwitchParams, msg *models.PathSwitchRequest) (rsp *models.PathSwitchAcknowledge, ersp *models.PathSwitchFailure, prob *models.ProblemDetails) {
	if ranUe := ranue.FindRanUe(params.UeId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle PathSwitchRequest", params.UeId)
		return nil, nil, notFoundProblem
	} else if rsp, ersp, err := ranUe.ReceivePathswitch(context.TODO(), params.Callback, msg); err != nil {
		return nil, nil, internalProblem
	} else {
		return rsp, ersp, nil
	}
}
