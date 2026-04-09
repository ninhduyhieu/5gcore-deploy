package uecontext

import (
	"etrib5gc/nfs/amf/sm"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

// Look for the SmContext or create a new one then forward the Nas message
func (ueCtx *UeContext) handleN1SmUplink(msg *nas.UlNasTransport, isGpp bool) error {
	n1SmPdu := []byte(msg.PayloadContainer)

	//handle old Pdu session Id
	if msg.OldPduSessionId != nil {
		ueCtx.Infof("N1Sm to be forwarded to old session %d", *msg.OldPduSessionId)
		if smCtx := ueCtx.findSmContext(*msg.OldPduSessionId); smCtx == nil {
			return fmt.Errorf("SmContext not found to handle old Pdu session %d", *msg.OldPduSessionId)
		} else {
			ueCtx.forwardN1Sm(smCtx, n1SmPdu)
			return nil
		}
	}

	sessionId := uint8(*msg.PduSessionId)

	ueCtx.Debugf("N1Sm to be forwarded to session %d", sessionId)

	//check for emergency request type
	requestType := msg.RequestType
	if requestType != nil {
		switch uint8(*requestType) {
		case nas.UlNasTransportRequestTypeInitialEmergencyRequest:
			fallthrough
		case nas.UlNasTransportRequestTypeExistingEmergencyPduSession:
			return fmt.Errorf("Emergency PDU Session not supported")
		}
	}

	smCtx := ueCtx.findSmContext(sessionId)
	var err error

	//has the session context
	if smCtx != nil {
		ueCtx.Tracef("SmContext for session %d found to forward N1Sm", sessionId)
		// no request type, just forward N1Sm
		if requestType == nil {
			ueCtx.Tracef("RequestType empty for session %d", sessionId)
			ueCtx.forwardN1Sm(smCtx, n1SmPdu)
			return nil
		}

		switch uint8(*requestType) {
		case nas.UlNasTransportRequestTypeInitialRequest:
			//a duplicated pdu session (for initial pdu session request)
			//need to release it
			ueCtx.Tracef("RequestType = INITIAL for session %d", sessionId)

			//when remove a duplicated session, AMF will not handle any message
			//transfer between SMF and UE/gnB
			ueCtx.Warnf("Existing SmContext [%d] found for an initial request type UlNasTransport, remove it", sessionId)
			req := &models.SmContextUpdateData{
				Release: new(bool),
				Cause:   models.CAUSE_REL_DUE_TO_DUPLICATE_SESSION_ID,
			}
			*req.Release = true

			if _, _, err = smCtx.SendUpdateSmContext(req, nil, nil); err != nil {
				ueCtx.Warnf("SMF failed to remove duplicated session [%d]: %+v", err, sessionId)
			} else {
				ueCtx.Infof("A duplicated session [id=%d] was removed at SMF", sessionId)
			}
			ueCtx.deleteSmContext(sessionId) //remove session regardless of how SMF handle the deletion

			//now create a new sm context to forward the message
			if err = ueCtx.createSmContext(isGpp, msg, false); err != nil {
				return utils.WrapError("Create SmContext", err)
			}

		//existing pdu session
		case nas.UlNasTransportRequestTypeExistingPduSession:
			ueCtx.Debugf("RequestType = EXISTING for session %d", sessionId)
			ueCtx.forwardN1Sm(smCtx, n1SmPdu)
			return nil

		// other types, just forward
		default:
			ueCtx.Debugf("RequestType = OTHERS for session %d", sessionId)
			ueCtx.forwardN1Sm(smCtx, n1SmPdu)
			return nil
		}
	} else { // SmContext does not exist
		ueCtx.Tracef("SmContext not found for id=%d", sessionId)

		if requestType == nil {
			return fmt.Errorf("SmContext not exist and RequestType is nil [session %d]", sessionId)
		}
		switch uint8(*requestType) {
		//initial request, find SMF and create an SmContext
		case nas.UlNasTransportRequestTypeInitialRequest:
			if err = ueCtx.createSmContext(isGpp, msg, false); err == nil {
				return utils.WrapError("Create SmContext", err)
			}

			// SmContext may exist in SMF but AMF has lost its track
		case nas.UlNasTransportRequestTypeModificationRequest, nas.UlNasTransportRequestTypeExistingPduSession:
			// Use information from UDM to create SmContext then ask SMF to
			// create the session
			if err = ueCtx.createSmContext(isGpp, msg, true); err != nil {
				return utils.WrapError("Create SmContext", err)
			}
		default:
			return fmt.Errorf("Unknown RequestType %d", uint8(*requestType))
		}
	}
	return nil
}

func (ueCtx *UeContext) forwardN1Sm(smCtx *sm.SmContext, n1Sm []byte) {
	_uePool.pubWorkers.Go(func() {
		msg := &models.SmContextUpdateData{
			N1SmMsg: &models.RefToBinaryData{
				ContentId: "N1SmMsg",
			},
			Pei: ueCtx.pei,
			//Gpsi: ueCtx.gpsi,
		}
		if rsp, ersp, err := smCtx.SendUpdateSmContext(msg, n1Sm, nil); err != nil {
			//Fail to forward N1Sm to SMF
			ueCtx.Errorf("Fail to forward N1Sm for session %d: %+v", smCtx.GetId(), err)
			ueCtx.sendN1SmError(smCtx.IsGpp(), n1Sm, smCtx.GetId())
		} else {
			ueCtx.Infof("N1SM is forward to SMF of session %d", smCtx.GetId())
			ueCtx.forwardUpdateSmContextResponsesDownlink(smCtx, rsp, ersp)
		}
	})
}
