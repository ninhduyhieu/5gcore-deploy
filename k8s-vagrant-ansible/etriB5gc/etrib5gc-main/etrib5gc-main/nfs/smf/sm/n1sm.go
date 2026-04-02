package sm

import (
	"etrib5gc/internal/fsm"
	smfctx "etrib5gc/nfs/smf/context"
	"etrib5gc/nfs/smf/tunnel"

	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func (smCtx *SmContext) buildGsmStatus(pti uint8, cause uint8) ([]byte, error) {
	msg := &nas.GsmStatus{
		GsmCause: cause,
	}
	return smCtx.buildN1Sm(msg, pti)
}

func (smCtx *SmContext) sendGsmStatus(pti uint8, cause uint8) {
	if pdu, err := smCtx.buildGsmStatus(pti, cause); err != nil {
		smCtx.Errorf("Fail to build 5GsmStatus: %+v", err)
	} else {
		if err = smCtx.sendN1Sm(pdu); err != nil {
			smCtx.Errorf("Fail to send N1Sm: %+v", err)
		}
	}
}

func (smCtx *SmContext) sendN1Sm(n1Sm []byte) error {
	n1n2 := &N1N2Messages{
		n1Sm: n1Sm,
	}
	return smCtx.n1n2.sendN1N2(n1n2)
}

// add Pti and session identity then encode the N1Sm message
func (smCtx *SmContext) buildN1Sm(msg nas.GsmMessage, pti uint8) (pdu []byte, err error) {
	msg.SetPti(pti)
	msg.SetSessionId(uint8(smCtx.pduSessionId))
	var n1Sm []byte
	if n1Sm, err = nas.EncodeSm(msg); err != nil {
		err = utils.WrapError("Encode N1SM", err)
		return
	}
	if smCtx.n1n2.ranCli != nil { //wrapped in a NasDownlinkTransport
		dlMsg := &nas.DlNasTransport{
			PayloadContainerType: nas.PayloadContainerTypeN1SMInfo,
			PayloadContainer:     n1Sm,
			PduSessionId:         newUint8(uint8(smCtx.pduSessionId)),
		}

		dlMsg.SetSecurityHeader(nas.NasSecNone)
		if pdu, err = nas.EncodeMm(nil, dlMsg, true); err != nil {
			err = utils.WrapError("Encode N1MM", err)
		}
	} else {
		pdu = n1Sm
	}
	return
}

func (smCtx *SmContext) handleN1Sm(msg *nas.DecodedGsmMessage) {
	smCtx.Debug("Handle N1Sm")
	switch msg.MsgType {
	case nas.PduSessionEstablishmentRequestMsgType:
		smCtx.handlePduSessionEstablishmentRequest(msg.PduSessionEstablishmentRequest)

	case nas.PduSessionReleaseRequestMsgType:
		smCtx.handlePduSessionReleaseRequest(msg.PduSessionReleaseRequest)

	case nas.PduSessionReleaseCompleteMsgType:
		smCtx.handlePduSessionReleaseComplete(msg.PduSessionReleaseComplete)

	case nas.PduSessionModificationRequestMsgType:
		smCtx.handlePduSessionModificationRequest(msg.PduSessionModificationRequest)

	case nas.PduSessionModificationCompleteMsgType:
		smCtx.handlePduSessionModificationComplete(msg.PduSessionModificationComplete)

	case nas.PduSessionModificationCommandRejectMsgType:
		smCtx.handlePduSessionModificationCommandReject(msg.PduSessionModificationCommandReject)

	default:
		smCtx.Errorf("Receive unexpected N1Sm message type: %d", msg.MsgType)
		smCtx.sendGsmStatus(nas.PtiUnspecified, nas.Cause5GSMMessageTypeNonExistentOrNotImplemented)
	}
	return
}

// NOTE: session must be in SM_INACTIVE or SM_ACTIVE
// Why SM_ACTIVE? need to check if it is correct. May it be related to handover?
func (smCtx *SmContext) handlePduSessionEstablishmentRequest(req *nas.PduSessionEstablishmentRequest) {
	smCtx.Debugf("Handle N1Sm PduSessionEstablishmentRequest")
	state := smCtx.state.CurrentState()
	if state != SM_INACTIVE {
		smCtx.Warnf("N1Sm PduSessionEstablishmentRequest not handled in state %s", stateString(state))
		smCtx.rejectPduSessionEstablishmentRequest(req.GetPti(), nas.Cause5GSMMessageNotCompatibleWithTheProtocolState)
		return
	}

	//update SmContext with information from N1Sm
	// Retrieve PduSessionID
	smCtx.Infof("Process N1SM PduSessionEstablishmentRequest")
	smCtx.pduSessionId = uint32(req.GetSessionId())

	if req.SscMode != nil {
		smCtx.sscMode = *req.SscMode
	}

	// Retrieve MaxIntegrityProtectedDataRate of UE for
	// UP Security
	switch req.IntegrityProtectionMaximumDataRate.Uplink() {
	case 0x00:
		smCtx.maxUplink = models.MAXINTEGRITYPROTECTEDDATARATE_64_KBPS
	case 0xff:
		smCtx.maxUplink = models.MAXINTEGRITYPROTECTEDDATARATE_MAX_UE_RATE
	}

	switch req.IntegrityProtectionMaximumDataRate.Downlink() {
	case 0x00:
		smCtx.maxDownlink = models.MAXINTEGRITYPROTECTEDDATARATE_64_KBPS
	case 0xff:
		smCtx.maxDownlink = models.MAXINTEGRITYPROTECTEDDATARATE_MAX_UE_RATE
	}

	// Handle PduSessionType
	acceptCause, rejectCause, sessionType := smfctx.GetSessionType(req.PduSessionType, smCtx.dnn, smCtx.smData)
	if rejectCause != nil {
		smCtx.Errorf("Fail to determine PduSessionType")
		smCtx.rejectPduSessionEstablishmentRequest(req.GetPti(), *rejectCause)
		return
	}
	smCtx.sessionType = sessionType
	smCtx.processPco(req.ExtendedProtocolConfigurationOptions)

	if err := smCtx.activateSessionResources(); err != nil {
		smCtx.Errorf("Fail activate resources for session: %+v", err)
		smCtx.cancelSessionActivation(req.GetPti(), nas.Cause5GSMNetworkFailure)
		return
	}

	// send acceptance toward gnB and UE
	var n1Sm, n2SmInfo []byte
	var err error
	if n2SmInfo, err = smCtx.buildPduSessionResourceSetupRequestTransfer(); err != nil {
		smCtx.Errorf("Build PDUSessionResourceSetupRequestTransfer failed: %+v", err)
		smCtx.cancelSessionActivation(req.GetPti(), nas.Cause5GSMNetworkFailure)
		return
	}
	smCtx.Info("N2Info PDUSessionResourceSetupRequestTransfer is built")

	if n1Sm, err = smCtx.buildPduSessionEstablishmentAccept(req.GetPti(), acceptCause); err != nil {
		smCtx.Errorf("Build GSM PDUSessionEstablishmentAccept failed: %+v", err)
		smCtx.cancelSessionActivation(req.GetPti(), nas.Cause5GSMNetworkFailure)
		return
	}
	smCtx.Info("N1Sm PDUSessionEstablishmentAccept is built")

	n1n2 := &N1N2Messages{
		n1Sm:         n1Sm,
		n2SmInfo:     n2SmInfo,
		n2SmInfoType: models.N2SMINFOTYPE_PDU_RES_SETUP_REQ,
	}

	if err = smCtx.n1n2.sendN1N2(n1n2); err != nil {
		smCtx.Errorf("Fail to send N1N2 message: %+v", err)
		smCtx.cancelSessionActivation(req.GetPti(), nas.Cause5GSMNetworkFailure)
	} else {
		smCtx.Infof("N1N2Message was sent")
		smCtx.upCnxState = UPCNXSTATE_ACTIVATING
		smCtx.state.SetNextEvent(fsm.NewEmptyEventData(ActCmplEvent))
	}
}

// create SM policy then activate data paths
func (smCtx *SmContext) activateSessionResources() error {
	smCtx.Tracef("Start PDU session resource activation")

	//1.  create SmPolicy (request PCF)
	if err := smCtx.createSmPol(); err != nil {
		return utils.WrapError("Create SmPolicy", err)
	}
	smCtx.Infof("Retrieved SmPolicy from PCF")

	//1. allocate UeIp
	if ueIp, err := smfctx.AllocateUeIp(smCtx.dnn); err != nil {
		return utils.WrapError("Allocate Ue IP", err)
	} else {
		smCtx.ueIp = ueIp
	}

	smCtx.Infof("IP address is allocated for UE: %s", smCtx.ueIp.IP().String())

	//2. create a tunnel
	smCtx.tunnel = tunnel.CreateTunnel(smCtx, smCtx.ranTransportNets)
	smCtx.Infof("A tunnel has been created for the session")

	//3. create datapaths and interact with UPFs
	if err := smCtx.tunnel.HandleSmPolicyDecision(smCtx.smPol); err != nil {
		return utils.WrapError("Setup User Plane data paths", err)
	}
	return nil
}

// remove sm policy and remove data paths
func (smCtx *SmContext) releaseSessionResources() {
	//1. tel PCF to release session context
	if err := smCtx.deleteSmPol(); err != nil {
		smCtx.Warnf("Fail to delete SmPol at PCF: %+v", err)
	} else {
		smCtx.smPolId = ""
		smCtx.Infof("SmPolicy was deleted")
	}
	//2. release Ue IP
	if smCtx.ueIp != nil {
		smCtx.ueIp.Free()
		smCtx.ueIp = nil
		smCtx.Infof("Ue ip address was released")
	}

	//3. release the tunnel (if it is created/activated
	if smCtx.tunnel != nil {
		if err := smCtx.tunnel.Release(); err != nil {
			smCtx.Errorf("Fail to release tunnel: %+v", err)
		}
		smCtx.tunnel = nil
		smCtx.Infof("UP tunnel was released")
	}
}

// release all session resource; send rejection then remove SmContext
func (smCtx *SmContext) cancelSessionActivation(pti, cause uint8) {
	smCtx.releaseSessionResources()
	smCtx.rejectPduSessionEstablishmentRequest(pti, cause)

	//Remove SmContext if it is neccesary
	_pool.remove(smCtx)
	if err := smCtx.notifyAmf(); err != nil {
		smCtx.Warnf("Fail to notify AMF: %+v", err)
	}
}

// send N1Sm to reject session establishment
func (smCtx *SmContext) rejectPduSessionEstablishmentRequest(pti, cause uint8) {
	if n1Sm, err := smCtx.buildPduSessionEstablishmentReject(pti, cause); err != nil {
		smCtx.Errorf("Fail to build PduSessionEstablishmentReject: %+v", err)
		smCtx.sendGsmStatus(pti, nas.Cause5GSMNetworkFailure)
	} else if err = smCtx.sendN1Sm(n1Sm); err != nil {
		smCtx.Errorf("Fail to send PduSessionEstablishmentReject: %+v", err)
	}
}

// NOTE: session must be in SM_ACTIVE state
func (smCtx *SmContext) handlePduSessionReleaseRequest(req *nas.PduSessionReleaseRequest) {
	smCtx.Debugf("Handle N1Sm PduSessionReleaseRequest")
	state := smCtx.state.CurrentState()
	if state != SM_ACTIVE {
		n1SmCause := nas.Cause5GSMMessageNotCompatibleWithTheProtocolState
		if n1Sm, err := smCtx.buildPduSessionReleaseReject(req.GetPti(), n1SmCause); err != nil {
			smCtx.Errorf("Fail to build PduSessionReleaseReject: %+v", err)
			smCtx.sendGsmStatus(req.GetPti(), nas.Cause5GSMNetworkFailure)
		} else if smCtx.sendN1Sm(n1Sm); err != nil {
			smCtx.Errorf("Fail to send PduSessionReleaseReject: %+v", err)
		} else {
			smCtx.Infof("N1SM PduSessionReleaseReject was sent")
		}
		return
	}

	//trigger a release procedure
	event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
		evType: RELEASE_UE,
		dat:    req,
	})
	smCtx.state.SetNextEvent(event)
	/*
		smCtx.handleReleaseTrigger(&EventData{
			evType: RELEASE_UE,
			dat:    req,
		})
	*/
	return
}

