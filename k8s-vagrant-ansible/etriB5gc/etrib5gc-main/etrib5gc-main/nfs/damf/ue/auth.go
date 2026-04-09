package ue

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"etrib5gc/mesh"
	"etrib5gc/nfs/amf/procs/auth"
	damfctx "etrib5gc/nfs/damf/context"
	"github.com/reogac/sbi/models"
)

func (ueCtx *UeContext) startAuthentication(ctx context.Context) {
	ueCtx.Debugf("suci=%s; plmnId=%s", ueCtx.suci.String(), models.PlmnIdToString(*ueCtx.plmnId()))

	//create ausf client
	sid := common.AusfServiceName(ueCtx.plmnId())
	ausfCli, err := mesh.Consumer(sid, ueCtx.metadata)
	if err != nil {
		ueCtx.Errorf("Fail to create Ausf client %+v", err)
		ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
			cause: CAUSE_CORE_FAIL,
		}))
		return
	}

	ueCtx.proc = auth.Start(&auth.Context{
		Callback:  ueCtx.onAuthenticationComplete(ctx),
		Log:       ueCtx.Entry,
		SendNasDl: ueCtx.sendNasDl,
		Abba:      damfctx.GetAbba(),
		OnT3560: func(proc *auth.AuthProc) {
			ueCtx.sendEvent(context.Background(), fsm.NewEventData(AuthTimerEvent, proc))
		},
		PlmnId:         damfctx.PlmnId(),
		Suci:           ueCtx.suci.String(),
		AusfCli:        ausfCli,
		SelectNewNgKsi: ueCtx.selectNewNgKsi,
	})

}
func (ueCtx *UeContext) onAuthenticationComplete(ctx context.Context) func(*auth.Result, bool) {
	return func(rst *auth.Result, rejected bool) {
		ueCtx.proc = nil
		//callback to be executed when authentication completes
		if rst == nil {
			//move to removed state
			if rejected {
				ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
					cause: CAUSE_AUTH_RJT,
				}))
			} else {
				ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
					cause: CAUSE_AUTH_FAIL,
				}))
			}
		} else {
			//update authentication context for the InitialUeMsg
			ueCtx.msg.AuthCtx = &models.UeAuthCtx{
				Supi:   rst.Supi,
				Kamf:   rst.Kamf,
				Eap:    rst.Eap,
				NgKsi:  rst.NgKsi,
				PlmnId: *ueCtx.plmnId(),
			}
			//move to fowarding state
			ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(ForwardEvent))
		}
	}
}
