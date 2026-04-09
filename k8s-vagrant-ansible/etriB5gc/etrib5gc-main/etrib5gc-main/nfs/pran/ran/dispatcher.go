package ran

import (
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/ies"
)

func (r *Ran) HandleInitiatingMsg(iMsg *ngap.NgapMessage) {
	switch iMsg.ProcedureCode.Value {
	case ies.ProcedureCode_NGSetup:
		if content, ok := iMsg.Msg.(*ies.NGSetupRequest); ok {
			r.handleNGSetupRequest(content)
			return
		}
	case ies.ProcedureCode_InitialUEMessage:
		if content, ok := iMsg.Msg.(*ies.InitialUEMessage); ok {
			r.handleInitialUEMessage(content)
			return
		}
	case ies.ProcedureCode_UplinkNASTransport:
		if content, ok := iMsg.Msg.(*ies.UplinkNASTransport); ok {
			r.handleUplinkNasTransport(content)
			return
		}
	case ies.ProcedureCode_NGReset:
		if content, ok := iMsg.Msg.(*ies.NGReset); ok {
			r.handleNGReset(content)
			return
		}
	case ies.ProcedureCode_HandoverCancel:
		if content, ok := iMsg.Msg.(*ies.HandoverCancel); ok {
			r.handleHandoverCancel(content)
			return
		}
	case ies.ProcedureCode_UEContextReleaseRequest:
		if content, ok := iMsg.Msg.(*ies.UEContextReleaseRequest); ok {
			r.handleUEContextReleaseRequest(content)
			return
		}
	case ies.ProcedureCode_NASNonDeliveryIndication:
		if content, ok := iMsg.Msg.(*ies.NASNonDeliveryIndication); ok {
			r.handleNasNonDeliveryIndication(content)
			return
		}
	case ies.ProcedureCode_LocationReportingFailureIndication:
		if content, ok := iMsg.Msg.(*ies.LocationReportingFailureIndication); ok {
			r.handleLocationReportingFailureIndication(content)
			return
		}

	case ies.ProcedureCode_ErrorIndication:
		if content, ok := iMsg.Msg.(*ies.ErrorIndication); ok {
			r.handleErrorIndication(content)
			return
		}
	case ies.ProcedureCode_UERadioCapabilityInfoIndication:
		if content, ok := iMsg.Msg.(*ies.UERadioCapabilityInfoIndication); ok {
			r.handleUERadioCapabilityInfoIndication(content)
			return
		}
	case ies.ProcedureCode_HandoverNotification:
		if content, ok := iMsg.Msg.(*ies.HandoverNotify); ok {
			r.handleHandoverNotify(content)
			return
		}
	case ies.ProcedureCode_HandoverPreparation:
		if content, ok := iMsg.Msg.(*ies.HandoverRequired); ok {
			r.handleHandoverRequired(content)
			return
		}
	case ies.ProcedureCode_RANConfigurationUpdate:
		if content, ok := iMsg.Msg.(*ies.RANConfigurationUpdate); ok {
			r.handleRanConfigurationUpdate(content)
			return
		}
	case ies.ProcedureCode_RRCInactiveTransitionReport:
		if content, ok := iMsg.Msg.(*ies.RRCInactiveTransitionReport); ok {
			r.handleRRCInactiveTransitionReport(content)
			return
		}
	case ies.ProcedureCode_PDUSessionResourceNotify:
		if content, ok := iMsg.Msg.(*ies.PDUSessionResourceNotify); ok {
			r.handlePDUSessionResourceNotify(content)
			return
		}
	case ies.ProcedureCode_PathSwitchRequest:
		if content, ok := iMsg.Msg.(*ies.PathSwitchRequest); ok {
			r.handlePathSwitchRequest(content)
			return
		}
	case ies.ProcedureCode_LocationReport:
		if content, ok := iMsg.Msg.(*ies.LocationReport); ok {
			r.handleLocationReport(content)
			return
		}
	case ies.ProcedureCode_UplinkUEAssociatedNRPPaTransport:
		if content, ok := iMsg.Msg.(*ies.UplinkUEAssociatedNRPPaTransport); ok {
			r.handleUplinkUEAssociatedNRPPATransport(content)
			return
		}
	case ies.ProcedureCode_UplinkRANConfigurationTransfer:
		if content, ok := iMsg.Msg.(*ies.UplinkRANConfigurationTransfer); ok {
			r.handleUplinkRanConfigurationTransfer(content)
			return
		}
	case ies.ProcedureCode_PDUSessionResourceModifyIndication:
		if content, ok := iMsg.Msg.(*ies.PDUSessionResourceModifyIndication); ok {
			r.handlePDUSessionResourceModifyIndication(content)
			return
		}
	case ies.ProcedureCode_CellTrafficTrace:
		if content, ok := iMsg.Msg.(*ies.CellTrafficTrace); ok {
			r.handleCellTrafficTrace(content)
			return
		}
	case ies.ProcedureCode_UplinkRANStatusTransfer:
		if content, ok := iMsg.Msg.(*ies.UplinkRANStatusTransfer); ok {
			r.handleUplinkRanStatusTransfer(content)
			return
		}
	case ies.ProcedureCode_UplinkNonUEAssociatedNRPPaTransport:
		if content, ok := iMsg.Msg.(*ies.UplinkNonUEAssociatedNRPPaTransport); ok {
			r.handleUplinkNonUEAssociatedNRPPATransport(content)
			return
		}
	default:
		r.Warnf("Not implemented(InitiatingMessage, procedureCode:%d)\n", iMsg.ProcedureCode.Value)
		return
	}

	r.Errorf("Msg with empty content for procedure code %d", iMsg.ProcedureCode.Value)
	return
}

