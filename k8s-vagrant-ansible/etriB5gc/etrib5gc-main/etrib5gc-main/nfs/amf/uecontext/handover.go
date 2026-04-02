package uecontext

import (
	"etrib5gc/common"
	amfctx "etrib5gc/nfs/amf/context"
	"fmt"
	"github.com/reogac/sbi/models"
	"time"
)

type HandoverContext struct {
	source, target   RanUe
	switchedSessions map[uint8]bool
	ncc              uint8
	nh               []byte
	timer            common.Timer
}

func (ueCtx *UeContext) HandleHandoverRequired(isGpp bool, target interface{}, req *models.HandoverRequired) (*models.HandoverCommand, *models.HandoverPreparationFailure, error) {
	ueCtx.hoMu.Lock()
	defer ueCtx.hoMu.Unlock()

	//1. check source RanUe status
	if !ueCtx.isRegistered(isGpp) {
		return nil, nil, fmt.Errorf("Ue not registered")
	}

	if !ueCtx.HasValidSecmode() {
		return nil, nil, fmt.Errorf("Ue has no security context")
	}

	source := ueCtx.getRanUe(isGpp)

	//2. get target RanUe
	targetRanUe, _ := target.(RanUe)

	//3. start handover procedure
	hoCtx := &HandoverContext{
		source:           source,
		target:           targetRanUe,
		switchedSessions: make(map[uint8]bool),
		ncc:              ueCtx.currentSecCtx.Ncc(),
		nh:               ueCtx.currentSecCtx.Nh(),
		timer:            common.NewTimer(time.Duration(amfctx.HandoverTimeout())*time.Millisecond, ueCtx.onHandoverTimeout, nil),
	}

	ueCtx.hoCtx = hoCtx

	//1. ask SMFs for session resource preparation
	n2DlList := ueCtx.smfHandoverPreparing(req.Sessions)

	//2. ask target gnB to get prepared

	newSecInd := false //no new security context (same AMF N2 handover)
	request := &models.HandoverRequest{
		SourceToTargetContent: req.SourceToTargetContent,
		Sessions:              n2DlList,
		HandoverType:          req.HandoverType,
		Cause:                 req.Cause,
		UeAmbr:                ueCtx.getAmbr(),
		UeSecurityCapability:  ueCtx.getUeSecurityCapability(),
		Guami:                 amfctx.GetGuami(),
		SecurityContext: models.SecurityContext{
			Ncc: int16(hoCtx.ncc),
			Nh:  hoCtx.nh,
		},
		NewSecInd: newSecInd,
	}
	targetRanUe.AddToPool()
	if rsp, ersp, err := targetRanUe.SendHandoverRequest(request); err != nil || ersp != nil {
		ueCtx.Errorf("Fail to request target gnB for handover")

		ueCtx.smfHandoverNotify(true)
		targetRanUe.ReleaseContext(models.N2Cause{})
		//send handover preparation failure
		return nil, &models.HandoverPreparationFailure{
			Cause: models.N2Cause{},
		}, nil
	} else {
		ueCtx.updateMetadata(rsp.NfSelection)
		//notify SMFs of handover prepared
		n2DlList := ueCtx.smfHandoverPrepared(rsp.Sessions)

		//mark success switched sessions
		for _, session := range n2DlList {
			if session.N2SmInfoType == models.N2SMINFOTYPE_HANDOVER_CMD {
				hoCtx.switchedSessions[uint8(session.SessionId)] = true
			}
		}

		//3b. send handover command to the source gnB
		//start handover timer
		hoCtx.timer.Start()
		return &models.HandoverCommand{
			TargetToSourceContent: rsp.TargetToSourceContent,
			Sessions:              n2DlList,
			HandoverType:          request.HandoverType,
		}, nil, nil
	}
}

// target gnB notify of handover complete
func (ueCtx *UeContext) HandleHandoverNotify(isGpp bool, msg *models.HandoverNotify) error {
	ueCtx.hoMu.Lock()
	defer ueCtx.hoMu.Unlock()

	if ueCtx.hoCtx == nil {
		return fmt.Errorf("No handover on-going")
	}

	//clear handover timer
	ueCtx.hoCtx.clearTimer()

	//1. notify SMF
	ueCtx.smfHandoverNotify(true)

	//2. ask source gnB to release UeContext
	ueCtx.hoCtx.source.ReleaseContext(models.N2Cause{})

	//3. replace source's RanUe with target's RanUe for UeContext
	ranUe := ueCtx.hoCtx.target

	ueCtx.ranUeMu.Lock()
	if ranUe.IsGpp() {
		ueCtx.gpp.ranUe = ranUe
	} else {
		ueCtx.nonGpp.ranUe = ranUe
	}
	ueCtx.ranUeMu.Unlock()

	//4. update NH and increase NCC for the current security context (now that
	//handover is completed)
	ueCtx.currentSecCtx.UpdateNh()
	return nil
}

