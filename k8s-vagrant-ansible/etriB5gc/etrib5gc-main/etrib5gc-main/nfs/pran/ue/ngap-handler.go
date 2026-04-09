package ue

import (
	"context"
	"etrib5gc/internal/eventmux"
	"etrib5gc/mesh"
	pranctx "etrib5gc/nfs/pran/context"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/apis/amf/handover"
	"github.com/reogac/sbi/apis/amf/n2nas"
	"github.com/reogac/sbi/models"
)

type LongEvent interface {
	Wait() chan error
	Finalize(error)
}

type PtrTo[T any] interface{ ~*T }

func ReceiveNgapMessage[T any, PT PtrTo[T]](ueCtx *UeContext, evType uint8, dat PT) {
	evData := eventmux.NewEventData[T](evType, dat)
	ev := eventmux.NewEventData(NgapEvent, evData)
	if err := _exec.Send(context.Background(), ueCtx.execSlot, ev); err != nil {
		ueCtx.Errorf("Fail to send event: %+v", err)
	}
}

func (ueCtx *UeContext) handleNgapEvent(ctx context.Context, e *eventmux.EventData) {
	if ueCtx.amfCli == nil {
		ueCtx.Warnf("No AMF to handle NAS message")
		return
	}

	switch e.Type() {
	case NAS_UPLINK:
		msg := eventmux.GetEventData[ies.UplinkNASTransport](e)
		ueCtx.handleNasUplink(msg)

	case NAS_ERR:
		msg := eventmux.GetEventData[ies.NASNonDeliveryIndication](e)
		ueCtx.handleUplinkNasError(msg)

	case UECTX_SETUP_RSP:
		msg := eventmux.GetEventData[ies.InitialContextSetupResponse](e)
		ueCtx.handleUeContextSetupResponse(msg)

	case UECTX_SETUP_FAIL:
		msg := eventmux.GetEventData[ies.InitialContextSetupFailure](e)
		ueCtx.handleUeContextSetupFailure(msg)

	case UECTX_MODIFY_RSP:
		msg := eventmux.GetEventData[ies.UEContextModificationResponse](e)
		ueCtx.handleUeContextModifyResponse(msg)

	case UECTX_MODIFY_FAIL:
		msg := eventmux.GetEventData[ies.UEContextModificationFailure](e)
		ueCtx.handleUeContextModifyFailure(msg)

	case UECTX_RELEASE_REQ:
		msg := eventmux.GetEventData[ies.UEContextReleaseRequest](e)
		ueCtx.handleUeContextReleaseRequest(msg)

	case UECTX_RELEASE_CMPL:
		msg := eventmux.GetEventData[ies.UEContextReleaseComplete](e)
		ueCtx.handleUeContextReleaseComplete(msg)

	case PDU_REL_RES:
		msg := eventmux.GetEventData[ies.PDUSessionResourceReleaseResponse](e)
		ueCtx.handlePduSessionResourceReleaseResponse(msg)

	case PDU_SET_RES:
		msg := eventmux.GetEventData[ies.PDUSessionResourceSetupResponse](e)
		ueCtx.handlePduSessionResourceSetupResponse(msg)

	case PDU_MOD_RES:
		msg := eventmux.GetEventData[ies.PDUSessionResourceModifyResponse](e)
		ueCtx.handlePduSessionResourceModifyResponse(msg)

	case PDU_NOTIFY:
		msg := eventmux.GetEventData[ies.PDUSessionResourceNotify](e)
		ueCtx.handlePduSessionResourceNotify(msg)

	case PDU_MOD_IND:
		msg := eventmux.GetEventData[ies.PDUSessionResourceModifyIndication](e)
		ueCtx.handlePduSessionResourceModifyIndication(msg)

	case RRC_STATE_REPORT:
		msg := eventmux.GetEventData[ies.RRCInactiveTransitionReport](e)
		ueCtx.handleRrcInactiveTransitionReport(msg)

	case RADIO_CAP_IND:
		msg := eventmux.GetEventData[ies.UERadioCapabilityInfoIndication](e)
		ueCtx.handleUeRadioCapabilityInfoIndication(msg)

	case HO_REQUIRED:
		msg := eventmux.GetEventData[ies.HandoverRequired](e)
		ueCtx.handleHandoverRequired(msg)

	case HO_REQ_ACK:
		msg := eventmux.GetEventData[ies.HandoverRequestAcknowledge](e)
		ueCtx.handleHandoverRequestAcknowledge(msg)

	case HO_FAILURE:
		msg := eventmux.GetEventData[ies.HandoverFailure](e)
		ueCtx.handleHandoverFailure(msg)

	case HO_NOTIFY:
		msg := eventmux.GetEventData[ies.HandoverNotify](e)
		ueCtx.handleHandoverNotify(msg)

	case HO_CANCEL:
		msg := eventmux.GetEventData[ies.HandoverCancel](e)
		ueCtx.handleHandoverCancel(msg)

	case HO_UL_RAN_STATUS:
		msg := eventmux.GetEventData[ies.UplinkRANStatusTransfer](e)
		ueCtx.handleUplinkRanStatusTransfer(msg)

	case PATHSWITCH_REQ:
		msg := eventmux.GetEventData[ies.PathSwitchRequest](e)
		ueCtx.handlePathSwitchRequest(msg)
	}
}
func (ueCtx *UeContext) handleRrcInactiveTransitionReport(msg *ies.RRCInactiveTransitionReport) {
	ueCtx.Warnf("NGAP RRCInactiveTransitionReport not handled")
	//TODO:
	//ueCtx.updateLocationInformation(msg.UserLocationInformation)
	//ueCtx.updateRrcState(msg.RRCState)
}

