package ran

import (
	"etrib5gc/nfs/pran/ue"
	"github.com/lvdund/ngap/ies"
)

func (r *Ran) handleInitialUEMessage(msg *ies.InitialUEMessage) {
	r.Debug("Receive NGAP InitialUEMessage")

	ueCtx := ue.FindWithRemoteId(r.conn, msg.RANUENGAPID)

	if ueCtx != nil {
		ueCtx.Warnf("UeContext existed for a InitialUeMessage, remove it now")
		ueCtx.Kill()
	}

	if err := ue.CreateUeContext(r, msg); err != nil {
		r.Errorf("Fail to create new UeContext: %+v", err)
		cause := &ies.Cause{
			Choice: ies.CausePresentMisc,
			Misc: &ies.CauseMisc{
				Value: ies.CauseMiscUnspecified,
			},
		}
		r.sendErrorIndication(nil, &msg.RANUENGAPID, cause, nil)
	}
}

func (r *Ran) handleUplinkNasTransport(msg *ies.UplinkNASTransport) {
	r.Debug("Receive NGAP UplinkNasTransport")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP UplinkNasTransport", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UplinkNASTransport](ueCtx, ue.NAS_UPLINK, msg)
	}
}

func (r *Ran) handleErrorIndication(msg *ies.ErrorIndication) {
	r.Debug("Receive NGAP ErrorIndication")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)
	// TODO: handle error based on cause/criticalityDiagnostics
}

func (r *Ran) handleNasNonDeliveryIndication(msg *ies.NASNonDeliveryIndication) {
	r.Debug("Receive NGAP NasNonDeliveryIndication")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP NasNonDeliveryIndication", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.NASNonDeliveryIndication](ueCtx, ue.NAS_ERR, msg)
	}
}

func (r *Ran) handleInitialContextSetupResponse(msg *ies.InitialContextSetupResponse) {
	r.Debug("Receive NGAP InitialContextSetupResponse")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP InitialUeContextSetupResponse", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.InitialContextSetupResponse](ueCtx, ue.UECTX_SETUP_RSP, msg)
	}
}

func (r *Ran) handleInitialContextSetupFailure(msg *ies.InitialContextSetupFailure) {
	r.Debug("Receive NGAP InitialContextSetupFailure")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP InitialUeContextSetupFailure", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.InitialContextSetupFailure](ueCtx, ue.UECTX_SETUP_FAIL, msg)
	}
}
func (r *Ran) handleUEContextReleaseRequest(msg *ies.UEContextReleaseRequest) {
	r.Debug("Receive NGAP UeContextReleaseRequest")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UeContext not found to handle UeContextReleaseRequest")
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UEContextReleaseRequest](ueCtx, ue.UECTX_RELEASE_REQ, msg)
	}
}

func (r *Ran) handleUEContextReleaseComplete(msg *ies.UEContextReleaseComplete) {
	r.Debug("Receive NGAP UEContextReleaseComplete")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UeContext not found to handle InitialUeContextReleaseComplete")
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UEContextReleaseComplete](ueCtx, ue.UECTX_RELEASE_CMPL, msg)
	}
}

func (r *Ran) handleUEContextModificationResponse(msg *ies.UEContextModificationResponse) {
	r.Debug("Receive NGAP UeContextModificationResponse")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle UeContextModificationResponse", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UEContextModificationResponse](ueCtx, ue.UECTX_MODIFY_RSP, msg)
	}
}

func (r *Ran) handleUEContextModificationFailure(msg *ies.UEContextModificationFailure) {
	r.Debug("Receive NGAP UeContextModificationFailure")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP UeContextModificationFailure", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UEContextModificationFailure](ueCtx, ue.UECTX_MODIFY_FAIL, msg)
	}
}

func (r *Ran) handleRRCInactiveTransitionReport(msg *ies.RRCInactiveTransitionReport) {
	r.Debug("Receive NGAP RRCInactiveTransitionReport")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP RRCInactiveTransitionReport", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.RRCInactiveTransitionReport](ueCtx, ue.RRC_STATE_REPORT, msg)
	}
}

