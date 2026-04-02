package ue

import (
	"etrib5gc/internal/eventmux"
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/smf/n1n2ul"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"sync"
)

type SmContext struct {
	*logrus.Entry
	ueCtx      *UeContext
	id         uint8  //PduSessionId
	ref        string //SmContext id at SMF
	smfCli     sbi.ConsumerClient
	modifyJob  *SessionResourceModifyContext  //pending resource modification request
	setupJob   *SessionResourceSetupContext   //pending resource setup request
	releaseJob *SessionResourceReleaseContext //pending resource release request
	mutex      sync.Mutex
}

func (smCtx *SmContext) clean() {
	smCtx.mutex.Lock()
	defer smCtx.mutex.Unlock()

	if smCtx.setupJob != nil {
		smCtx.setupJob.Finalize(fmt.Errorf("UeContext is released"))
	}

	if smCtx.releaseJob != nil {
		smCtx.releaseJob.Finalize(fmt.Errorf("UeContext is released"))
	}
	if smCtx.modifyJob != nil {
		smCtx.modifyJob.Finalize(fmt.Errorf("UeContext is released"))
	}
}

func (smCtx *SmContext) receiveSetupResponse(transfer []byte, success bool) {
	smCtx.mutex.Lock()
	defer smCtx.mutex.Unlock()

	if smCtx.setupJob == nil {
		smCtx.Warnf("Receive an unsolicited PduSessionResourceSetupResponse")
		return
	}
	ctx := smCtx.setupJob
	ctx.Response = &models.SessionResourceSetupResponse{
		Success:  success,
		Transfer: transfer,
	}
	smCtx.Infof("PduSessionResourceSetupResponse from gnB is forwarded to SMF")
	ctx.Finalize(nil)
}

func (smCtx *SmContext) receiveModifyResponse(transfer []byte, success bool) {
	smCtx.mutex.Lock()
	defer smCtx.mutex.Unlock()

	if smCtx.modifyJob == nil {
		smCtx.Warnf("Receive an unsolicited PduSessionResourceModifyResponse")
		return
	}
	ctx := smCtx.modifyJob
	ctx.Response = &models.SessionResourceModifyResponse{
		Success:  success,
		Transfer: transfer,
	}

	smCtx.Infof("PduSessionResourceModifyResponse from gnB is forwarded to SMF")
	ctx.Finalize(nil)
}

func (smCtx *SmContext) receiveReleaseResponse(transfer []byte) {
	smCtx.mutex.Lock()
	defer smCtx.mutex.Unlock()

	if smCtx.releaseJob == nil {
		smCtx.Warnf("Receive an unsolicited PduSessionResourceReleaseResponse")
		return
	}
	ctx := smCtx.releaseJob
	ctx.Response = &models.SessionResourceReleaseResponse{
		Transfer: transfer,
	}

	smCtx.Infof("PduSessionResourceReleaseResponse from gnB is forwarded to SMF")
	ctx.Finalize(nil)
	smCtx.ueCtx.removeSmContext(smCtx.id)
}

func (smCtx *SmContext) receiveNotify(transfer []byte, release bool) {
	req := &models.SessionResourceNotification{
		Transfer: transfer,
		Release:  release,
	}
	if err := n1n2ul.SessionResourceNotify(smCtx.smfCli, smCtx.ref, req); err != nil {
		smCtx.Errorf("Fail to forward SessionResourceNotify from gnB to SMF: %+v", err)
	} else {
		smCtx.Info("SessionResourceNotify from gnB is forwarded to SMF: %+v", err)
	}
}

func (smCtx *SmContext) receiveModifyIndication(transfer []byte) {
	req := &models.SessionResourceModifyIndication{
		Transfer: transfer,
	}
	if rsp, err := n1n2ul.SessionResourceModifyIndication(smCtx.smfCli, smCtx.ref, req); err != nil {
		smCtx.Errorf("Fail to forward SessionResourceModifyIndication from gnB to SMF: %+v", err)
	} else {
		smCtx.Info("SessionResourceModificationIndication from gnB is forwarded to SMF")
		if err = smCtx.ueCtx.sendPduSessionResourceModifyConfirm(smCtx.id, rsp.Transfer, rsp.Success); err != nil {
			smCtx.Errorf("Fail to forward SessionResourceModifyConfirm from SMF to gNB: %+v", err)
		} else {
			smCtx.Info("SessionResourceModifyConfirm from SMF is forwarded to gnB")
		}
	}
}

type SessionResourceSetupContext struct {
	Request  *models.SessionResourceSetupRequest
	Response *models.SessionResourceSetupResponse
	*eventmux.AsyncTask
}

