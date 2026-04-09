package uecontext

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/eventmux"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/procs/iden"
	"fmt"
	"github.com/reogac/utils"
	"time"

	"etrib5gc/internal/fsm"
	"github.com/reogac/nas/secctx"
	"github.com/reogac/sbi/models"

	"github.com/reogac/nas"
)

const (
	MAX_T3550_CNT int = 2
	MAX_T3555_CNT int = 2
)

const (
	ATTACH_TIMEOUT time.Duration = time.Second
)

type CommonProc interface {
	ReceiveN1Mm(context.Context, *nas.DecodedGmmMessage)
	Close()
}
type RanUePool interface {
}

type AttachmentInfo struct {
	ranUe       RanUe
	authCtx     *models.UeAuthCtx      //authenticated information from DAMF
	gmmMsg      *nas.DecodedGmmMessage //decoded N1MM message
	sendContext bool                   //need to send InitialUeContextRequest?
	*eventmux.AsyncTask
}

// An AttachContext hold relevant information of a registration procedure
type AttachContext struct {
	registrationRequest *nas.RegistrationRequest
	serviceRequest      *nas.ServiceRequest
	regType             uint8 //registration type
	ueCtx               *UeContext
	nasPdu              []byte       //NAS RegistrationAccept
	t3550               common.Timer //Wait for NAS RegistrationComplete
	t3550Cnt            int
	proc                CommonProc //an on-going authentication, identification or security mode establishment procedure
	nasProtected        bool
	sendContext         bool                  //gnB request for InitialUeContextSetup
	hdp                 uint8                 //Kamf horizonal derivation indicator (intra-AMF mobility update, handover)
	report              *SyncPduSessionReport //for PduSession synchronization during attachment
	isGpp               bool                  //from GPP access?
	eap                 []byte                //Eap message from last authentication procedure (from DAMF)
}

// try to attach RanUe to UeContext
func (ueCtx *UeContext) BindRanUe(ranUe RanUe, msg *models.InitialUeMessage, gmm *nas.DecodedGmmMessage) error {
	ueCtx.updateMetadata(msg.NfSelection)
	info := &AttachmentInfo{
		ranUe:       ranUe,
		authCtx:     msg.AuthCtx,
		gmmMsg:      gmm,
		sendContext: msg.ContextRequest,
		AsyncTask:   eventmux.NewAsyncTask(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), ATTACH_TIMEOUT)
	defer cancel()
	if err := <-ueCtx.sendEvent(ctx, fsm.NewEventData(AttachEvent, info)); err != nil {
		return utils.WrapError("Send AttachEvent to FSM", err)
	} else {
		select {
		case <-ctx.Done():
		case err = <-info.Wait():
			return utils.WrapError("Handle Attachment", err)
		}
	}
	return nil
}

// handle RanUe attachment in state machine
func (ueCtx *UeContext) handleAttachment(ctx context.Context, info *AttachmentInfo) {
	// integrity check the message again now that Ue may have a valid security context
	if gmmMsg := info.gmmMsg; gmmMsg.SecHeader != nas.NasSecNone && ueCtx.currentSecCtx != nil {
		var ngKsi *nas.KeySetIdentifier
		switch gmmMsg.MsgType {
		case nas.RegistrationRequestMsgType:
			ngKsi = &gmmMsg.RegistrationRequest.Ngksi
		case nas.ServiceRequestMsgType:
			ngKsi = &gmmMsg.ServiceRequest.Ngksi
		case nas.DeregistrationRequestFromUeMsgType:
			ngKsi = &gmmMsg.DeregistrationRequestFromUe.Ngksi
		}
		if ngKsi != nil && (ngKsi.Id == 7 || ueCtx.isCurrentSecurityContext(ngKsi)) {
			nasCtx := ueCtx.currentSecCtx.NasContext()
			if nasMsg, err := nas.Decode(nasCtx, gmmMsg.Raw, info.ranUe.IsGpp()); err != nil { //TODO: just need to check for MAC code
				info.Finalize(utils.WrapError("Decode Nas with security context", err))
				return
			} else {
				ueCtx.Infof("Nas message is integrity protected")
				gmmMsg = nasMsg.Gmm
			}
		}
	}
	info.ranUe.AddToPool() //add RanUe to pool (create local ID)
	info.Finalize(nil)

	ueCtx.handleN1Mm(ctx, info.ranUe.IsGpp(), info.gmmMsg, info)
}