func (r *Ran) handleUERadioCapabilityInfoIndication(msg *ies.UERadioCapabilityInfoIndication) {
	r.Debug("Receive NGAP UERadioCapabilityInfoIndication")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP UERadioCapabilityInfoIndication", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UERadioCapabilityInfoIndication](ueCtx, ue.RADIO_CAP_IND, msg)
	}
}

func (r *Ran) handleUERadioCapabilityCheckResponse(msg *ies.UERadioCapabilityCheckResponse) {
	r.Debug("Receive NGAP UERadioCapabilityCheckResponse")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP UERadioCapabilityCheckResponse", msg.AMFUENGAPID)
	} else {
		//TODO: do nothing for now
	}
}

func (r *Ran) handlePDUSessionResourceSetupResponse(msg *ies.PDUSessionResourceSetupResponse) {
	r.Debug("Receive NGAP PDUSessionResourceSetupResponse")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP PDUSessionResourceSetupResponse", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.PDUSessionResourceSetupResponse](ueCtx, ue.PDU_SET_RES, msg)
	}
}

func (r *Ran) handlePDUSessionResourceReleaseResponse(msg *ies.PDUSessionResourceReleaseResponse) {
	r.Debug("Receive NGAP PDUSessionResourceReleaseResponse")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP PDUSessionResourceReleaseResponse", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.PDUSessionResourceReleaseResponse](ueCtx, ue.PDU_REL_RES, msg)
	}
}

func (r *Ran) handlePDUSessionResourceModifyResponse(msg *ies.PDUSessionResourceModifyResponse) {
	r.Debug("Receive NGAP PDUSessionResourceModifyResponse")
	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP PDUSessionResourceModifyResponse", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.PDUSessionResourceModifyResponse](ueCtx, ue.PDU_MOD_RES, msg)
	}
}

// TS139.413-V15.3.0 8.2.4
func (r *Ran) handlePDUSessionResourceNotify(msg *ies.PDUSessionResourceNotify) {
	r.Debug("Receive NGAP PDUSessionResourceNotify")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP PDUSessionResourceNotify", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.PDUSessionResourceNotify](ueCtx, ue.PDU_NOTIFY, msg)
	}
}

// TS139.413-V15.3.0 8.2.5
func (r *Ran) handlePDUSessionResourceModifyIndication(msg *ies.PDUSessionResourceModifyIndication) {
	r.Debug("Receive NGAP PDUSessionResourceModifyIndication")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UeContext not found [AmfUeNgapId=%d] to handle NGAP PDUSessionResourceModifyIndication", msg.AMFUENGAPID)
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.PDUSessionResourceModifyIndication](ueCtx, ue.PDU_MOD_IND, msg)
	}
}

/****** INTERFACE MANAGEMENT HANDLERS ******/

//NOTE: some NGAP messages are deprecated in cloud-native core as AMF and gnB
//no longer needs a strong bonding.

func (r *Ran) handleNGSetupRequest(msg *ies.NGSetupRequest) {
	r.Debugf("Receive NGAP NGSetupRequest")

	if cause := r.updateRanInfo(&msg.GlobalRANNodeID, msg.RANNodeName, &msg.DefaultPagingDRX, msg.SupportedTAList); cause == nil {
		_pool.add(r)
		r.Infof("GnB context is added, send NGSetupResponse to gnB")
		r.sendNGSetupResponse()
	} else {
		r.sendNGSetupFailure(*cause)
	}
}

func (r *Ran) handleNGResetAcknowledge(msg *ies.NGResetAcknowledge) {
	r.Debug("Receive NGAP NGResetAcknowledge")

	handleCriticalityDiagnostics(msg.CriticalityDiagnostics)
	if list := msg.UEassociatedLogicalNGconnectionList; len(list) > 0 {
		for i, item := range list {
			if item.AMFUENGAPID != nil && item.RANUENGAPID != nil {
				r.Tracef("%d: AmfUeNgapID[%d] UeContextNgapID[%d]", i+1, item.AMFUENGAPID, item.RANUENGAPID)
			} else if item.AMFUENGAPID != nil {
				r.Tracef("%d: AmfUeNgapID[%d] UeContextNgapID[-1]", i+1, item.AMFUENGAPID)
			} else if item.RANUENGAPID != nil {
				r.Tracef("%d: AmfUeNgapID[-1] UeContextNgapID[%d]", i+1, item.RANUENGAPID)
			}
		}
	}

}