func CreateSessionResourceSetupContext(req *models.SessionResourceSetupRequest) *SessionResourceSetupContext {
	return &SessionResourceSetupContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

type SessionResourceModifyContext struct {
	Request  *models.SessionResourceModifyRequest
	Response *models.SessionResourceModifyResponse
	*eventmux.AsyncTask
}

func CreateSessionResourceModifyContext(req *models.SessionResourceModifyRequest) *SessionResourceModifyContext {
	return &SessionResourceModifyContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

type SessionResourceReleaseContext struct {
	Request  *models.SessionResourceReleaseRequest
	Response *models.SessionResourceReleaseResponse
	*eventmux.AsyncTask
}

func CreateSessionResourceReleaseContext(req *models.SessionResourceReleaseRequest) *SessionResourceReleaseContext {
	return &SessionResourceReleaseContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

func (ueCtx *UeContext) findSmContext(id uint8) *SmContext {
	if id < 1 || id > MAX_PDU_SESSIONS {
		return nil
	}
	return ueCtx.sessions[id-1]
}
func (ueCtx *UeContext) addSmContext(smCtx *SmContext) {
	ueCtx.sessions[smCtx.id-1] = smCtx
}

func (ueCtx *UeContext) removeSmContext(id uint8) {
	if id > 0 && id <= MAX_PDU_SESSIONS {
		ueCtx.sessions[id-1] = nil
	}
}
func (ueCtx *UeContext) createSessionResource(info *SessionResourceSetupContext) {
	var err error
	msg := info.Request
	sessionId := uint8(msg.SessionId)
	if ueCtx.findSmContext(sessionId) != nil {
		info.Finalize(fmt.Errorf("Fail to create session context, session existed"))
		return
	}
	smCtx := &SmContext{
		ueCtx: ueCtx,
		id:    sessionId,
		ref:   msg.Ref,
		Entry: ueCtx.WithFields(logrus.Fields{
			"ref": msg.Ref,
		}),
	}
	if smCtx.smfCli, err = mesh.ConsumerFromEndpoint(&msg.Smf); err != nil {
		info.Finalize(utils.WrapError("Create SMF consumer", err))
		return
	}
	if err = ueCtx.sendPduSessionResourceSetupRequest(sessionId, msg.Transfer, msg.N1Sm, &msg.Snssai); err == nil {
		ueCtx.addSmContext(smCtx)
		smCtx.Infof("PduSessionResourceSetupRequest from SMF is forwwarded to gnB")
		smCtx.setupJob = info
		info.SetFinalizer(func(old func(error)) func(error) {
			return func(err error) {
				smCtx.setupJob = nil
				old(err)
			}
		})
	} else {
		info.Finalize(utils.WrapError("Forward PduSessionSetupRequest from SMF to gnB", err))
	}
	return
}

func (ueCtx *UeContext) modifySessionResource(info *SessionResourceModifyContext) {
	var err error
	msg := info.Request
	sessionId := uint8(msg.SessionId)
	if smCtx := ueCtx.findSmContext(sessionId); smCtx == nil {
		info.Finalize(fmt.Errorf("Session %d not found", sessionId))
	} else if smCtx.modifyJob != nil {
		info.Finalize(fmt.Errorf("Modification on going"))
	} else {
		if err = ueCtx.sendPduSessionResourceModifyRequest(msg); err == nil {
			smCtx.Infof("PduSessionResourceModifyRequest from SMF is forwwarded to gnB")
			smCtx.modifyJob = info
			info.SetFinalizer(func(old func(error)) func(error) {
				return func(err error) {
					smCtx.modifyJob = nil
					old(err)
				}
			})
		} else {
			info.Finalize(utils.WrapError("Forward PduSessionModifyRequest from SMF  to gnB", err))
		}
	}
	return
}

func (ueCtx *UeContext) releaseSessionResource(info *SessionResourceReleaseContext) {
	var err error
	msg := info.Request
	sessionId := uint8(msg.SessionId)
	if smCtx := ueCtx.findSmContext(sessionId); smCtx == nil {
		info.Finalize(fmt.Errorf("Session %d not found", sessionId))
	} else if smCtx.releaseJob != nil {
		info.Finalize(fmt.Errorf("Release on going"))
	} else {
		if err = ueCtx.sendPduSessionResourceReleaseCommand(sessionId, msg.Transfer, msg.N1Sm); err == nil {
			smCtx.Infof("PduSessionResourceReleaseCommand from SMF is forwwarded to gnB")
			smCtx.releaseJob = info
			info.SetFinalizer(func(old func(error)) func(error) {
				return func(err error) {
					smCtx.releaseJob = nil
					old(err)
				}
			})

		} else {
			info.Finalize(utils.WrapError("Forward PduSessionReleaseCommand from SMF to gnB", err))
		}
	}
	return
}