func (ueCtx *UeContext) handleUeRadioCapabilityInfoIndication(msg *ies.UERadioCapabilityInfoIndication) {
	ueCtx.Warnf("NGAP UERadioCapabilityInfoIndication not handled")
	/* TODO: handle the mesage
	r.Info("Handle UE Radio Capability Info Indication")
		var dat models.RadioCapabilityInfoIndication
		if radioCap != nil {
			dat.RadioCap = hex.EncodeToString(radioCap.Value)
		}
		if radioCap4Paging != nil {
			if radioCap4Paging.UERadioCapabilityForPagingOfNR != nil {
				dat.RadioCap4PagingNr = hex.EncodeToString(radioCap4Paging.UERadioCapabilityForPagingOfNR.Value)
			}
			if radioCap4Paging.UERadioCapabilityForPagingOfEUTRA != nil {
				dat.RadioCap4PagingEutra = hex.EncodeToString(radioCap4Paging.UERadioCapabilityForPagingOfEUTRA.Value)
			}
		}
	// TS 38.413 8.14.1.2/TS 23.502 4.2.8a step5/TS 23.501, clause 5.4.4.1.
	// send its most up to date UE Radio Capability information to the RAN in the N2 REQUEST message.
	*/
}

/******** NAS *********/
func (ueCtx *UeContext) handleNasUplink(msg *ies.UplinkNASTransport) {
	if err := n2nas.NasUl(ueCtx.amfCli, ueCtx.amfUeId, &models.NasUplinkTransport{
		NasPdu: msg.NASPDU,
	}); err != nil {
		ueCtx.Errorf("Fail to forward NasUl to core: %+v", err)
	} else {
		ueCtx.Infof("NasUl from gnB is forwarded to core")
	}
}

func (ueCtx *UeContext) handleUplinkNasError(msg *ies.NASNonDeliveryIndication) {
	causePresent, causeValue := causeConvert(&msg.Cause)
	sbiMsg := &models.UplinkNasError{
		NasPdu: msg.NASPDU,
		Cause: models.N2Cause{
			CausePresent: int16(causePresent),
			CauseValue:   int16(causeValue),
		},
	}

	if err := n2nas.NasErr(ueCtx.amfCli, ueCtx.amfUeId, sbiMsg); err != nil {
		ueCtx.Errorf("Fail to forward Nas non delicery indication to core: %+v", err)
	} else {
		ueCtx.Infof("Nas non-delivery indication from gnB is forward to core")
	}
}

/******** UeContext management handlers*********/

func (ueCtx *UeContext) handleUeContextSetupResponse(msg *ies.InitialContextSetupResponse) {
	ueCtx.Infof("Receive an UeContextSetupResponse")

	if ueCtx.setupJob == nil {
		ueCtx.Warnf("Receive an unsolicited UeContextSetupResponse")
		return
	}
	var transferList []models.N2SmInfoUplinkContent

	for _, s := range msg.PDUSessionResourceSetupListCxtRes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceSetupResponseTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_SETUP_RSP,
		})
	}

	for _, s := range msg.PDUSessionResourceFailedToSetupListCxtRes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceSetupUnsuccessfulTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_SETUP_FAIL,
		})
	}

	info := ueCtx.setupJob
	info.Response = &models.UeContextSetupResponse{
		Transfers: transferList,
	}
	ueCtx.setupJob.Finalize(nil)
}