// NOTE: session should be in SM_INACTIVATING
func (smCtx *SmContext) handlePduSessionReleaseComplete(req *nas.PduSessionReleaseComplete) {
	smCtx.Debugf("Handle N1Sm PduSessionReleaseComplete")
	curState := smCtx.state.CurrentState()
	if curState != SM_INACTIVATING {
		smCtx.Errorf("PduSessionReleaseComplete not handled in current state[%d]", curState)
		smCtx.sendGsmStatus(req.GetPti(), nas.Cause5GSMMessageTypeNotCompatibleWithTheProtocolState)
		return
	}

	smCtx.Infof("Received N1Sm PduSessionReleaseComplete")
	smCtx.finalizeRelease(smCtx.relCtx.trigger)
	return
}

// Must be in SM_ACTIVE
func (smCtx *SmContext) handlePduSessionModificationRequest(req *nas.PduSessionModificationRequest) {
	smCtx.Debugf("Handle N1Sm PduSessionModificationRequest")
	state := smCtx.state.CurrentState()
	if state != SM_ACTIVE {
		n1SmCause := nas.Cause5GSMMessageNotCompatibleWithTheProtocolState
		if n1Sm, err := smCtx.buildPduSessionModificationReject(req.GetPti(), n1SmCause); err != nil {
			smCtx.Errorf("Fail to build PduSessionModificationReject: %+v", err)
			smCtx.sendGsmStatus(req.GetPti(), nas.Cause5GSMNetworkFailure)
		} else if smCtx.sendN1Sm(n1Sm); err != nil {
			smCtx.Errorf("Fail to send PduSessionModificationReject: %+v", err)
		} else {
			smCtx.Infof("N1SM PduSessionModificationReject was sent")
		}
		return
	}

	event := fsm.NewEventData(ModificationTriggerEvent, &EventData{
		evType: MODIFY_N1SM,
		dat:    req,
	})
	smCtx.state.SetNextEvent(event)
}

