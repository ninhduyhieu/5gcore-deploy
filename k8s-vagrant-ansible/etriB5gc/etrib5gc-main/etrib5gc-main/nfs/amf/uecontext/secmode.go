package uecontext

import (
	"context"
	"etrib5gc/internal/fsm"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/procs/secmode"
	"etrib5gc/nfs/amf/secctx"
	"github.com/reogac/nas"
)

// start security mode establishment procedure
func (ueCtx *UeContext) establishSecurityMode() {
	attCtx := ueCtx.attCtx
	secCtx := ueCtx.nonCurrentSecCtx
	retransmitNasMsg := true //always request retranmission initial NAS message (only set to false when SMC is not triggered by an attachment)
	ueCtx.Infof("Start security mode establishment procedure for %s with NAS re-transmittion=%v", secCtx.NgKsi().String(), retransmitNasMsg)

	//create keys for nas context
	encAlg, intAlg := amfctx.SelectAlgorithms(ueCtx.secCap)
	ueCtx.Tracef("Selected algorithms: Enc=%d, Int=%d", encAlg, intAlg)
	if err := secCtx.DeriveNasKeys(encAlg, intAlg, attCtx.hdp); err != nil {
		//send a reject
		ueCtx.Errorf("Fail to derive algorithm keys: %+v", err)
		ueCtx.abortAttachmentProcedure(nil)
		return
	} else {
		ueCtx.Infof("NAS keys are derived")
	}

	eap := ueCtx.attCtx.eap //attCtx must be non-nil

	//create builder to build SecurityModeCommand message
	builder := func(msg *nas.SecurityModeCommand) error {
		msg.ReplayedUeSecurityCapabilities = *ueCtx.secCap
		msg.AdditionalSecurityInformation = nas.NewAdditionalSecurityInformation(retransmitNasMsg, attCtx.hdp != secctx.HDP_NONE)

		if len(ueCtx.pei) > 0 {
			msg.ImeisvRequest = newUint8(nas.IMEISVNotRequested)
		} else {
			msg.ImeisvRequest = newUint8(nas.IMEISVNotRequested)
		}

		if len(eap) > 0 {
			msg.EapMessage = eap
			msg.Abba = ueCtx.abba
		}
		return nil
	}
	//start the security mode context establishment procedure with the UE
	attCtx.proc = secmode.Start(&secmode.Context{
		IsGpp:    attCtx.isGpp,
		Log:      ueCtx.Entry,
		Callback: ueCtx.onSecmodeComplete,
		Builder:  builder,
		SecCtx:   secCtx,
		SendNasDl: func(pdu []byte) error {
			return ueCtx.sendNas(pdu, attCtx.isGpp)
		},
		OnT3560: func(proc *secmode.SecmodeProcedure) {
			ueCtx.sendEvent(context.TODO(), fsm.NewEventData(SecmodeTimerEvent, proc))
		},
	})
}

func (ueCtx *UeContext) onSecmodeComplete(rst *secmode.Result, err error) {
	//detached security mode establishment procedure
	ueCtx.attCtx.proc = nil
	if err != nil {
		ueCtx.Errorf("Secmode establishment fails: %+v", err)
		if ueCtx.attCtx.regType == nas.RegistrationType5GSInitialRegistration {
			ueCtx.Infof("Force UE to start a primary authentication")
			ueCtx.resetAuthenticationContext()
			//go back to authentication
			ueCtx.goNextRegistrationStep()
		} else {
			ueCtx.abortAttachmentProcedure(nil)
		}
		return
	}

	//update pei
	if rst.Id != nil {
		//TODO: only accept IMEI
		if err := ueCtx.updateId(rst.Id); err != nil {
			ueCtx.Warnf("Fail to update UeContext with IMEISV: %+v", err)
		}
	}

	ueCtx.Infof("Security Mode is established for %s", rst.SecCtx.NgKsi().String())
	//must have a valid secmode now
	ueCtx.updateSecmode(rst.SecCtx)

	//as soon as we receive the first security protected NAS message, we should
	//derive AS keys
	if ueCtx.attCtx.sendContext {
		if err = ueCtx.currentSecCtx.DeriveAsKeys(); err != nil {
			ueCtx.Errorf("Derive AS key failed: %+v", err)
			n1Cause := nas.Cause5GMMProtocolErrorUnspecified
			ueCtx.abortAttachmentProcedure(&n1Cause)
			return
		} else {
			ueCtx.Infof("AS key is derived")
		}
	} else {
		ueCtx.Infof("RAN not request context")
	}

	n1Cause := nas.Cause5GMMInvalidMandatoryInformation
	if len(rst.Container) > 0 {
		ueCtx.Tracef("Decode nas container in SecmodComplete")
		if nasMSg, err := nas.Decode(nil, rst.Container, ueCtx.attCtx.isGpp); err == nil {
			//update the initial nas message
			if err = ueCtx.attCtx.updateNasMsg(nasMSg.Gmm); err != nil {
				ueCtx.Errorf("Update re-transmissed NasMsg failed: %+v", err)
				ueCtx.abortAttachmentProcedure(&n1Cause)
				return
			} else {
				ueCtx.Infof("Use Nas message from the re-transmited NAS container to continue")
			}
		} else {
			ueCtx.Errorf("Decode NasContainer failed: %+v", err)
			ueCtx.abortAttachmentProcedure(&n1Cause)
			return
		}
	} else {
		ueCtx.Errorf("UE did not re-transmit intial request in Nas container")
		ueCtx.abortAttachmentProcedure(&n1Cause)
		return
	}

	//move to context setting up step
	ueCtx.goNextRegistrationStep()
}
