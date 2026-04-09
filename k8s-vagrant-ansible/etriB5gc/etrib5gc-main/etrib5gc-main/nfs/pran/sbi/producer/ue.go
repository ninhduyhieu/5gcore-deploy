package producer

import (
	"etrib5gc/nfs/pran/ue"
	"fmt"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *producerImpl) HandleUeContextRelease(ueId int64, msg *models.UeContextReleaseCommand) (rsp *models.UeContextReleaseComplete, prob *models.ProblemDetails) {
	p.Debugf("Receive UeContextReleaseCommand [cuUeId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [cuUeId= %d] to handle UeContextReleaseCommand", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
		return
	} else {
		ctxInfo := ue.CreateUeCtxReleaseContext(msg)
		if err := ue.ReceiveAsyncSbiEvent(ueCtx, ue.UECTX_RELEASE, ctxInfo); err == nil {
			rsp = ctxInfo.Complete
		} else {
			ueCtx.Errorf("Fail to handle UeContextRelease command: %+v", err)
			prob = internalProblem
		}
	}
	return
}

func (p *producerImpl) HandleUeContextSetup(ueId int64, msg *models.UeContextSetupRequest) (rsp *models.UeContextSetupResponse, ersp *models.UeContextSetupFailure, prob *models.ProblemDetails) {
	p.Debugf("Receive UeContextSetup [cuUeId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [cuUeId= %d] to handle InitialUeContextSetupRequest", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
	} else {
		ctxInfo := ue.CreateUeCtxSetupContext(msg)
		if err := ue.ReceiveAsyncSbiEvent[ue.UeCtxSetupContext](ueCtx, ue.UECTX_SETUP, ctxInfo); err == nil {
			rsp, ersp = ctxInfo.Response, ctxInfo.Failure
		} else {
			ueCtx.Errorf("Fail to handle UeContextSetup request: %+v", err)
			prob = internalProblem
		}
	}

	return
}

func (p *producerImpl) HandleUpdateAmfUeContextInfo(ueId int64, msg *models.AmfUeContextInfo) (prob *models.ProblemDetails) {
	p.Debugf("Receive UpdateAmfUeContextInfo [cuUeId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [cuUeId= %d] to update AMF info", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}

	} else {
		ue.ReceiveSbiEvent(ueCtx, ue.UPDATE_AMF_INFO, msg)
	}
	return
}

func (p *producerImpl) HandleUeContextModify(ueId int64, msg *models.UeContextModifyRequest) (rsp *models.UeContextModifyResponse, ersp *models.UeContextModifyFailure, prob *models.ProblemDetails) {
	p.Debugf("Receive UeContextModify [cuUeId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [cuUeId= %d] to handle UeContextModificationRequest", ueId)
		ersp = &models.UeContextModifyFailure{
			Error: models.ProblemDetails{
				Status: http.StatusNotFound,
				Detail: fmt.Sprintf("UE context not found for %d", ueId),
			},
		}
		return
	} else {
		ctxInfo := ue.CreateUeCtxModifyContext(msg)
		if err := ue.ReceiveAsyncSbiEvent(ueCtx, ue.UECTX_MODIFY, ctxInfo); err == nil {
			rsp, ersp = ctxInfo.Response, ctxInfo.Failure
		} else {
			ueCtx.Errorf("Fail to handle UeContextModify request: %+v", err)
			ersp = &models.UeContextModifyFailure{
				Error: *internalProblem,
			}
		}
	}
	return
}
