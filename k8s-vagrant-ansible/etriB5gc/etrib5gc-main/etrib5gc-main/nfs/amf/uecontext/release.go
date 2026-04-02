package uecontext

import (
	"context"
	"github.com/reogac/sbi/models"
)

// handle UeContext Release Request from gnB
func (ueCtx *UeContext) ReceiveUeContextReleaseRequest(isGpp bool, msg *models.UeContextReleaseRequest) {
	//release existing pdu sessions (if they exist)
	ueCtx.deactivatePduSessions(msg.SessionList)
	ueCtx.releaseUeContextForAccess(context.Background(), isGpp)
}

// release context for a given access, detach RanUE from UeContext
func (ueCtx *UeContext) releaseUeContextForAccess(ctx context.Context, isGpp bool) {
	if isGpp {
		ueCtx.releaseUeContext(TARGET_GPP, models.N2Cause{})
	} else {
		ueCtx.releaseUeContext(TARGET_NON_GPP, models.N2Cause{})
	}
}

// release context (can be both accesses)
func (ueCtx *UeContext) releaseUeContext(targetRan uint8, cause models.N2Cause) {
	if targetRan == TARGET_GPP || targetRan == TARGET_BOTH {
		//3GPP
		if ranUe := ueCtx.gpp.ranUe; ranUe != nil {
			ueCtx.doReleaseUeContext(ranUe, cause)
			ueCtx.gpp.ranUe = nil
		}
	}
	if targetRan == TARGET_NON_GPP || targetRan == TARGET_BOTH {
		//Non3GPP
		if ranUe := ueCtx.nonGpp.ranUe; ranUe != nil {
			ueCtx.doReleaseUeContext(ranUe, cause)
			ueCtx.nonGpp.ranUe = nil
		}
	}
}

// command access side to release Ue
func (ueCtx *UeContext) doReleaseUeContext(ranUe RanUe, cause models.N2Cause) /*(models.UeContextReleaseComplete, error)*/ {
	if rsp, err := ranUe.ReleaseContext(cause); err != nil {
		ueCtx.Errorf("Fail to release RanUe: %+v", err)
	} else if rsp != nil {
		ueCtx.deactivatePduSessions(rsp.SessionList)
	}
	//any sessions for this access should be released now
	ueCtx.releasePduSessionsForAccess(ranUe.IsGpp())
}

func (ueCtx *UeContext) releasePduSessionsForAccess(isGpp bool) {
	sessions := ueCtx.sessions.getSessionsForAccess(isGpp)
	tasks := []func(){}
	for _, smCtx := range sessions {
		tasks = append(tasks, func() {
			ueCtx.releaseSmContext(smCtx, "")
		})
	}
	executeTasks(tasks)
}

func (ueCtx *UeContext) deactivatePduSessions(sessions []int16) {
	tasks := []func(){}
	for _, sId := range sessions {
		if smCtx := ueCtx.findSmContext(uint8(sId)); smCtx != nil {
			fn := func(rsp *models.UpdateSmContextResponse, ersp *models.UpdateSmContextErrorResponse) {
				//TODO:smCtx.sendN1N2SmDownlink(rsp, ersp)
			}

			tasks = append(tasks, func() {
				if smCtx.GetUpCnxState() == models.UPCNXSTATE_SUSPENDED {
					return
				}
				msg := &models.SmContextUpdateData{
					UpCnxState: models.UPCNXSTATE_DEACTIVATED,
					//TODO: add cause
				}

				if err := smCtx.UpdateSmContext(msg, nil, nil, fn); err != nil {
					ueCtx.Errorf("Fail to deactivate UpCnxState for session %d", smCtx.GetId())
				}
			})
		}
	}
	executeTasks(tasks)
}

/*
func (ueCtx *UeContext) releasePduSessions(sessions []int16, cause models.Cause, n2Cause models.N2Cause) {
	ngapCause := &models.NgApCause{
		Group: int(n2Cause.CausePresent),
		Value: int(n2Cause.CauseValue),
	}

	tasks := []func(){}
	for _, sId := range sessions {
		if smCtx := ueCtx.findSmContext(uint8(sId)); smCtx != nil {
			tasks = append(tasks, func() {
				smCtx.releaseSmContext(cause, ngapCause, nil, "", []byte{})
			})
		}
	}
	executeTasks(tasks)
}
*/