func (r *Ran) handleNGReset(msg *ies.NGReset) {
	r.Debug("Receive NGAP NGReset")

	switch msg.ResetType.Choice {
	case ies.ResetTypePresentNgInterface:
		r.Info("Reset Ran Context due to NGReset request from gnB")
		r.removeUes()
		r.sendNGResetAcknowledge(nil, nil)
	case ies.ResetTypePresentPartofngInterface:
		r.Info("Remove UEs Ran from Context due to NGReset request from gnB")
		ueList := msg.ResetType.PartOfNGInterface
		if ueList == nil {
			r.Error("PartOfNGInterface is nil")
			return
		}

		for _, ueItem := range ueList {
			r.removeUe(ueItem.RANUENGAPID, ueItem.AMFUENGAPID)
		}
		r.sendNGResetAcknowledge(ueList, nil)
	default:
		r.Warnf("Invalid ResetType[%d]", msg.ResetType.Choice)
	}
}

func (r *Ran) handleRanConfigurationUpdate(msg *ies.RANConfigurationUpdate) {
	r.Debug("Receive NGAP RanConfigurationUpdate")

	if cause := r.updateRanInfo(msg.GlobalRANNodeID, msg.RANNodeName, msg.DefaultPagingDRX, msg.SupportedTAList); cause == nil {
		r.sendRanConfigurationUpdateAcknowledge(nil)
	} else {
		r.sendRanConfigurationUpdateFailure(*cause, nil)
	}
}

// deprecated
func (r *Ran) handleAMFconfigurationUpdateAcknowledge(ack *ies.AMFConfigurationUpdateAcknowledge) {
	r.Debug("Receive NGAP AMFConfigurationUpdateAcknowledge")
}

// deprecated
func (r *Ran) handleAMFconfigurationUpdateFailure(failure *ies.AMFConfigurationUpdateFailure) {
	r.Warnf("Amf configuration update failure is ignored")
}

/************ CONFIGURATION TRANSFER HANDLERS ************/

func (r *Ran) handleUplinkRanConfigurationTransfer(uplinkRANConfigurationTransfer *ies.UplinkRANConfigurationTransfer) {
	r.Debug("Receive NGAP UplinkRanConfigurationTransfer")
}

func (r *Ran) sendDownlinkRanConfigurationTransfer(transfer *ies.SONConfigurationTransfer) (err error) {
	r.Warnf("Send NGAP DownlinkRanConfigurationTransfer not implemented")
	return
}

/*********************** HANDOVER HANDLERS ******************/

// gnB requests handover preparation
func (r *Ran) handleHandoverRequired(msg *ies.HandoverRequired) {
	r.Debug("Receive NGAP HandoverRequired")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Info("Receive NGAP HandoverRequired")
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.HandoverRequired](ueCtx, ue.HO_REQUIRED, msg)
	}
}

// target gnB acknowledges handover request
func (r *Ran) handleHandoverRequestAcknowledge(msg *ies.HandoverRequestAcknowledge) {
	r.Debug("Receive NGAP HandoverRequestAcknowledge")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP HandoverRequestAcknowledge", msg.AMFUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.HandoverRequestAcknowledge](ueCtx, ue.HO_REQ_ACK, msg)
	}
}

// target gnB fail to allocate resource for handover
func (r *Ran) handleHandoverFailure(msg *ies.HandoverFailure) {
	r.Debug("Receive NGAP HandoverFailure")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Info("Receive NGAP HandoverFailure")
		r.sendUeNotFoundError(&msg.AMFUENGAPID, nil)
	} else {
		ue.ReceiveNgapMessage[ies.HandoverFailure](ueCtx, ue.HO_FAILURE, msg)
	}
}

