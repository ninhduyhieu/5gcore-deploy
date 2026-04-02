package uecontext

import (
	"context"
	"etrib5gc/internal/fsm"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/utils"
)

// reveive Nas MM uplink from gnB
func (ueCtx *UeContext) HandleNasUplink(isGpp bool, pdu []byte) error {
	var msg *nas.DecodedGmmMessage
	if nasMsg, err := nas.Decode(ueCtx.getNasContext(), pdu, isGpp); err != nil {
		return utils.WrapError("Decode NasUl failed", err)
	} else if msg = nasMsg.Gmm; msg == nil { //still have no decoded Gmm
		return fmt.Errorf("Decoded Nas has no N1Mm") //malicious RAN?
	}

	//send to GMM state machine to handle
	ueCtx.sendEvent(context.Background(), fsm.NewEventData(N1MmEvent, &RanSbiEventData{
		isGpp: isGpp,
		evDat: msg,
	}))
	return nil
}

// handle Nas MM in state machine
func (ueCtx *UeContext) handleN1Mm(ctx context.Context, isGpp bool, msg *nas.DecodedGmmMessage, initCtx *AttachmentInfo) {
	if msg.MacFailed {
		switch msg.MsgType {
		case nas.RegistrationRequestMsgType, nas.ServiceRequestMsgType, nas.DeregistrationRequestFromUeMsgType, nas.IdentityResponseMsgType, nas.AuthenticationResponseMsgType, nas.AuthenticationFailureMsgType:
			if initCtx == nil { //not initial UE message
				ueCtx.Warnf("Mac failed when decode message of type %d", msg.MsgType)
			}
		default:
			ueCtx.Errorf("Mac failed when decode message of type %d", msg.MsgType)
			ueCtx.sendNasError(isGpp)
			return
		}
	}

	if attCtx := ueCtx.attCtx; attCtx != nil && attCtx.proc != nil { //registration on-going
		//forward the message to the common procedure to handle
		attCtx.proc.ReceiveN1Mm(ctx, msg)
		return
	}

	curState := ueCtx.state.CurrentState()
	switch msg.MsgType {
	case nas.RegistrationRequestMsgType:
		if curState == MM_IDLE {
			ueCtx.startRegistration(context.Background(), isGpp, msg, initCtx)
		} else {
			ueCtx.Warnf("Receive NAS RegistrationRequest not in MM_IDLE")
			ueCtx.sendRegistrationReject(isGpp, nas.Cause5GMMMessageTypeNotCompatibleWithTheProtocolState)
		}
	case nas.ServiceRequestMsgType:
		if curState == MM_IDLE {
			ueCtx.startRegistration(context.Background(), isGpp, msg, initCtx)
		} else {
			ueCtx.Warnf("Receive NAS ServiceRequest not in MM_IDLE")
			ueCtx.sendServiceReject(isGpp, nas.Cause5GMMMessageTypeNotCompatibleWithTheProtocolState, nil)
		}

	case nas.DeregistrationRequestFromUeMsgType:
		if ueCtx.isRegistered(isGpp) {
			ueCtx.handleDeregistrationRequest(ctx, isGpp, msg.DeregistrationRequestFromUe, msg.MacFailed)
		} else {
			ueCtx.Warnf("NAS DeregistrationRequestFromUe not handled [Ue not registered]")
		}

	case nas.UlNasTransportMsgType:
		ueCtx.handleUlNasTransport(ctx, isGpp, msg.UlNasTransport)

	case nas.NotificationResponseMsgType:
		ueCtx.handleNotificationResponse(ctx, isGpp, msg.NotificationResponse)

	case nas.ConfigurationUpdateCompleteMsgType:
		ueCtx.handleConfigurationUpdateComplete(ctx, isGpp, msg.ConfigurationUpdateComplete)

	case nas.GmmStatusMsgType:
		ueCtx.handleGmmStatus(ctx, isGpp, msg.GmmStatus)

	case nas.RegistrationCompleteMsgType:
		if curState == MM_REGISTERING {
			ueCtx.handleRegistrationComplete(ctx, isGpp, msg.RegistrationComplete)
		} else {
			ueCtx.Warnf("Receive NAS RegistrationComplete not in MM_REGISTERING")
		}

	case nas.DeregistrationAcceptFromUeMsgType:
		if curState == MM_DEREGISTERING {
			ueCtx.handleDeregistrationAccept(ctx, isGpp, msg.DeregistrationAcceptFromUe)
		} else {
			ueCtx.Warnf("Receive NAS DeregistrationAcceptToUe not in MM_DEREGISTERING")
		}

	default:
		ueCtx.Warnf("Receive unexpected NAS message: MsgType=%d", msg.MsgType)
	}
}

func (ueCtx *UeContext) handleGmmStatus(ctx context.Context, isGpp bool, msg *nas.GmmStatus) {
	ueCtx.Warnf("Handle NAS GmmStatus [not implemented]")
}

func (ueCtx *UeContext) handleNotificationResponse(ctx context.Context, isGpp bool, msg *nas.NotificationResponse) {
	if ueCtx.n1n2 == nil {
		ueCtx.Warnf("Receive NotificationResponse while there is no pending N1N2")
	} else {
		ueCtx.n1n2.t3565.Stop() //stop notification timer
		//TODO: handle PduSessionStatus from the message
	}
}

// NOTE: ignore any uplink nas messages which are not supported now
func (ueCtx *UeContext) handleUlNasTransport(ctx context.Context, isGpp bool, msg *nas.UlNasTransport) {
	ueCtx.Debugf("Handle N1MM UplinkNasTransport")
	containerType := msg.PayloadContainerType

	switch containerType {
	// TS 24.501 5.4.5.2.3 case a)
	case nas.PayloadContainerTypeN1SMInfo:
		//only handle this case (forward N1Sm message toward an SMF)

		n1SmPdu := []byte(msg.PayloadContainer)
		if len(n1SmPdu) == 0 {
			ueCtx.Warnf("UplinkNasTransport with empty Pdu")
			return //ignore
		}

		if msg.PduSessionId == nil {
			ueCtx.Warnf("UplinkNasTransport with no Pdu session id")
			ueCtx.sendN1SmError(isGpp, n1SmPdu, MAX_PDU_SESSIONS)
			return
		}

		if err := ueCtx.handleN1SmUplink(msg, isGpp); err != nil {
			ueCtx.Errorf("Fail to forward N1Sm to SMF: %+v", err)
			sessionId := uint8(*msg.PduSessionId)
			ueCtx.sendN1SmError(isGpp, n1SmPdu, sessionId)
		}

	case nas.PayloadContainerTypeSMS:
		ueCtx.Error("PayloadContainerTypeSMS not supported")

	case nas.PayloadContainerTypeLPP:
		ueCtx.Error("PayloadContainerTypeLPP not supported")

	case nas.PayloadContainerTypeSOR:
		ueCtx.Error("PayloadContainerTypeSOR not supported")

	case nas.PayloadContainerTypeUEPolicy:
		ueCtx.Warn("AMF Transfering UEPolicy To PCF not implemented")
		//TODO: to be implemented

	case nas.PayloadContainerTypeUEParameterUpdate:
		ueCtx.Warn("AMF Transfering UEParameterUpdate To UDM not implemented")
		//TODO: to be implemented

	case nas.PayloadContainerTypeMultiplePayload:
		ueCtx.Error("PayloadContainerTypeMultiplePayload not supported")
	default:
		ueCtx.Error("Unknown PayloadContainer type: %d", containerType)
	}
}