// start registration procedure [in MM_IDLE]
// analyze the request to update UeContext and bring up current or non-current security context
// then go to MM_REGISTERING
func (ueCtx *UeContext) startRegistration(ctx context.Context, isGpp bool, msg *nas.DecodedGmmMessage, initCtx *AttachmentInfo) {
	//create and attach a registration context to the UeContext
	attCtx := &AttachContext{
		ueCtx:        ueCtx,
		isGpp:        isGpp,
		nasProtected: !msg.MacFailed && msg.SecHeader != nas.NasSecNone,
		hdp:          secctx.HDP_NONE, //NOTE/TODO: depending on the type of registration
		t3550: common.NewTimer(amfctx.T3550(), func() {
			//Timer waiting for RegistrationComplete expired
			ueCtx.sendEvent(context.TODO(), fsm.NewEmptyEventData(RegCmplTimerEvent))
		}, nil),
	}

	var ngKsi *nas.KeySetIdentifier

	if msg.RegistrationRequest != nil {
		ueCtx.Infof("Handle NAS RegistrationRequest, NAS protection = %v", attCtx.nasProtected)
		regMsg := msg.RegistrationRequest
		attCtx.registrationRequest = regMsg

		//get key identifier and registration type from the request
		ngKsi = &regMsg.Ngksi
		//check registration type
		registrationType := regMsg.RegistrationType.GetType()
		switch registrationType {
		case nas.RegistrationType5GSInitialRegistration:
			ueCtx.Tracef("RegistrationType: Initial Registration")

		case nas.RegistrationType5GSMobilityRegistrationUpdating:
			ueCtx.Tracef("RegistrationType: Mobility Registration Updating")
			if !ueCtx.isRegistered(isGpp) {
				ueCtx.Errorf("Invalid registation type")
				ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GSMMessageNotCompatibleWithTheProtocolState))
				return
			}

		case nas.RegistrationType5GSPeriodicRegistrationUpdating:
			ueCtx.Tracef("RegistrationType: Periodic Registration Updating")
			if !ueCtx.isRegistered(isGpp) {
				ueCtx.Errorf("Invalid registation type")
				ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GSMMessageNotCompatibleWithTheProtocolState))
				return
			}

		case nas.RegistrationType5GSEmergencyRegistration:
			ueCtx.Errorf("Emergency Registration not supported")
			ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GSMMessageNotCompatibleWithTheProtocolState))
			return

		case nas.RegistrationType5GSReserved:
			ueCtx.Tracef("RegistrationType: Reserved")
			registrationType = nas.RegistrationType5GSInitialRegistration

		default:
			ueCtx.Warnf("RegistrationType: %v, change state to InitialRegistration", registrationType)
			registrationType = nas.RegistrationType5GSInitialRegistration
		}
		attCtx.regType = registrationType
		ueCtx.updateCapabilities(regMsg)
	} else {
		ueCtx.Infof("Handle NAS ServiceRequest, NAS protection = %v", attCtx.nasProtected)
		attCtx.serviceRequest = msg.ServiceRequest
		ngKsi = &msg.ServiceRequest.Ngksi
	}

	ueCtx.Tracef("UE indicates security context: %s", ngKsi.String())

	//try to decode NasContainer and bring up security context
	if !attCtx.nasProtected {
		if ngKsi.Id == 7 || ueCtx.isCurrentSecurityContext(ngKsi) { //UE indicates a current security context
			var nasContainer []byte
			if attCtx.registrationRequest != nil { //there is a protected nas payload container
				nasContainer = attCtx.registrationRequest.NasMessageContainer
			} else if msg.ServiceRequest != nil {
				nasContainer = attCtx.serviceRequest.NasMessageContainer
			}
			if len(nasContainer) > 0 {
				nasCtx := ueCtx.currentSecCtx.NasContext()
				if gmmMsg, err := nasCtx.DecodeMmContainer(nasContainer, attCtx.isGpp); err != nil {
					ueCtx.Warnf("Fail to decode NAS message container in RegistrationRequest: %+v", err)
				} else {
					ueCtx.Infof("NAS container is decoded with the current security context, NAS is protected")
					attCtx.nasProtected = true
					if gmmMsg.RegistrationRequest != nil && attCtx.registrationRequest != nil {
						attCtx.registrationRequest = gmmMsg.RegistrationRequest
						ueCtx.updateCapabilities(gmmMsg.RegistrationRequest)
					} else if gmmMsg.ServiceRequest != nil && attCtx.serviceRequest != nil {
						attCtx.serviceRequest = gmmMsg.ServiceRequest
					} else {
						ueCtx.Warnf("Invalid message type in NAS container: %d", gmmMsg.MsgType)
					}
				}
			} else {
				ueCtx.Debugf("Request has no NasContainer")
			}
		}
	}

	if !attCtx.nasProtected { //nas is not protected, reset authentication context
		ueCtx.resetAuthenticationContext()
	}

	if msg.ServiceRequest != nil && !attCtx.nasProtected {
		//ServiceRequest must be nas protected
		ueCtx.Warnf("ServiceRequest is not security protected, reject it")
		ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GMMSemanticallyIncorrectMessage))
		return
	}

	//reject if Ue still has no security capability
	if ueCtx.secCap == nil {
		ueCtx.Warnf("Ue has not provided security capabilities")
		ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GMMInvalidMandatoryInformation))
		return
	}

	//if Ue has been authenticated at DAMF, update the UeContext authenticated
	//infos and create non-current security context
	if initCtx != nil { //registration with InitialUeMessage
		attCtx.sendContext = initCtx.sendContext
		if authCtx := initCtx.authCtx; authCtx != nil { //a primaty authentication has been occured (at DAMF)
			if err := ueCtx.setHomeNetwork(&authCtx.PlmnId); err != nil {
				ueCtx.Errorf("Fail to connect home network", err)
				ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GMMProtocolErrorUnspecified))
				return
			}

			attCtx.eap = authCtx.Eap
			//update UeContext with authenticated information from DAMF
			ueCtx.setSupi(authCtx.Supi)
			ueCtx.createNonCurrentSecCtx(authCtx.Kamf, &authCtx.NgKsi, isGpp)
			ueCtx.amData = authCtx.AmData
		}
	}

	//as soon as we receive the first security protected NAS message, we should
	//derive AS keys
	if attCtx.nasProtected && attCtx.sendContext { //has a valid security context
		if err := ueCtx.currentSecCtx.DeriveAsKeys(); err != nil {
			ueCtx.Errorf("Fail to derive AS key: %+v", err)
			ueCtx.rejectAttachment(attCtx, newUint8(nas.Cause5GMMProtocolErrorUnspecified))
			return
		} else {
			ueCtx.Infof("AS key is derived")
		}
	}

	ueCtx.attCtx = attCtx

	//go to MM_REGISTERING
	if initCtx != nil {
		ueCtx.state.SetNextEvent(fsm.NewEventData(RegisterEvent, initCtx.ranUe))
	} else {
		ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(RegisterEvent))
	}
}

