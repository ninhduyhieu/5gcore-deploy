package sm

import (
	"encoding/binary"
	"etrib5gc/internal/fsm"
	"fmt"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

type RanUeInfo interface {
	getTransportNets() []string
	getRanUeInfo() *models.RanUeInfo
}

// RanUeInfo from SmContextUpdateData (RanUeInfo update through AMF)
// NOTE: in cases where handover/path switching are trigger from RAN directly,
// we need another implementation for the RanUeInfo interface
type smContextUpdateData struct {
	*models.SmContextUpdateData
}

func (d *smContextUpdateData) getTransportNets() []string {
	return d.RanTransportNets
}

func (d *smContextUpdateData) getRanUeInfo() *models.RanUeInfo {
	return d.RanUeInfo
}

func (smCtx *SmContext) handlePathSwitchRequest(ue RanUeInfo, n2SmInfo []byte) error {
	smCtx.Debugf("Handle N2SmInfo PathSwitchRequest")

	//update ran information
	if info := ue.getRanUeInfo(); info != nil {
		if err := smCtx.n1n2.setRanUeContext(info); err != nil {
			return utils.WrapError("Update RanUeInfo", err)
		}
	}

	smCtx.ranTransportNets = ue.getTransportNets()

	//decode message
	msg := new(ies.PathSwitchRequestTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode N2SmInfo", err)
	}

	//set default cause
	causePresent := 0
	causeValue := aper.Enumerated(0)
	failToSwitch := false

	//check if session in active state
	curstate := smCtx.state.CurrentState()
	if curstate != SM_ACTIVE {
		//session is not active
		failToSwitch = true
		//TODO: set a right cause
	} else if !smCtx.tunnel.IsConnectedToRan(smCtx.ranTransportNets) {
		//Ran is unreachable
		failToSwitch = true
		//TODO: set a right cause
	} else if GTPTunnel := msg.DLNGUUPTNLInformation.GTPTunnel; GTPTunnel == nil {
		//No GTP info from RAN
		failToSwitch = true
		//TODO: set a right cause
	} else {
		//modify RAN-connected UPF
		if err := smCtx.tunnel.UpdateRanInfo(GTPTunnel.TransportLayerAddress.Bytes, binary.BigEndian.Uint32(GTPTunnel.GTPTEID)); err != nil {
			smCtx.Warnf("Fail to update Ran information for the tunnel: %+v", err)
			//TODO: set a right cause

			//TODO: trigger session release, keep SmContext, move to SM_INACTIVE
			/*
				dat := common.NewEventData[ies.PathSwitchRequestSetupFailedTransfer](RELEASE_PATHSWITCH, msg)
				event := fsm.NewEventData[common.EventData](ReleaseTriggerEvent, dat)
				smCtx.state.SetNextEvent(event)
			*/
		} else {
			if n2SmInfo, err := smCtx.buildPathSwitchRequestAcknowledgeTransfer(); err != nil {
				return utils.WrapError("Build PathSwitchRequestAcknowledgeTransfer", err)
			} else {
				if err = smCtx.sendN2SmInfo(n2SmInfo, models.N2SMINFOTYPE_PATH_SWITCH_REQ_ACK); err != nil {
					return utils.WrapError("Send PathSwitchRequestAcknowledgeTransfer", err)
				}
			}
		}
	}

	if failToSwitch {
		//send PathSwithUnsuccessfulTransfer to Ran
		if n2SmInfo, err := smCtx.buildPathSwitchRequestUnsuccessfulTransfer(causePresent, causeValue); err != nil {
			return utils.WrapError("Build PathSwithUnccessfulTransfer", err)
		} else {
			if err = smCtx.sendN2SmInfo(n2SmInfo, models.N2SMINFOTYPE_PATH_SWITCH_REQ_FAIL); err != nil {
				return utils.WrapError("Send PathSwitchRequestUnsuccessfulTransfer", err)
			}
		}

	}
	return nil
}

func (smCtx *SmContext) handlePathSwitchSetupFail(ue RanUeInfo, n2SmInfo []byte) error {
	smCtx.Debugf("Handle N2SmInfo PathSwitchSetupFail")

	//update ran information
	if info := ue.getRanUeInfo(); info != nil {
		if err := smCtx.n1n2.setRanUeContext(info); err != nil {
			return utils.WrapError("Update RanUeInfo", err)
		}
	}
	smCtx.ranTransportNets = ue.getTransportNets()

	//check if sesion in active state
	curstate := smCtx.state.CurrentState()
	if curstate != SM_ACTIVE {
		return fmt.Errorf("SmContext not active")
	}

	//decode message
	msg := new(ies.PathSwitchRequestSetupFailedTransfer)

	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode PathSwitchRequestSetupFailedTransfer", err)
	}

	smCtx.Infof("Received a PathSwitchRequestSetupFailedTransfe")
	//trigger session release
	event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
		evType: RELEASE_PATHSWITCH,
		dat:    msg,
	})
	smCtx.state.SetNextEvent(event)

	return nil
}