// Must be in SM_MODIFYING
func (smCtx *SmContext) handlePduSessionModificationCommandReject(req *nas.PduSessionModificationCommandReject) {
	smCtx.Debugf("Handle N1Sm PduSessionModificationReject")
	state := smCtx.state.CurrentState()
	if state != SM_MODIFYING {
		smCtx.Errorf("PduSessionModificationReject not handled in current state[%d]", state)
		smCtx.sendGsmStatus(req.GetPti(), nas.Cause5GSMMessageTypeNotCompatibleWithTheProtocolState)
		return
	}

	smCtx.Infof("Received N1Sm PduSessionModificationReject")
	//TODO:  finalize/cancel the modification
	//then move back to SM_ACTIVE
	event := fsm.NewEmptyEventData(ModCmplEvent)
	smCtx.state.SetNextEvent(event)
}

// Must be in SM_MODIFYING
func (smCtx *SmContext) handlePduSessionModificationComplete(req *nas.PduSessionModificationComplete) {
	smCtx.Debugf("Handle N1Sm PduSessionModificationComplete")
	state := smCtx.state.CurrentState()
	if state != SM_MODIFYING {
		smCtx.Errorf("PduSessionModificationComplete not handled in current state[%d]", state)
		smCtx.sendGsmStatus(req.GetPti(), nas.Cause5GSMMessageTypeNotCompatibleWithTheProtocolState)
		return
	}

	smCtx.Infof("Received N1Sm PduSessionModificationComplete")
	//TODO:  finalize/cancel the modification
	//then move back to SM_ACTIVE
	event := fsm.NewEmptyEventData(ModCmplEvent)
	smCtx.state.SetNextEvent(event)
}

