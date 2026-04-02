package sm

import (
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

// session must be in SM_ACTIVE
func (smCtx *SmContext) handleUpCnxState(mode models.UpCnxState) (err error) {
	state := smCtx.state.CurrentState()
	if state != SM_ACTIVE {
		err = fmt.Errorf("Session not active")
		return
	}
	switch mode {
	case models.UPCNXSTATE_ACTIVATING:
		err = smCtx.handleUpCnxStateActivating()
	case models.UPCNXSTATE_DEACTIVATED:
		err = smCtx.handleUpCnxStateDeactivating()
	default:
		err = fmt.Errorf("Unknown UpCnxState request")
	}
	return
}

// the session is active, the SMF notifies RAN to setup the resource for the session
func (smCtx *SmContext) handleUpCnxStateActivating() (err error) {
	smCtx.Debugf("Handle UpCnxStateActivating")
	switch smCtx.upCnxState {
	case UPCNXSTATE_SUSPENDED:
		//ask RAN to prepare resource
		var n2SmInfo []byte
		if n2SmInfo, err = smCtx.buildPduSessionResourceSetupRequestTransfer(); err != nil {
			err = utils.WrapError("Build PduSessionResourceRequestTransfer", err)
		} else {
			smCtx.upCnxState = UPCNXSTATE_ACTIVATING
			if err = smCtx.sendN2SmInfo(n2SmInfo, models.N2SMINFOTYPE_PDU_RES_SETUP_REQ); err != nil {
				err = utils.WrapError("Send N2SmInfo", err)
			} else {
				smCtx.Infof("N2SmInfo PduSessionResourceRequestTransfer was sent")
			}
		}
	default:
		smCtx.Warnf("UpCnxState is not SUSPENDED")
	}

	return
}

func (smCtx *SmContext) handleUpCnxStateDeactivating() (err error) {
	smCtx.Debugf("Handle UpCnxStateDeactivating")
	switch smCtx.upCnxState {
	case UPCNXSTATE_ACTIVATED, UPCNXSTATE_ACTIVATING:
		if err = smCtx.tunnel.DisconnectRan(); err != nil {
			smCtx.Warnf("Fail to modify Pfcp sessions: %+v", err)
		}
		smCtx.upCnxState = UPCNXSTATE_SUSPENDED
		var pdu []byte
		if pdu, err = smCtx.buildPduSessionResourceReleaseCommandTransfer(); err != nil {
			utils.WrapError("Build PduSessionResourceReleaseCommand", err)
		} else {
			if err = smCtx.sendN2SmInfo(pdu, models.N2SMINFOTYPE_PDU_RES_REL_CMD); err != nil {
				utils.WrapError("Send N2SmInfo", err)
			} else {
				smCtx.Infof("N2SmInfo PduSessionResourceReleaseCommand was sent")
			}
		}
	default:
		smCtx.Warnf("UpCnxState is not ACTIVATED or ACTIVATING")
	}
	return
}