// source gnB request to cancel on-going handover
func (ueCtx *UeContext) HandleHandoverCancel(isGpp bool, msg *models.HandoverCancel) (*models.HandoverCancelAcknowledge, error) {
	ueCtx.hoMu.Lock()
	defer ueCtx.hoMu.Unlock()

	if ueCtx.hoCtx == nil {
		return nil, fmt.Errorf("No handover on-going")
	}
	// prepare cancel acknowledgement
	rsp := &models.HandoverCancelAcknowledge{}
	// stop handover timer
	ueCtx.hoCtx.clearTimer()

	// notify SMFs to cancel on-going handover
	ueCtx.smfHandoverNotify(false)
	// send UeContext release to target gnB
	ueCtx.hoCtx.target.ReleaseContext(models.N2Cause{})

	return rsp, nil
}

func (ueCtx *UeContext) onHandoverTimeout() {
	ueCtx.hoMu.Lock()
	defer ueCtx.hoMu.Unlock()

	if ueCtx.hoCtx == nil {
		return
	}
	// notify SMFs to cancel on-going handover
	ueCtx.smfHandoverNotify(false)

	// send UeContext release to target gnB
	ueCtx.hoCtx.target.ReleaseContext(models.N2Cause{})

}

// clear timer to watch for handover expired
func (hoCtx *HandoverContext) clearTimer() {
	if hoCtx.timer != nil {
		hoCtx.timer.Stop()
		hoCtx.timer = nil
	}
}

// forward N2SmInfo from source gnB to SMF (HandoverRequire)
func (ueCtx *UeContext) smfHandoverPreparing(sessions []models.N2SmInfoUplinkContent) (n2DlList []models.N2SmInfoDownlinkContent) {
	hoCtx := ueCtx.hoCtx
	jobs := []func(){}

	for _, session := range sessions {
		if smCtx := ueCtx.findSmContext(uint8(session.SessionId)); smCtx != nil {
			hoCtx.switchedSessions[uint8(session.SessionId)] = false //mark swiching session
			jobs = append(jobs, func() {
				msg := &models.SmContextUpdateData{
					N2SmInfoType: models.N2SMINFOTYPE_HANDOVER_REQUIRED,
					HoState:      models.HOSTATE_PREPARING,
				}

				if rsp, ersp, err := smCtx.SendUpdateSmContext(msg, nil, session.N2SmInfo); err != nil {
					ueCtx.Errorf("Fail to forward HandoverRequired transfer for session %d", smCtx.GetId())
				} else {
					if n2, _ := ueCtx.processUpdateSmContextResponses(smCtx, rsp, ersp); n2 != nil {
						n2DlList = append(n2DlList, *n2)
					}
				}
			})
		} else {
			ueCtx.Errorf("Session %d not found", session.SessionId)
		}
	}
	//execute all jobs to request updates at SMFs
	executeTasks(jobs)
	return
}

// forward N2SmInfo from target gnB to SMF (HandoverRequestAck)
func (ueCtx *UeContext) smfHandoverPrepared(sessions []models.N2SmInfoUplinkContent) (n2DlList []models.N2SmInfoDownlinkContent) {
	jobs := []func(){}

	for _, session := range sessions {
		if smCtx := ueCtx.findSmContext(uint8(session.SessionId)); smCtx != nil {
			jobs = append(jobs, func() {
				msg := &models.SmContextUpdateData{
					N2SmInfoType: session.N2SmInfoType,
					HoState:      models.HOSTATE_PREPARED,
				}

				if rsp, ersp, err := smCtx.SendUpdateSmContext(msg, nil, session.N2SmInfo); err != nil {
					ueCtx.Errorf("Fail to forward HandoverRequestAck transfer for session %d", smCtx.GetId())
				} else {
					if n2, _ := ueCtx.processUpdateSmContextResponses(smCtx, rsp, ersp); n2 != nil {
						n2DlList = append(n2DlList, *n2)
					}
				}
			})
		} else {
			ueCtx.Errorf("Session %d not found", session.SessionId)
		}
	}

	//execute all jobs to request updates at SMFs
	executeTasks(jobs)
	return
}

// notify SMF of handover outcome (either success or cancelled)
func (ueCtx *UeContext) smfHandoverNotify(isCancel bool) {
	jobs := []func(){}

	for sessionId, success := range ueCtx.hoCtx.switchedSessions {
		if smCtx := ueCtx.findSmContext(sessionId); smCtx != nil {
			if isCancel { //handover cancel (all sessions)
				jobs = append(jobs, func() {
					if _, _, err := smCtx.SendUpdateSmContext(&models.SmContextUpdateData{
						HoState: models.HOSTATE_CANCELLED,
					}, nil, nil); err != nil {
						ueCtx.Errorf("Fail to notify handover status for session %d: %+v", sessionId, err)
					}
				})
			} else { //handover notify
				if success { //only success session
					jobs = append(jobs, func() {
						if _, _, err := smCtx.SendUpdateSmContext(&models.SmContextUpdateData{
							HoState: models.HOSTATE_COMPLETED,
						}, nil, nil); err != nil {
							ueCtx.Errorf("Fail to notify handover status for session %d: %+v", sessionId, err)
						}
					})
				}
			}
		} else {
			ueCtx.Errorf("Session %d not found to cancel", sessionId)
		}
	}

	//execute all jobs to request updates at SMFs
	executeTasks(jobs)
}
