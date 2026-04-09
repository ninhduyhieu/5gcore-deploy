package producer

import (
	"context"
	"etrib5gc/nfs/amf/uecontext"
	"github.com/reogac/sbi/apis/amf/comm"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *producerImpl) HandleCancelRelocateUEContext(ueContextId string, body *models.CancelRelocateUEContextRequest) (prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleAMFStatusChangeUnSubscribe(subscriptionId string) (prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleAMFStatusChangeSubscribeModfy(subscriptionId string, body *models.SubscriptionData) (rsp *models.SubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleEBIAssignment(ueContextId string, body *models.AssignEbiData) (rsp *models.AssignedEbiData, ersp *models.AssignEbiError, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleUEContextTransfer(ueContextId string, body *models.UEContextTransferRequest) (rsp *models.UEContextTransferResponse, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleRelocateUEContext(ueContextId string, body *models.RelocateUEContextRequest) (headers map[string]string, rsp *models.UeContextRelocatedData, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleN1N2MessageUnSubscribe(params *comm.N1N2MessageUnSubscribeParams) (prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleNonUeN2MessageTransfer(body *models.NonUeN2MessageTransferRequest) (rsp *models.N2InformationTransferRspData, ersp *models.N2InformationTransferError, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleAMFStatusChangeSubscribe(body *models.SubscriptionData) (headers map[string]string, rsp *models.SubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleReleaseUEContext(ueContextId string, body *models.UEContextRelease) (prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleRegistrationStatusUpdate(ueContextId string, body *models.UeRegStatusUpdateReqData) (rsp *models.UeRegStatusUpdateRspData, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleNonUeN2InfoUnSubscribe(n2NotifySubscriptionId string) (prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleCreateUEContext(ueContextId string, body *models.CreateUEContextRequest) (headers map[string]string, rsp *models.CreateUEContextResponse, ersp *models.UeContextCreateError, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleN1N2MessageSubscribe(ueContextId string, body *models.UeN1N2InfoSubscriptionCreateData) (headers map[string]string, rsp *models.UeN1N2InfoSubscriptionCreatedData, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleNonUeN2InfoSubscribe(body *models.NonUeN2InfoSubscriptionCreateData) (headers map[string]string, rsp *models.NonUeN2InfoSubscriptionCreatedData, prob *models.ProblemDetails) {
	return
}

func (p *producerImpl) HandleN1N2MessageTransfer(ueContextId string, body *models.N1N2MessageTransferRequest) (rsp *models.N1N2MessageTransferRspData, ersp *models.N1N2MessageTransferError, prob *models.ProblemDetails) {
	//1. extract message, (get target access type)
	var ueCtx *uecontext.UeContext
	if ueCtx = uecontext.FindUeBySupi(ueContextId); ueCtx == nil {
		p.Errorf("UeContext not found [supi = %s] to handle N1N2Transfer", ueContextId)
		ersp = &models.N1N2MessageTransferError{
			Error: models.ProblemDetails{
				Status: http.StatusNotFound,
				Detail: "Ue Context not found",
			},
		}
		return
	}
	rsp, ersp = ueCtx.ReceiveN1N2(context.TODO(), body)
	return
}