func (smCtx *SmContext) buildPduSessionEstablishmentReject(pti, n1cause uint8) (pdu []byte, err error) {
	msg := &nas.PduSessionEstablishmentReject{
		GsmCause: n1cause,
	}
	pdu, err = smCtx.buildN1Sm(msg, pti)
	return
}

func (smCtx *SmContext) buildPduSessionEstablishmentAccept(pti uint8, cause *uint8) (pdu []byte, err error) {
	msg := &nas.PduSessionEstablishmentAccept{
		SelectedPduSessionType:               smCtx.sessionType,
		SelectedSscMode:                      1,
		SNssai:                               smCtx.snssai.NasType(),
		Dnn:                                  nas.NewDnn(smCtx.dnn),
		ExtendedProtocolConfigurationOptions: smCtx.pco,
	}

	smCtx.tunnel.FillN1SmPduSessionEstablishmentAccept(msg)

	ueIp := smCtx.ueIp.IP()
	if len(ueIp) > 0 {
		addr := ueIpBytes(ueIp, smCtx.sessionType)
		smCtx.Tracef("Set UE's IP address for PduSessionEstablishmentAccept: %v", addr)
		msg.PduAddress = nas.NewPduAddress(smCtx.sessionType, addr[:])
	}
	if cause != nil {
		msg.GsmCause = newUint8(*cause)
	}
	pdu, err = smCtx.buildN1Sm(msg, pti)
	return
}