func (ueCtx *UeContext) handleUeContextSetupFailure(msg *ies.InitialContextSetupFailure) {
	if ueCtx.setupJob == nil {
		ueCtx.Warnf("Receive an unsolicited UeContextSetupFailure")
		return
	}
	//if RAN fail to init the UeContext, we should release it
	var sessionList []int16

	if list := msg.PDUSessionResourceFailedToSetupListCxtFail; len(list) > 0 {
		sessionList = make([]int16, len(list))
		for i, s := range list {
			sessionList[i] = int16(s.PDUSessionID)
		}
	}

	sbiMsg := &models.UeContextSetupFailure{
		Cause:      n2Cause(&msg.Cause),
		FailedList: sessionList,
	}

	info := ueCtx.setupJob
	info.Failure = sbiMsg
	ueCtx.setupJob.Finalize(nil)
}

func (ueCtx *UeContext) handleUeContextModifyResponse(msg *ies.UEContextModificationResponse) {
	if ueCtx.modifyJob == nil {
		ueCtx.Warnf("Receive an unsolicited UeContextModifyResponse")
		return
	}

	sbiMsg := &models.UeContextModifyResponse{
		// TODO: fill the message
	}

	info := ueCtx.modifyJob
	info.Response = sbiMsg
	ueCtx.modifyJob.Finalize(nil)
}

func (ueCtx *UeContext) handleUeContextModifyFailure(msg *ies.UEContextModificationFailure) {
	if ueCtx.modifyJob == nil {
		ueCtx.Warnf("Receive an unsolicited UeContextModifyFailure")
		return
	}

	//causePresent, causeValue := causeConvert(msg.Cause)

	sbiMsg := &models.UeContextModifyFailure{
		//CausePresent: int16(causePresent),
		//CauseValue:   int16(causeValue),
	}

	info := ueCtx.modifyJob
	info.Failure = sbiMsg
	ueCtx.modifyJob.Finalize(nil)
}

func (ueCtx *UeContext) handleUeContextReleaseRequest(msg *ies.UEContextReleaseRequest) {
	if ueCtx.releaseCtx != nil {
		//release on-going// nothing todo
		ueCtx.Warnf("Receive Ngap UeContextReleaseRequest while release is on-going [ignored]")
	} else {
		var sessionList []int16
		if list := msg.PDUSessionResourceListCxtRelReq; len(list) > 0 {
			sessionList = make([]int16, len(list))
			for i, s := range list {
				sessionList[i] = int16(s.PDUSessionID)
			}

		}

		sbiMsg := &models.UeContextReleaseRequest{
			Cause:       n2Cause(&msg.Cause),
			SessionList: sessionList,
		}
		//forward to AMF
		ueCtx.requestUeContextRelease(context.TODO(), sbiMsg, nil)
	}
}

func (ueCtx *UeContext) handleUeContextReleaseComplete(msg *ies.UEContextReleaseComplete) {
	ueCtx.Infof("Receive UeContextReleaseComplete from gnB")
	if relCtx := ueCtx.releaseCtx; relCtx.job != nil {
		ueCtx.Debug("Stop gnbTimer and response to AMF")
		relCtx.gnbTimer.Stop()

		var sessionList []int16

		if list := msg.PDUSessionResourceListCxtRelCpl; len(list) > 0 {
			sessionList = make([]int16, len(list))
			for i, s := range list {
				sessionList[i] = int16(s.PDUSessionID)
			}
		}

		info := relCtx.job
		info.Complete = &models.UeContextReleaseComplete{
			SessionList: sessionList,
		}
		info.Finalize(nil)
	} else if relCtx.gnbTimer != nil {
		ueCtx.Infof("Stop gnbTimer and clear context")
		relCtx.gnbTimer.Stop()
		ueCtx.clean() //clean local UeContext
	} else {
		ueCtx.Warnf("UeContextReleaseComplete is unsolicited")
	}
}

/******** Pdu session resource management handlers*********/

func (ueCtx *UeContext) handlePduSessionResourceSetupResponse(msg *ies.PDUSessionResourceSetupResponse) {
	var transferList []models.N2SmInfoUplinkContent

	for _, s := range msg.PDUSessionResourceSetupListSURes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceSetupResponseTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_SETUP_RSP,
		})
	}

	for _, s := range msg.PDUSessionResourceFailedToSetupListSURes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceSetupUnsuccessfulTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_SETUP_FAIL,
		})
	}

	if len(transferList) > 0 {
		ueCtx.sendN2SmInfoUplink(&models.N2SmInfoUplinkTransport{
			Transfers: transferList,
		})
	}
}
func (ueCtx *UeContext) handlePduSessionResourceReleaseResponse(msg *ies.PDUSessionResourceReleaseResponse) {

	var transferList []models.N2SmInfoUplinkContent
	for _, s := range msg.PDUSessionResourceReleasedListRelRes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceReleaseResponseTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_REL_RSP,
		})
	}
	ueCtx.sendN2SmInfoUplink(&models.N2SmInfoUplinkTransport{
		Transfers: transferList,
	})
}