// target gnB notify handover complete
func (r *Ran) handleHandoverNotify(msg *ies.HandoverNotify) {
	r.Debug("Receive NGAP HandoverNotify")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP HandoverNotify", msg.AMFUENGAPID)
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.HandoverNotify](ueCtx, ue.HO_NOTIFY, msg)
	}
}

// source gnB cancel an on-going handover
func (r *Ran) handleHandoverCancel(msg *ies.HandoverCancel) {
	r.Debug("Receive NGAP HandoverCancel")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP HandoverCancel", msg.AMFUENGAPID)
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.HandoverCancel](ueCtx, ue.HO_CANCEL, msg)
	}
}

func (r *Ran) handleUplinkRanStatusTransfer(msg *ies.UplinkRANStatusTransfer) {
	r.Debug("Receive NGAP UplinkRanStatusTransfer")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP UplinkRanStatusTransfer", msg.AMFUENGAPID)
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		ue.ReceiveNgapMessage[ies.UplinkRANStatusTransfer](ueCtx, ue.HO_UL_RAN_STATUS, msg)
	}
}

// TS 23.502 4.9.1
func (r *Ran) handlePathSwitchRequest(msg *ies.PathSwitchRequest) {
	r.Debug("Receive NGAP PathSwitchRequest")
	//look for UeContext using AmfUeNgapId
	if ueCtx := ue.FindWithLocalId(msg.SourceAMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP PathSwitchRequest", msg.SourceAMFUENGAPID)
		r.sendPathSwitchRequestFailure(msg.SourceAMFUENGAPID, msg.RANUENGAPID, nil, nil)
	} else {
		//update ran and ranUeNgapId (switched Ran)
		ueCtx.UpdateRanInfo(r, msg.RANUENGAPID)
		ue.ReceiveNgapMessage[ies.PathSwitchRequest](ueCtx, ue.PATHSWITCH_REQ, msg)
	}
}

/*************************** OTHERS ********************/

func (r *Ran) handleLocationReportingFailureIndication(msg *ies.LocationReportingFailureIndication) {
	r.Debug("Receive NGAP LocationReportingFailureIndication")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP LocationReportingFailureIndication", msg.AMFUENGAPID)
	} else {
		//TODO:
	}
}

func (r *Ran) handleLocationReport(msg *ies.LocationReport) {
	r.Debug("Receive NGAP LocationReport")

	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP LocationReport", msg.AMFUENGAPID)
	} else {
		//TODO:
	}
}
func (r *Ran) handleCellTrafficTrace(msg *ies.CellTrafficTrace) {
	r.Debug("Receive NGAP CallTrafficTrace")
	if ueCtx := ue.FindWithLocalId(msg.AMFUENGAPID); ueCtx == nil {
		r.Warnf("UE Context not found [AmfUeNgapId=%d] to handle NGAP CellTrafficTrace", msg.AMFUENGAPID)
		r.sendUeNotFoundError(&msg.AMFUENGAPID, &msg.RANUENGAPID)
	} else {
		// TODO: TS 32.422 4.2.2.10
		// When AMF receives this new NG signalling message containing the Trace Recording Session Reference (TRSR)
		// and Trace Reference (TR), the AMF shall look up the SUPI/IMEI(SV) of the given call from its database and
		// shall send the SUPI/IMEI(SV) numbers together with the Trace Recording Session Reference and Trace Reference
		// to the Trace Collection Entity.

	}

}

func (r *Ran) handleUplinkUEAssociatedNRPPATransport(uplinkUEAssociatedNRPPaTransport *ies.UplinkUEAssociatedNRPPaTransport) {
	r.Debug("Receive NGAP UplinkUEAssociatedRPPATransport")
	// TODO: Forward NRPPaPDU to LMF
}

func (r *Ran) handleUplinkNonUEAssociatedNRPPATransport(uplinkNonUEAssociatedNRPPATransport *ies.UplinkNonUEAssociatedNRPPaTransport) {
	r.Debug("Receive NGAP UplinkNonUEAssociatedRPPATransport")
	// TODO: Forward NRPPaPDU to LMF
}