func (smCtx *SmContext) buildPduSessionReleaseCommand(pti, n1cause uint8) (pdu []byte, err error) {
	msg := &nas.PduSessionReleaseCommand{
		GsmCause: n1cause,
	}
	pdu, err = smCtx.buildN1Sm(msg, pti)
	return
}

func (smCtx *SmContext) buildPduSessionReleaseReject(pti, n1cause uint8) (pdu []byte, err error) {
	msg := &nas.PduSessionReleaseReject{
		GsmCause: n1cause,
	}
	pdu, err = smCtx.buildN1Sm(msg, pti)
	return
}

func (smCtx *SmContext) buildPduSessionModificationReject(pti, n1cause uint8) (pdu []byte, err error) {
	msg := &nas.PduSessionModificationReject{
		GsmCause: n1cause,
	}
	pdu, err = smCtx.buildN1Sm(msg, pti)
	return
}

func (smCtx *SmContext) BuildPduSessionModificationCommand() (err error) {
	//msg *nas.PduSessionModificationCommand
	//TODO: dynamic filling with SmContext content
	//msg.SetQosRule()
	//msg.AuthorizedQosRules.SetLen()
	//msg.SessionAMBR.SetSessionAMBRForDownlink([2]uint8{0x11, 0x11})
	//msg.SessionAMBR.SetSessionAMBRForUplink([2]uint8{0x11, 0x11})
	//msg.SessionAMBR.SetUnitForSessionAMBRForDownlink(10)
	//msg.SessionAMBR.SetUnitForSessionAMBRForUplink(10)
	//msg.SessionAMBR.SetLen(uint8(len(msg.SessionAMBR.Octet)))
	return
}
