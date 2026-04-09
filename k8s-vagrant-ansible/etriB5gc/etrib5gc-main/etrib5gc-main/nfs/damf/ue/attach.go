package ue

import (
	"context"
	"etrib5gc/internal/fsm"
	"github.com/reogac/nas"
)

// extract UeContext data from Nas MM message
func (ueCtx *UeContext) handleAttachment(ctx context.Context) {
	gmm := ueCtx.gmm
	var mobileId *nas.MobileIdentity
	switch gmm.MsgType {
	case nas.RegistrationRequestMsgType:
		ueCtx.Tracef("Init UeContext with RegistrationRequest")
		mobileId = &gmm.RegistrationRequest.MobileIdentity
	case nas.ServiceRequestMsgType:
		ueCtx.Tracef("Init UeContext with ServiceRequest")
		mobileId = &gmm.ServiceRequest.STmsi
	case nas.DeregistrationRequestFromUeMsgType:
		ueCtx.Tracef("Init UeContext with DeregistrationRequest")
		mobileId = &gmm.DeregistrationRequestFromUe.MobileIdentity
	default:
		ueCtx.Errorf("Not supoported initial Ue Message: %d", gmm.MsgType)
		ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
			cause: CAUSE_N1MM,
		}))
		return
	}

	idType := mobileId.GetType()
	switch idType {
	case nas.MobileIdentity5GSTypeSuci:
		suci := mobileId.Id.(*nas.Suci)
		if supiFormat := suci.GetSupiFormat(); supiFormat == nas.SupiFormatImsi {
			ueCtx.suci = suci.Content.(*nas.SupiImsi)
			//start authentication here
			ueCtx.startAuthentication(ctx)
			return
		} else {
			ueCtx.Errorf("Not support SUCI format %d in RegistrationRequest", supiFormat)
		}
	case nas.MobileIdentity5GSType5gGuti:
		guti := mobileId.Id.(*nas.Guti)
		ueCtx.Debugf("Inital Nas message with a Guti: %s", guti.String())
		if isGutiSupported(guti) {
			if err := ueCtx.createAmfClient(&guti.AmfId); err != nil {
				ueCtx.Warnf("Fail to connect Amf: %+v", err)
			} else {
				//move to fowarding state
				ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(ForwardEvent))
				return
			}
		}

	case nas.MobileIdentity5GSType5gSTmsi:
		tmsi5gs := mobileId.Id.(*nas.Tmsi5Gs)
		ueCtx.Debugf("RegistrationRequest with a 5GsTmsi: %s", tmsi5gs.String())
		amfId := tmsi5gs.AmfId
		amfId.SetRegion(ueCtx.amfRegion)
		//create Amf client
		if err := ueCtx.createAmfClient(&amfId); err != nil {
			ueCtx.Errorf("Fail to connect Amf: %+v", err)
		} else {
			//move to fowarding state
			ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(ForwardEvent))
			return
		}
	default: //other type of identities is not accepted at initial AMF
		ueCtx.Warn("SUCI/GUTI/5GSTMSI is missing")
	}

	switch gmm.MsgType {
	case nas.ServiceRequestMsgType, nas.DeregistrationRequestFromUeMsgType:
		//can't infer AMF id from mobile identity
		//fail to get AmData for Ue
		ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
			cause: CAUSE_NO_AMFID,
		}))

	default:
		ueCtx.startIdentification(ctx)
	}

}