func (ueCtx *UeContext) handlePduSessionResourceModifyResponse(msg *ies.PDUSessionResourceModifyResponse) {
	var transferList []models.N2SmInfoUplinkContent

	for _, s := range msg.PDUSessionResourceModifyListModRes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceModifyResponseTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_MOD_RSP,
		})
	}

	for _, s := range msg.PDUSessionResourceFailedToModifyListModRes {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceModifyUnsuccessfulTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_MOD_FAIL,
		})
	}
	if len(transferList) > 0 {
		ueCtx.sendN2SmInfoUplink(&models.N2SmInfoUplinkTransport{
			Transfers: transferList,
		})
	}
}

func (ueCtx *UeContext) handlePduSessionResourceNotify(msg *ies.PDUSessionResourceNotify) {
	var transferList []models.N2SmInfoUplinkContent

	for _, s := range msg.PDUSessionResourceNotifyList {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceNotifyTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_NTY,
		})
	}

	for _, s := range msg.PDUSessionResourceReleasedListNot {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceNotifyReleasedTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_NTY_REL,
		})
	}
	if len(transferList) > 0 {
		ueCtx.sendN2SmInfoUplink(&models.N2SmInfoUplinkTransport{
			Transfers: transferList,
		})
	}
}
func (ueCtx *UeContext) handlePduSessionResourceModifyIndication(msg *ies.PDUSessionResourceModifyIndication) {
	var transferList []models.N2SmInfoUplinkContent

	for _, s := range msg.PDUSessionResourceModifyListModInd {
		transferList = append(transferList, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PDUSessionResourceModifyIndicationTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PDU_RES_MOD_IND,
		})
	}

	ueCtx.sendN2SmInfoUplink(&models.N2SmInfoUplinkTransport{
		Transfers: transferList,
	})
}

/**************HANDOVER**************************/

func (ueCtx *UeContext) handleHandoverRequired(msg *ies.HandoverRequired) {
	var err error
	targetId, err := convertTargetId(&msg.TargetID)
	if err != nil {
		ueCtx.Errorf("Fail to parse NGAP target Id: %+v", err)
		ueCtx.sendHandoverPreparationFailure(nil)
		return
	}
	var sessions []models.N2SmInfoUplinkContent

	for _, s := range msg.PDUSessionResourceListHORqd {
		sessions = append(sessions, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.HandoverRequiredTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_HANDOVER_REQUIRED,
		})
	}
	//convert ngap message to sbi message
	sbiMsg := &models.HandoverRequired{
		HandoverType:          int16(msg.HandoverType.Value),
		TargetId:              *targetId,
		SourceToTargetContent: msg.SourceToTargetTransparentContainer,
		Cause:                 n2Cause(&msg.Cause),
		Sessions:              sessions,
	}
	if msg.DirectForwardingPathAvailability != nil {
		flag := msg.DirectForwardingPathAvailability.Value == ies.DirectForwardingPathAvailabilityDirectpathavailable
		sbiMsg.DirectFwdPathFlag = newBool(flag)
	}

	ueCtx.Infof("Send HandoverRequired to AMF")

	if rsp, ersp, err := handover.HandoverRequired(ueCtx.amfCli, ueCtx.amfUeId, sbiMsg); err != nil {
		ueCtx.Errorf("Fail to send HandoverRequired to AMF: %+v", err)
		ueCtx.sendHandoverPreparationFailure(nil)
	} else if ersp != nil {
		ueCtx.sendHandoverPreparationFailure(ersp)
	} else {
		ueCtx.sendHandoverCommand(rsp)
	}
}