func (r *AttachContext) getRanUe() RanUe {
	return r.ueCtx.getRanUe(r.isGpp)
}

// abort registration, send registration/service reject toward UE
func (r *AttachContext) abort(n1Cause *uint8) {
	if r.proc != nil {
		r.proc.Close()
		r.proc = nil
	}

	r.ueCtx.rejectAttachment(r, n1Cause)
	r.ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(RegisterFailEvent))
	return
}

func (ueCtx *UeContext) abortAttachmentProcedure(n1Cause *uint8) {
	ueCtx.attCtx.abort(n1Cause)
}

// send reject through a RanUe
func (ueCtx *UeContext) rejectAttachment(attCtx *AttachContext, n1Cause *uint8) {
	//send a registration reject
	var n1CauseValue uint8 = nas.Cause5GMMProtocolErrorUnspecified
	if n1Cause != nil {
		n1CauseValue = *n1Cause
	}

	if attCtx.registrationRequest != nil { //send registration reject
		ueCtx.sendRegistrationReject(attCtx.isGpp, n1CauseValue)
	} else { //send service reject
		var statusList *[16]bool
		if attCtx.report != nil {
			statusList = attCtx.report.StatusList
		}
		ueCtx.sendServiceReject(attCtx.isGpp, n1CauseValue, statusList)
	}
}

