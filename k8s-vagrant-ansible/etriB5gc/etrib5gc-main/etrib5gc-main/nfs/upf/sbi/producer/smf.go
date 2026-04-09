package producer

import (
	"etrib5gc/nfs/upf/sessions"
	"etrib5gc/nfs/upf/smfman"
	"fmt"
	"github.com/reogac/pfcp/message"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (prod *Producer) HandleAssociationRequest(callback *models.EndpointInfo, body *message.PFCPAssociationSetupRequest) (rsp *message.PFCPAssociationSetupResponse, prob *models.ProblemDetails) {
	prod.Infof("Receive AssociationRequest from SMF")
	var err error
	if rsp, err = smfman.RegisterSmf(*callback, body); err != nil {
		prod.Errorf("Fail to register SMF: %+v", err)
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Smf not registered",
		}
	}

	return
}

func (prod *Producer) HandleDisassociationRequest(smfId string) (prob *models.ProblemDetails) {
	prod.Infof("Receive DisassociationRequest from SMF: %s", smfId)
	var err error
	if err = smfman.UnregisterSmf(smfId); err != nil {
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: "Smf not found",
		}
	}
	return
}

func (prod *Producer) HandleSessionEstablishment(smfId string, req *message.PFCPSessionEstablishmentRequest) (rsp *message.PFCPSessionEstablishmentResponse, prob *models.ProblemDetails) {
	prod.Infof("Receive SessionEstablishmentRequest from SMF")
	var smf *smfman.Smf
	if smf = smfman.FindSmf(smfId); smf == nil {
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "not found SMF",
		}
		return
	}

	var err error
	if req.CPFSEID.Seid() == 0 {
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Get F-SEID failed"),
		}
		return
	}

	rSeid := req.CPFSEID.Seid()
	var s *sessions.PfcpSession
	//create a new session (will fail if session existed)
	s = sessions.CreateSession(smf.SbiClient(), rSeid)

	//handle the session establishment
	if rsp, err = s.HandleSessionEstablishment(req); err != nil {
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Handle session establishment failed: %+v", err),
		}
	}

	return
}

func (prod *Producer) HandleSessionModification(seid int64, req *message.PFCPSessionModificationRequest) (rsp *message.PFCPSessionModificationResponse, prob *models.ProblemDetails) {
	prod.Infof("Receive SessionModificationRequest from SMF")
	if s := sessions.FindSession(uint64(seid)); s == nil {
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Session not found",
		}
		return
	} else {
		var err error
		if rsp, err = s.HandleSessionModification(req); err != nil {
			prob = &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: fmt.Sprintf("Handle session modification failed: %+v", err),
			}
		}
	}
	return
}

func (prod *Producer) HandleSessionDeletion(seid int64, req *message.PFCPSessionDeletionRequest) (rsp *message.PFCPSessionDeletionResponse, prob *models.ProblemDetails) {
	prod.Infof("Receive SessionDeletionRequest from SMF - seid %d", seid)
	if s := sessions.FindSession(uint64(seid)); s == nil {
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Session not found",
		}
		return
	} else {
		var err error
		if rsp, err = s.HandleSessionDeletion(req); err != nil {
			prob = &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: fmt.Sprintf("Handle session modification failed: %+v", err),
			}
		} else {
			sessions.DeleteSession(uint64(seid))
		}
	}
	return
}
