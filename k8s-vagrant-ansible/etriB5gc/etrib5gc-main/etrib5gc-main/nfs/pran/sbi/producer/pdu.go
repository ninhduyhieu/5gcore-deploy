package producer

import (
	"etrib5gc/nfs/pran/ue"
	"fmt"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *producerImpl) HandleN2SmInfoDownlink(ueId int64, msg *models.N2SmInfoDownlinkTransport) (prob *models.ProblemDetails) {
	p.Debugf("Receive N2SmInfoDownlink [ueId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [id= %d] to handle N2SmInfoDownlink request", ueId)
		return &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
	} else {
		ue.ReceiveSbiEvent(ueCtx, ue.N2SMINFO_DOWNLINK, msg)
	}

	return nil
}

func (p *producerImpl) HandleSessionResourceSetup(ueId int64, msg *models.SessionResourceSetupRequest) (rsp *models.SessionResourceSetupResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive SessionResourceSetup [ueId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [id= %d] to handle session resource setup request", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
	} else {
		ctxInfo := ue.CreateSessionResourceSetupContext(msg)
		if err := ue.ReceiveAsyncSbiEvent(ueCtx, ue.PDU_SETUP, ctxInfo); err != nil {
			ueCtx.Errorf("Fail to handle PduSessionResourceSetup request: %+v", err)
			prob = internalProblem
		} else {
			rsp = ctxInfo.Response
		}
	}
	return
}

func (p *producerImpl) HandleSessionResourceModify(ueId int64, msg *models.SessionResourceModifyRequest) (rsp *models.SessionResourceModifyResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive SessionResourceModify [ueId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [id= %d] to handle session resource modify request", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
	} else {
		ctxInfo := ue.CreateSessionResourceModifyContext(msg)
		if err := ue.ReceiveAsyncSbiEvent(ueCtx, ue.PDU_MODIFY, ctxInfo); err != nil {
			ueCtx.Errorf("Fail to handle PduSessionResourceModify request: %+v", err)
			prob = internalProblem
		} else {
			rsp = ctxInfo.Response
		}
	}
	return
}

func (p *producerImpl) HandleSessionResourceRelease(ueId int64, msg *models.SessionResourceReleaseRequest) (rsp *models.SessionResourceReleaseResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive SessionResourceRelease [ueId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [id= %d] to handle session resource release request", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
	} else {
		ctxInfo := ue.CreateSessionResourceReleaseContext(msg)
		if err := ue.ReceiveAsyncSbiEvent(ueCtx, ue.PDU_RELEASE, ctxInfo); err != nil {
			ueCtx.Errorf("Fail to handle PduSessionResourceRelease request: %+v", err)
			prob = internalProblem
		} else {
			rsp = ctxInfo.Response
		}
	}
	return
}