func (ueCtx *UeContext) handleHandoverRequestAcknowledge(msg *ies.HandoverRequestAcknowledge) {
	if ueCtx.handoverJob == nil {
		ueCtx.Warnf("HandoverRequestAcknowledge is unsolicited")
		return
	}

	//create sbiMessage
	var sessions []models.N2SmInfoUplinkContent
	for _, s := range msg.PDUSessionResourceAdmittedList {
		sessions = append(sessions, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.HandoverRequestAcknowledgeTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_HANDOVER_REQ_ACK,
		})
	}
	for _, s := range msg.PDUSessionResourceFailedToSetupListHOAck {
		sessions = append(sessions, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.HandoverResourceAllocationUnsuccessfulTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_HANDOVER_RES_ALLOC_FAIL,
		})
	}
	sbiMsg := &models.HandoverRequestAcknowledge{
		Sessions:              sessions,
		TargetToSourceContent: msg.TargetToSourceTransparentContainer,
		RanUeId:               ueCtx.localId,
		RanNets:               ueCtx.ran.RanNets(),
		NfSelection:           pranctx.GetNfSelection(),
		NasSplit:              pranctx.NasSplit(),
	}
	//update RanUeNgapId for UeContext
	ueCtx.ranNgapId = msg.RANUENGAPID
	//then finalized pending handover request context
	info := ueCtx.handoverJob
	info.Response = sbiMsg

	ueCtx.handoverJob.Finalize(nil)
}

func (ueCtx *UeContext) handleHandoverFailure(msg *ies.HandoverFailure) {
	if ueCtx.handoverJob == nil {
		ueCtx.Warnf("HandoverFailure is unsolicited")
		return
	}

	info := ueCtx.handoverJob
	info.ErrResponse = &models.HandoverRequestFailure{
		Cause: n2Cause(&msg.Cause),
	}

	ueCtx.handoverJob.Finalize(nil)
}

func (ueCtx *UeContext) handleHandoverNotify(msg *ies.HandoverNotify) {
	var err error
	var loc *models.UserLocation
	if loc, err = locConvert(&msg.UserLocationInformation); err != nil {
		return
	}
	sbiMsg := &models.HandoverNotify{
		Loc: *loc,
	}
	ueCtx.Infof("Send HandoverNotify to AMF")
	if err := handover.HandoverNotify(ueCtx.amfCli, ueCtx.amfUeId, sbiMsg); err != nil {
		ueCtx.Errorf("Fail to send HandoverNotify to AMF: %+v", err)
		//TODO: send error indication to let gnb know
	}
}

func (ueCtx *UeContext) handleHandoverCancel(msg *ies.HandoverCancel) {
	sbiMsg := &models.HandoverCancel{
		Cause: n2Cause(&msg.Cause),
	}
	ueCtx.Infof("Send HandoverCancel to AMF")
	if rsp, err := handover.HandoverCancel(ueCtx.amfCli, ueCtx.amfUeId, sbiMsg); err != nil {
		ueCtx.Errorf("Fail to send HandoverCancel:%+v", err)
	} else {
		ueCtx.sendHandoverCancelAcknowledge(rsp)
	}
	//clear and remove UeContext
	ueCtx.clean()
}

func (ueCtx *UeContext) handleUplinkRanStatusTransfer(msg *ies.UplinkRANStatusTransfer) {
	ueCtx.Warnf("UplinkRanStatusTransfer not handled")
}

func (ueCtx *UeContext) handlePathSwitchRequest(msg *ies.PathSwitchRequest) {
	var err error
	var loc *models.UserLocation
	if loc, err = locConvert(&msg.UserLocationInformation); err != nil {
		ueCtx.Errorf("Fail to parse UserLocationInformation: %+v", err)
		return
	}

	ueCtx.Infof("Send PathSwitchRequest to AMF")
	sbiMsg := &models.PathSwitchRequest{
		Loc: *loc,
	}
	for _, s := range msg.PDUSessionResourceToBeSwitchedDLList {
		sbiMsg.Sessions = append(sbiMsg.Sessions, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PathSwitchRequestTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PATH_SWITCH_REQ,
		})
	}
	for _, s := range msg.PDUSessionResourceFailedToSetupListPSReq {
		sbiMsg.Sessions = append(sbiMsg.Sessions, models.N2SmInfoUplinkContent{
			SessionId:    int16(s.PDUSessionID),
			N2SmInfo:     s.PathSwitchRequestSetupFailedTransfer,
			N2SmInfoType: models.N2SMINFOTYPE_PATH_SWITCH_SETUP_FAIL,
		})
	}
	//send to AMF
	if rsp, ersp, err := handover.PathSwitch(ueCtx.amfCli, handover.PathSwitchParams{
		Callback: mesh.EndpointInfo(),
		UeId:     ueCtx.amfUeId,
	}, sbiMsg); err != nil {
		ueCtx.Errorf("Fail to send PathSwitchRequest to AMF:%+v", err)
		ueCtx.sendPathSwitchFailure(nil)
	} else if ersp != nil {
		ueCtx.sendPathSwitchFailure(ersp)
		//NOTE: release UeContext?
	} else if rsp != nil {
		ueCtx.sendPathSwitchAcknowledge(rsp)
	}
}