// NasMessage is re-transmissed in a NasContainer of the SecurityModeComplete
func (ctx *AttachContext) updateNasMsg(gmmMsg *nas.DecodedGmmMessage) (err error) {
	if gmmMsg == nil {
		err = fmt.Errorf("UE send an Empty N1MM in the NAS container")
		return
	}
	if ctx.registrationRequest != nil {
		if gmmMsg.RegistrationRequest != nil {
			ctx.registrationRequest = gmmMsg.RegistrationRequest
			//NOTE: should UeContext be updated with information from the request
			ctx.ueCtx.updateCapabilities(ctx.registrationRequest)
		} else {
			err = fmt.Errorf("RegistrationRequest not found in the decoded NAS container")
		}
	} else {
		if gmmMsg.ServiceRequest != nil {
			ctx.serviceRequest = gmmMsg.ServiceRequest
		} else {
			err = fmt.Errorf("ServiceRequest not found in the decoded NAS container")
		}
	}
	return
}

func (ueCtx *UeContext) onT3570(proc *iden.IdentificationProcedure) {
	ueCtx.sendEvent(context.TODO(), fsm.NewEventData(IdenTimerEvent, proc))
}

// in registering, go to the next phase
func (ueCtx *UeContext) goNextRegistrationStep() {
	//depending on current status, select a next common procedure to proceed
	if ueCtx.currentSecCtx != nil { //has current security context
		ueCtx.setupContext()
	} else if len(ueCtx.supi) > 0 { //authenticated
		ueCtx.establishSecurityMode()
	} else if len(ueCtx.suci) > 0 { //has identity (suci)
		ueCtx.authenticateUe()
	} else { //has nothing
		ueCtx.identifyUe()
	}

}

