package sm

import (
	"bytes"
	"encoding/binary"

	//"etrib5gc/util/ngapconv"
	"fmt"

	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"

	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
)

const (
	UPCNXSTATE_DEACTIVATED uint8 = iota
	UPCNXSTATE_ACTIVATED
	UPCNXSTATE_ACTIVATING
	UPCNXSTATE_SUSPENDED
)

type N2SmInfo struct {
	infoType models.N2SmInfoType
	dat      []byte
}

func (smCtx *SmContext) isN2Deactivatable() bool {
	return smCtx.upCnxState == UPCNXSTATE_ACTIVATED || smCtx.upCnxState == UPCNXSTATE_ACTIVATING
}

func (smCtx *SmContext) sendN2SmInfo(n2SmInfo []byte, n2SmInfoType models.N2SmInfoType) error {
	n1n2 := &N1N2Messages{
		n2SmInfo:     n2SmInfo,
		n2SmInfoType: n2SmInfoType,
	}
	if err := smCtx.n1n2.sendN1N2(n1n2); err != nil {
		return utils.WrapError("Send N2SmInfo", err)
	}
	return nil
}

func (smCtx *SmContext) handleN2SmInfo(n2SmInfoType models.N2SmInfoType, n2SmInfo []byte) (err error) {
	smCtx.Debug("Handle N2SmInfo")
	switch n2SmInfoType {
	case models.N2SMINFOTYPE_PDU_RES_SETUP_RSP:
		err = smCtx.handlePduResourceSetupResponse(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_SETUP_FAIL:
		err = smCtx.handlePduResourceSetupFailure(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_NTY:
		err = smCtx.handlePduResourceModificationNotify(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_MOD_RSP:
		err = smCtx.handlePduResourceModifyResponse(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_MOD_FAIL:
		err = smCtx.handlePduResourceModifyFailure(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_MOD_IND:
		err = smCtx.handlePduResourceModifyIndication(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_NTY_REL:
		err = smCtx.handlePduResourceReleaseNotify(n2SmInfo)

	case models.N2SMINFOTYPE_PDU_RES_REL_RSP:
		err = smCtx.handlePduResourceReleaseResponse(n2SmInfo)

	default: //unknown N2SmInfoType (handover/pathswitch messages were handled separately)
		err = fmt.Errorf("Unknown N2SmInfo type")
	}
	return
}

// N2_STATE_ACTIVATING
func (smCtx *SmContext) handlePduResourceSetupFailure(n2SmInfo []byte) error {
	if smCtx.upCnxState != UPCNXSTATE_ACTIVATING {
		return fmt.Errorf("Invalid UpCnxState to handle PDUSessionResourceSetupUnsuccessfulTransfer")
	}

	//decode message
	msg := new(ies.PDUSessionResourceSetupUnsuccessfulTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PduSessionResourceSetupUnsuccessfulTransfer", err)
	}
	smCtx.Infof("Received N2SmInfo PduSessionResourceSetupUnsuccessfulTransfer")
	//TODO: release session now?
	return nil
}

// N2_STATE_ACTIVATING
func (smCtx *SmContext) handlePduResourceSetupResponse(n2SmInfo []byte) error {

	if smCtx.upCnxState != UPCNXSTATE_ACTIVATING {
		return fmt.Errorf("Invalid N2State to handle PDUSessionResourceSetupResponseTransfer")
	}

	msg := new(ies.PDUSessionResourceSetupResponseTransfer)
	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PDUSessionResourceSetupResponseTransfer", err)
	}

	qosflow := msg.DLQosFlowPerTNLInformation

	if qosflow.UPTransportLayerInformation.GTPTunnel == nil {
		return fmt.Errorf("No UPTransportLayerInformationPresentGTPTunnel in PDUSessionResourceSetupResponseTransfer")
	}

	smCtx.Infof("Received N2SmInfo PduSessionResourceSetupResponse")

	GTPTunnel := qosflow.UPTransportLayerInformation.GTPTunnel
	//update raninfo for the tunnel
	if err := smCtx.tunnel.UpdateRanInfo(GTPTunnel.TransportLayerAddress.Bytes, binary.BigEndian.Uint32(GTPTunnel.GTPTEID)); err != nil {
		smCtx.Errorf("Fail to activate UP: %+v", err)
		//TODO: may need to trigger release of the session
	}

	smCtx.upCnxState = UPCNXSTATE_ACTIVATED
	return nil
}

func (smCtx *SmContext) handlePduResourceReleaseResponse(n2SmInfo []byte) error {
	smCtx.Debugf("Handle N2SmInfo PduSessionResourceReleaseResponse")
	//decode message
	msg := new(ies.PDUSessionResourceReleaseResponseTransfer)

	if err := msg.Decode(aper.NewReader(bytes.NewBuffer(n2SmInfo))); err != nil {
		return utils.WrapError("Decode PDUSessionResourceReleaseResponseTransfer", err)
	}

	smCtx.Infof("Received N2SmInfo PDUSessionResourceReleaseResponseTransfer")
	smCtx.upCnxState = UPCNXSTATE_SUSPENDED
	//TODO: suspend the data paths here
	return nil
}

func (smCtx *SmContext) handlePduResourceModifyResponse(n2SmInfo []byte) error {
	smCtx.Debugf("Handle N2SmInfo PduResourceModifyResponse")
	//decode message
	msg := new(ies.PDUSessionResourceModifyResponseTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PDUSessionResourceModifyResponseTransfer", err)
	}

	smCtx.Infof("Received N2SmInfo PDUSessionResourceModifyResponseTransfer")

	if dlInfo := msg.DLNGUUPTNLInformation; dlInfo != nil {
		gtpTunnel := dlInfo.GTPTunnel
		if err := smCtx.tunnel.UpdateRanInfo(gtpTunnel.TransportLayerAddress.Bytes, binary.BigEndian.Uint32(gtpTunnel.GTPTEID)); err != nil {
			smCtx.Errorf("Fail to update downlink RAN information: %+v", err)
		}
	}
	okQfiList := []uint8{}
	failedQfiList := []uint8{}
	for _, item := range msg.QosFlowAddOrModifyResponseList {
		okQfiList = append(okQfiList, uint8(item.QosFlowIdentifier))
	}
	for _, item := range msg.QosFlowFailedToAddOrModifyList {
		failedQfiList = append(okQfiList, uint8(item.QosFlowIdentifier))
	}

	smCtx.tunnel.UpdateQosFlowStatus(okQfiList, failedQfiList)

	//TODO: handle the case if any error occurs (may need to release the
	//session?)
	return nil
}

func (smCtx *SmContext) handlePduResourceModifyFailure(n2SmInfo []byte) (err error) {
	smCtx.Infof("Handle N2SmInfo PduResourceModifyFailure")
	//decode message
	msg := new(ies.PDUSessionResourceModifyUnsuccessfulTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PDUSessionResourceModifyUnsuccessfulTransfer", err)
	}

	smCtx.Infof("Received N2SmInfo PDUSessionResourceModifyUnsuccessfulTransfer")

	//check if sesion in active state
	curstate := smCtx.state.CurrentState()
	if curstate != SM_MODIFYING {
		return fmt.Errorf("Session is not in modification procedure")
	}

	//TODO:intepret the cause then take neccesary actions

	return
}

func (smCtx *SmContext) handlePduResourceReleaseNotify(n2SmInfo []byte) (err error) {
	smCtx.Infof("Handle N2SmInfo PDUSessionResourceNotifyReleasedTransfer")

	//decode message
	msg := new(ies.PDUSessionResourceNotifyReleasedTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PDUSessionResourceNotifyReleasedTransfer", err)
	}
	smCtx.Infof("Received N2SmInfo PDUSessionResourceNotifyReleasedTransfer")

	//modify datapath to suspend gnB
	smCtx.suspendGnb()
	return
}

func (smCtx *SmContext) handlePduResourceModificationNotify(n2SmInfo []byte) (err error) {
	smCtx.Infof("Handle N2SmInfo PduResourceModificationNotify")

	//decode message
	msg := new(ies.PDUSessionResourceNotifyTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PDUSessionResourceNotifyTransfer", err)
	}

	smCtx.Infof("Received N2SmInfo PDUSessionResourceNotifyTransfer")

	curstate := smCtx.state.CurrentState()
	if curstate != SM_ACTIVE {
		return fmt.Errorf("Session is not active")
	}
	smCtx.modifySession(msg)
	return
}

func (smCtx *SmContext) handlePduResourceModifyIndication(n2SmInfo []byte) (err error) {
	smCtx.Debugf("Handle N2SmInfo PduResourceModifyIndication")

	//decode message
	msg := new(ies.PDUSessionResourceModifyIndicationTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PDUSessionResourceModifyIndicationTransfer", err)
	}

	smCtx.Infof("Received N2SmInfo PDUSessionResourceModifyIndicationTransfer")
	//trigger modification procedure(if need to send N1Sm)
	/*
		dat := common.NewEventData[ies.PDUSessionResourceModifyIndicationTransfer](MODIFY_RAN_INDICATE, msg)
		event := fsm.NewEventData[common.EventData](ModificationTriggerEvent, dat)
		smCtx.state.SetNextEvent(event)
	*/
	return
}

func (smCtx *SmContext) buildPduSessionResourceSetupRequestTransfer() (transfer []byte, err error) {

	msg := new(ies.PDUSessionResourceSetupRequestTransfer)

	// PDU Session Type
	//TODO: set according to the pdu session type in the smCtx
	msg.PDUSessionType = ies.PDUSessionType{
		Value: ies.PDUSessionTypeIpv4,
	}

	if smCtx.hoCtx != nil && smCtx.hoCtx.newTunnel != nil {
		smCtx.hoCtx.newTunnel.FillPduSessionResourceSetupRequest(msg)
	} else {
		smCtx.tunnel.FillPduSessionResourceSetupRequest(msg)
	}

	// Security Indication to NG-RAN (optional) TS 38.413 9.3.1.27 Only over
	// 3GPP access TS 23.501 5.10.3
	/*
		if ctx.AnType == models.AccessType__3_GPP_ACCESS && ctx.UpSecurity != nil {
			//TODO:

		}
	*/
	if transfer, err = msg.Encode(); err != nil {
		err = utils.WrapError("Encode PDUSessionResourceSetupRequestTransfer", err)
	}
	return
}

// TS 38.413 9.3.4.9
func (smCtx *SmContext) buildPathSwitchRequestAcknowledgeTransfer() (transfer []byte, err error) {

	// UL NG-U UP TNL Information(optional) TS 38.413 9.3.2.2
	msg := new(ies.PathSwitchRequestAcknowledgeTransfer)
	smCtx.tunnel.FillPathSwitchRequestAcknowledge(msg)

	// Received UP security policy mismatch from SMF locally stored TS 33.501
	// 6.6.1 Security Indication(optional) TS 38.413 9.3.1.27
	/*
			if !ctx.UpSecurityFromPathSwitchRequestSameAsLocalStored {

		//msg.SecurityIndication = upSecurityToNgap(models.UpSecurity from
		//DnnConfiguration from SmData)
			}
	*/
	if transfer, err = msg.Encode(); err != nil {
		err = utils.WrapError("Encode PathSwitchRequestAcknowledgeTransfer", err)
	}
	return
}

func (smCtx *SmContext) buildPathSwitchRequestUnsuccessfulTransfer(causePresent int, causeValue aper.Enumerated) (transfer []byte, err error) {
	msg := &ies.PathSwitchRequestUnsuccessfulTransfer{
		Cause: ies.Cause{
			Choice: uint64(causePresent),
		},
	}

	cause := msg.Cause

	switch uint64(causePresent) {
	case ies.CausePresentRadionetwork:
		cause.RadioNetwork = &ies.CauseRadioNetwork{
			Value: causeValue,
		}
	case ies.CausePresentTransport:
		cause.Transport = &ies.CauseTransport{
			Value: causeValue,
		}
	case ies.CausePresentNas:
		cause.Nas = &ies.CauseNas{
			Value: causeValue,
		}
	case ies.CausePresentProtocol:
		cause.Protocol = &ies.CauseProtocol{
			Value: causeValue,
		}
	case ies.CausePresentMisc:
		cause.Misc = &ies.CauseMisc{
			Value: causeValue,
		}
	}
	if transfer, err = msg.Encode(); err != nil {
		err = utils.WrapError("Encode PathSwitchRequestUnsuccessfulTransfer", err)
	}
	return
}

func (smCtx *SmContext) buildPduSessionResourceReleaseCommandTransfer() (transfer []byte, err error) {
	msg := &ies.PDUSessionResourceReleaseCommandTransfer{
		Cause: ies.Cause{
			Choice: ies.CausePresentNas,
			Nas: &ies.CauseNas{
				Value: ies.CauseNasNormalrelease,
			},
		},
	}
	if transfer, err = msg.Encode(); err != nil {
		err = utils.WrapError("Encode PDUSessionResourceReleaseCommandTransfer", err)
	}
	return
}

func (smCtx *SmContext) buildHandoverCommandTransfer(info *ies.UPTransportLayerInformation) (transfer []byte, err error) {
	msg := &ies.HandoverCommandTransfer{
		DLForwardingUPTNLInformation: info,
	}

	if transfer, err = msg.Encode(); err != nil {
		err = utils.WrapError("Encode HandoverCommandTransfer", err)
	}
	return
}

func (smCtx *SmContext) buildHandoverResourceAllocationUnsuccessfulTransfer(cause *ies.Cause) (transfer []byte, err error) {
	msg := &ies.HandoverResourceAllocationUnsuccessfulTransfer{}
	if cause != nil {
		msg.Cause = *cause
	} else {
		msg.Cause = ies.Cause{
			Choice: ies.CausePresentNas,
			Nas: &ies.CauseNas{
				Value: ies.CauseNasUnspecified,
			},
		}
	}

	if transfer, err = msg.Encode(); err != nil {
		err = utils.WrapError("Encode HandoverResourceAllocationUnsuccessfulTransfer", err)
	}
	return
}

func (smCtx *SmContext) suspendGnb() {
	if smCtx.tunnel != nil {
		if err := smCtx.tunnel.SuspendGnb(); err != nil {
			smCtx.Errorf("Fail to suspend Gnb: %+v", err)
		}
		smCtx.upCnxState = UPCNXSTATE_SUSPENDED
	}
}

func (smCtx *SmContext) modifySession(info *ies.PDUSessionResourceNotifyTransfer) {
	//TODO
}
