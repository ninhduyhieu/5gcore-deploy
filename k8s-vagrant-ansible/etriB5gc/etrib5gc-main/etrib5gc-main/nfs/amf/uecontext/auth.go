package uecontext

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"etrib5gc/mesh"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/procs/auth"
)

// start authentication procedure
func (ueCtx *UeContext) authenticateUe() {
	attCtx := ueCtx.attCtx

	//create AUSF consumer client
	sid := common.AusfServiceName(ueCtx.plmnId)
	ausfCli, err := mesh.Consumer(sid, ueCtx.metadata)
	if err != nil {
		ueCtx.Errorf("Create Ausf consumer failed: %+v", err)
		ueCtx.abortAttachmentProcedure(nil)
	}

	ueCtx.Infof("Start authentication procedure")
	attCtx.proc = auth.Start(&auth.Context{
		Callback: ueCtx.onAuthenticationComplete,
		Log:      ueCtx.Entry,
		SendNasDl: func(pdu []byte) error {
			return ueCtx.sendNas(pdu, attCtx.isGpp)
		},
		Abba: ueCtx.abba,
		OnT3560: func(proc *auth.AuthProc) {
			ueCtx.sendEvent(context.TODO(), fsm.NewEventData(AuthTimerEvent, proc))
		},
		PlmnId:         amfctx.PlmnId(),
		Suci:           ueCtx.suci,
		AusfCli:        ausfCli,
		SelectNewNgKsi: ueCtx.selectNewNgKsi,
	})

}

func (ueCtx *UeContext) onAuthenticationComplete(rst *auth.Result, rejected bool) {
	//detached authentication procedure
	ueCtx.attCtx.proc = nil

	attCtx := ueCtx.attCtx
	if rst == nil {
		//TODO: if rejection was send, do not send another rejection
		ueCtx.Warnf("Authentication fail!")
		ueCtx.abortAttachmentProcedure(nil)
	} else {
		attCtx.eap = rst.Eap

		//update authenticated information for UeContext
		ueCtx.setSupi(rst.Supi)
		ueCtx.createNonCurrentSecCtx(rst.Kamf, &rst.NgKsi, attCtx.isGpp)
		ueCtx.goNextRegistrationStep()
	}
}