// finalize registration procedure
func (attCtx *AttachContext) accept() {
	ueCtx := attCtx.ueCtx
	report := attCtx.report
	//handle pending N1N2

	//prepare to include n1n2 messages in sending acceptance
	/* NOTE: must comment now
	n1n2 := ueCtx.n1n2
		if forwardN1N2 {
			var dlN2SmInfo *models.N2SmInfoDownlinkContent
			if n1n2.smCtx != nil {
				//there is n2SmInfo
				dlN2SmInfo = &models.N2SmInfoDownlinkContent{
					SessionId:    int16(n1n2.smCtx.id),
					N2SmInfo:     n1n2.n2SmInfo,
					N2SmInfoType: n1n2.n2SmInfoType,
					Snssai:       &n1n2.smCtx.snssai.AllowedSnssai,
				}
			}

			if n1 := n1n2.n1Msg; n1 != nil {
				sessionId := n1.sessionId
				if n1.n1type == nas.PayloadContainerTypeN1SMInfo {
					sessionId = &n1n2.smCtx.id
				}
				if nasPdu, err := ranUe.buildDlNasTransport(n1.n1type, n1.payload, sessionId, nil, nil, 0); err == nil {
					ranUe.Errorf("Fail to encode DlNasTransport")
					ueCtx.notifyN1N2Failure(true)
				} else {
					if n1.n1type == nas.PayloadContainerTypeN1SMInfo && dlN2SmInfo != nil {
						dlN2SmInfo.NasPdu = nasPdu
					} else {
						report.n1Msg = nasPdu
					}
				}
			}

			if dlN2SmInfo != nil {
				report.n2SmInfoList = append(report.n2SmInfoList, *dlN2SmInfo)
			}
		}
	*/
	var err error
	//build N1 acceptance message
	if attCtx.registrationRequest != nil {
		attCtx.nasPdu, err = ueCtx.buildRegistrationAccept(attCtx.isGpp)
	} else {
		attCtx.nasPdu, err = ueCtx.buildServiceAccept(attCtx.isGpp)
	}

	if err != nil {
		ueCtx.Errorf("Fail to build Nas acceptance message: %+v", err)
		ueCtx.abortAttachmentProcedure(nil)
		return
	}

	var n2SmInfoList []models.N2SmInfoDownlinkContent
	if report != nil {
		n2SmInfoList = report.N2SmInfoList
	}

	//"Context Request" indicator received in the InitialUeMessage
	ranUe := ueCtx.getRanUe(attCtx.isGpp)
	if attCtx.sendContext {
		//need to send InitialContextRequest
		if attCtx.isGpp {
			if err = ueCtx.sendInitialContextSetupRequest(attCtx.isGpp, n2SmInfoList, attCtx.nasPdu); err != nil {
				ueCtx.Errorf("Fail to send InitialContextSetupRequest with N1MM acceptance message: %+v", err)
			} else {
				ueCtx.Infof("IntialContextSetupRequest with N1MM acceptance was sent to gnB")
			}
		} else { //non-3gpp: send InitialContextSetupRequest first, the send NAS
			if err = ueCtx.sendInitialContextSetupRequest(attCtx.isGpp, n2SmInfoList, nil); err == nil {
				ueCtx.Infof("IntialContextSetupRequest was sent to gnB")
				if err = ranUe.SendNas(attCtx.nasPdu); err != nil {
					ueCtx.Errorf("Fail to send N1MM acceptance mesasge for UE to gnB: %+v", err)
				} else {
					ueCtx.Infof("N1MM acceptance message for UE was sent to gnB")
				}
			} else {
				ueCtx.Errorf("Fail to send InitialContextSetupRequest to gnB: %+v", err)
			}
		}
		//TODO: may need to deregister UE if we fail to request RAN to
		//setup UE context
	} else {
		if len(n2SmInfoList) > 0 {
			if err = ranUe.SendN2SmInfoDownlink(n2SmInfoList, attCtx.nasPdu); err != nil {
				ueCtx.Errorf("Fail to N2SmInfoDownlink: %+v", err)
			} else {
				ueCtx.Infof("N2SmInfoDownlink was sent to gnB")
			}
		} else {
			if err = ranUe.SendNas(attCtx.nasPdu); err != nil {
				ueCtx.Errorf("Fail to N1MM acceptance message for UE to gnB:  %+v", err)
			} else {
				ueCtx.Infof("N1MM acceptance message for UE was sent to gnB")
			}
		}
	}

	//fail to send NAS acceptance and N1N2 messages downlink
	if err != nil {
		ueCtx.abortAttachmentProcedure(nil)
		return
	}

	if attCtx.registrationRequest != nil {
		//start timer to wait for RegistrationComplete
		ueCtx.Debugf("Start T3550 to wait for RegistrationComplete from UE")
		attCtx.t3550.Start()
		attCtx.t3550Cnt++
	} else {
		//TODO: may need to send NAS ConfigurationUpdateCommand
		//move to MM_IDLE
		ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(RegisterDoneEvent))
	}

	if report != nil {
		//send pending N1 messages
		tasks := []func(){}

		if len(report.N1Msg) > 0 {
			tasks = append(tasks, func() {
				if err := ranUe.SendNas(report.N1Msg); err != nil {
					ueCtx.Errorf("Fail to send N1Message downlink: %+v", err)
					ueCtx.notifyN1N2Failure(true)
					return
				}
			})
		}

		for _, n1Msg := range report.N1MsgList {
			tasks = append(tasks, func() {
				if err := ranUe.SendNas(n1Msg); err != nil {
					ueCtx.Errorf("Fail to send N1Message downlink: %+v", err)
					return
				}
			})
		}

		executeTasks(tasks)
	}
	//N1N2 messages have been forwarded, clear them
	removePendingN1N2(ueCtx.n1n2)
	ueCtx.n1n2 = nil
}
