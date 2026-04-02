package ue

import (
	"context"
	"etrib5gc/internal/fsm"
	"etrib5gc/nfs/amf/procs/iden"
	"github.com/reogac/nas"
)

func (ueCtx *UeContext) startIdentification(ctx context.Context) {
	//request SUCI
	idType := nas.MobileIdentity5GSTypeSuci
	ueCtx.proc = iden.Start(&iden.Context{
		Log:       ueCtx.Entry,
		Callback:  ueCtx.onIdentificationComplete(ctx),
		IdType:    idType,
		SendNasDl: ueCtx.sendNasDl,
		OnT3570: func(proc *iden.IdentificationProcedure) {
			ueCtx.sendEvent(context.Background(), fsm.NewEventData(IdenTimerEvent, proc))
		},
	})
}

func (ueCtx *UeContext) onIdentificationComplete(ctx context.Context) func(*nas.MobileIdentity, error) {
	return func(mobileId *nas.MobileIdentity, err error) {
		//detached identification procedure
		ueCtx.proc = nil

		if err != nil {
			ueCtx.Errorf("Identification fails: %+v", err)
			ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
				cause: CAUSE_ID_FAIL,
			}))
		} else {
			gSuci := mobileId.Id.(*nas.Suci)
			if format := gSuci.GetSupiFormat(); format == nas.SupiFormatImsi {
				ueCtx.suci = gSuci.Content.(*nas.SupiImsi)
				ueCtx.startAuthentication(ctx)
			} else {
				ueCtx.Errorf("UE send wrong SUCI format")
				ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
					cause: CAUSE_ID_FAIL,
				}))
			}
		}
	}
}