func (r *Ran) HandleSuccessfulMsg(sMsg *ngap.NgapMessage) {
	switch sMsg.ProcedureCode.Value {
	case ies.ProcedureCode_NGReset:
		if content, ok := sMsg.Msg.(*ies.NGResetAcknowledge); ok {
			r.handleNGResetAcknowledge(content)
			return
		}
	case ies.ProcedureCode_UEContextRelease:
		if content, ok := sMsg.Msg.(*ies.UEContextReleaseComplete); ok {
			r.handleUEContextReleaseComplete(content)
			return
		}
	case ies.ProcedureCode_PDUSessionResourceRelease:
		if content, ok := sMsg.Msg.(*ies.PDUSessionResourceReleaseResponse); ok {
			r.handlePDUSessionResourceReleaseResponse(content)
			return
		}
	case ies.ProcedureCode_UERadioCapabilityCheck:
		if content, ok := sMsg.Msg.(*ies.UERadioCapabilityCheckResponse); ok {
			r.handleUERadioCapabilityCheckResponse(content)
			return
		}
	case ies.ProcedureCode_AMFConfigurationUpdate:
		if content, ok := sMsg.Msg.(*ies.AMFConfigurationUpdateAcknowledge); ok {
			r.handleAMFconfigurationUpdateAcknowledge(content)
			return
		}
	case ies.ProcedureCode_InitialContextSetup:
		if content, ok := sMsg.Msg.(*ies.InitialContextSetupResponse); ok {
			r.handleInitialContextSetupResponse(content)
			return
		}
	case ies.ProcedureCode_UEContextModification:
		if content, ok := sMsg.Msg.(*ies.UEContextModificationResponse); ok {
			r.handleUEContextModificationResponse(content)
			return
		}
	case ies.ProcedureCode_PDUSessionResourceSetup:
		if content, ok := sMsg.Msg.(*ies.PDUSessionResourceSetupResponse); ok {
			r.handlePDUSessionResourceSetupResponse(content)
			return
		}
	case ies.ProcedureCode_PDUSessionResourceModify:
		if content, ok := sMsg.Msg.(*ies.PDUSessionResourceModifyResponse); ok {
			r.handlePDUSessionResourceModifyResponse(content)
			return
		}
	case ies.ProcedureCode_HandoverResourceAllocation:
		if content, ok := sMsg.Msg.(*ies.HandoverRequestAcknowledge); ok {
			r.handleHandoverRequestAcknowledge(content)
			return
		}
	default:
		r.Warnf("Not implemented(Successful Outcome, procedureCode:%d)", sMsg.ProcedureCode.Value)
		return
	}
	r.Errorf("Msg with empty content for procedure code %d", sMsg.ProcedureCode.Value)
}

func (r *Ran) HandleUnsuccessfulMsg(uMsg *ngap.NgapMessage) {
	switch uMsg.ProcedureCode.Value {
	case ies.ProcedureCode_AMFConfigurationUpdate:
		if content, ok := uMsg.Msg.(*ies.AMFConfigurationUpdateFailure); ok {
			r.handleAMFconfigurationUpdateFailure(content)
			return
		}
	case ies.ProcedureCode_InitialContextSetup:
		if content, ok := uMsg.Msg.(*ies.InitialContextSetupFailure); ok {
			r.handleInitialContextSetupFailure(content)
			return
		}
	case ies.ProcedureCode_UEContextModification:
		if content, ok := uMsg.Msg.(*ies.UEContextModificationFailure); ok {
			r.handleUEContextModificationFailure(content)
			return
		}
	case ies.ProcedureCode_HandoverResourceAllocation:
		if content, ok := uMsg.Msg.(*ies.HandoverFailure); ok {
			r.handleHandoverFailure(content)
			return
		}
	default:
		r.Warnf("Not implemented(Unsuccessful Outcome, procedureCode:%d)", uMsg.ProcedureCode.Value)
		return
	}

	r.Errorf("Msg with empty content for procedure code %d", uMsg.ProcedureCode.Value)
}
