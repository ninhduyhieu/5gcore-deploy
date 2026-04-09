package ue

import (
	"context"
	"etrib5gc/internal/eventmux"
	"fmt"
)

const (
	SbiEvent uint8 = iota
	NgapEvent
	ForceCloseEvent
	AmfUeCtxReleaseTimeoutEvent //amf did not send UeCtxReleaseCommand
	GnbUeCtxReleaseTimeoutEvent //gnb did not response to UeCtxReleaseCommand
)

const ( //NGAP
	NAS_UPLINK uint8 = iota
	NAS_ERR
	RRC_STATE_REPORT
	RADIO_CAP_IND
	UECTX_SETUP_RSP
	UECTX_SETUP_FAIL
	UECTX_RELEASE_REQ
	UECTX_RELEASE_CMPL
	UECTX_MODIFY_RSP
	UECTX_MODIFY_FAIL
	PDU_REL_RES
	PDU_SET_RES
	PDU_MOD_RES
	PDU_NOTIFY
	PDU_MOD_IND
	HO_REQUIRED
	HO_REQ_ACK
	HO_FAILURE
	HO_NOTIFY
	HO_CANCEL
	HO_UL_RAN_STATUS
	PATHSWITCH_REQ
)

const ( //SBI
	N2SMINFO_DOWNLINK uint8 = iota
	NAS_DOWNLINK
	UPDATE_AMF_INFO
	PDU_SETUP
	PDU_MODIFY
	PDU_RELEASE
	UECTX_SETUP
	UECTX_MODIFY
	UECTX_RELEASE
	HANDOVER_REQUEST
)

func handleEvent(ctx context.Context, ueCtx *UeContext, payload any) error {
	ev, _ := payload.(*eventmux.EventData)
	switch ev.Type() {
	case NgapEvent:
		ueCtx.handleNgapEvent(ctx, eventmux.GetEventData[eventmux.EventData](ev))
	case SbiEvent:
		ueCtx.handleSbiEvent(ctx, eventmux.GetEventData[eventmux.EventData](ev))
	case ForceCloseEvent:
		ch := eventmux.GetEventData[chan struct{}](ev)
		ueCtx.forceClose(ctx, *ch)
	case AmfUeCtxReleaseTimeoutEvent:
		ueCtx.releaseUeContext()
	case GnbUeCtxReleaseTimeoutEvent:
		if ueCtx.releaseCtx != nil && ueCtx.releaseCtx.job != nil {
			//finailize pending request from AMF
			ueCtx.releaseCtx.job.Finalize(fmt.Errorf("Gnb not response to UeContext release command"))
		} else {
			ueCtx.clean()
		}
	}
	return nil
}
