package sm

import (
	"encoding/binary"
	smfctx "etrib5gc/nfs/smf/context"
	"etrib5gc/nfs/smf/tunnel"
	"fmt"
	"net"

	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

type IndirectPath interface {
	Release()
	UpdateRanInfo(net.IP, uint32) error
	GetDlGtpTunnel() *ies.GTPTunnel
}

type HandoverContext struct {
	smCtx            *SmContext
	ranTransportNets []string //target gnB transport networks
	ranCli           sbi.ConsumerClient
	directMode       bool           //is a direct path between gnB available?
	isPrepared       bool           // has target gnB prepared?
	indirectPath     IndirectPath   // indirect path data structure
	newTunnel        *tunnel.Tunnel // new tunnel (make before break)
	newUeIp          *smfctx.LeasedIp
	hoTimer          common.Timer
}

func (smCtx *SmContext) createHandoverContext(directMode bool) *HandoverContext {
	ho := &HandoverContext{
		smCtx:      smCtx,
		directMode: directMode,
	}
	return ho
}

// source gnB start a Handover
func (smCtx *SmContext) handleHoStatePreparing(ctx *models.SmContextUpdateData, n2SmInfo []byte) error {
	//check N2SmInfo message type
	if ctx.N2SmInfoType != models.N2SMINFOTYPE_HANDOVER_REQUIRED {
		return fmt.Errorf("Invalid N2SmInfo data in handover preparing")
	}

	//session must be active
	state := smCtx.state.CurrentState()
	if state != SM_ACTIVE {
		return fmt.Errorf("Session not active [%s] to handle Handover", stateString(state))
	}

	//decode message
	msg := new(ies.HandoverRequiredTransfer)
	if err := msg.Decode(n2SmInfo); err != nil {
		return utils.WrapError("Decode HandoverRequiredTransfer", err)
	}

	hoCtx := smCtx.createHandoverContext(msg.DirectForwardingPathAvailability != nil)

	//NOTE:
	// - in SSC1 we must try to configure the tunnel so that the target gnB is
	// reachable
	// - in SSC3 we always release N4 session then create the tunnel again
	// - in SSC2 the if the target gnB is not reachable with current UPF paths, we
	// may release the N4 session then create a the tunnel again (like SSC3);
	// or we can try to reconfigure the tunnel (like SSC1)

	isMakeBeforeBreak := false
	createNewTunnel := false
	allocateNewUeIp := false
	var err error

	switch smCtx.sscMode {
	case SSC_MODE_1: //make before break
		isMakeBeforeBreak = true
		if !smCtx.tunnel.IsConnectedToRan(ctx.RanTransportNets) {
			//the target gnB can't connect to current UPF path, need to create a new tunnel
			createNewTunnel = true
		}
	case SSC_MODE_2: //make before break (or break before make)
		if !smCtx.tunnel.IsConnectedToRan(ctx.RanTransportNets) {
			//the target gnB can't connect to current UPF path, need to create a new tunnel
			createNewTunnel = true
		}
	case SSC_MODE_3: // break before make
		allocateNewUeIp = true
		createNewTunnel = true
	}

	if allocateNewUeIp {
		if hoCtx.newUeIp, err = smfctx.AllocateUeIp(smCtx.dnn); err != nil {
			return utils.WrapError("Allocate New ueIp", err)
		}
	}

	if createNewTunnel {
		newTunnel := tunnel.CreateTunnel(smCtx, ctx.RanTransportNets)
		if err = newTunnel.HandleSmPolicyDecision(smCtx.smPol); err != nil {
			err = utils.WrapError("Create new tunnel for handover", err)
		} else if isMakeBeforeBreak {
			hoCtx.newTunnel = newTunnel
		} else { //break before make
			//release old tunnel
			smCtx.tunnel.Release()
			smCtx.tunnel = newTunnel
		}
	}

	if err != nil { //release resource if there was an error
		smCtx.Errorf(err.Error())
		smCtx.cancelHandover(hoCtx)
	} else { //move in to SM_HANDOVER
		event := fsm.NewEventData(HandoverEvent, hoCtx)
		smCtx.state.SetNextEvent(event)
	}
	return nil
}

// target gnB acknowledge that handover resource is prepared or failed
func (smCtx *SmContext) handleHoStatePrepared(ctx *models.SmContextUpdateData, n2SmInfo []byte) error {
	//check state
	state := smCtx.state.CurrentState()
	if state != SM_HANDOVER {
		return fmt.Errorf("Session not in handover")
	}

	//check N2SmInfo message type
	if ctx.N2SmInfoType == models.N2SMINFOTYPE_HANDOVER_REQ_ACK { //resource allocation at target gnB success
		//decode message
		msg := new(ies.HandoverRequestAcknowledgeTransfer)
		if err := msg.Decode(n2SmInfo); err != nil {
			return utils.WrapError("Decode HandoverRequestAcknowledgeTransfer", err)
		}

		//update Ran information for the tunnel
		gtpTunnel := msg.DLNGUUPTNLInformation.GTPTunnel
		if gtpTunnel == nil {
			return fmt.Errorf("DLNGUUPGTPTunnel is empty")
		}

		//mark that the handover is prepared at the target gnB
		smCtx.hoCtx.isPrepared = true

		curTunnel := smCtx.tunnel
		if smCtx.hoCtx.newTunnel != nil {
			curTunnel = smCtx.hoCtx.newTunnel
		}

		//update downlink information for the RAN-connected UPF
		if err := curTunnel.UpdateRanInfo(gtpTunnel.TransportLayerAddress.Bytes, binary.BigEndian.Uint32(gtpTunnel.GTPTEID)); err != nil {
			return utils.WrapError("Update RanInfo for new tunnel", err)
		}

		dlForwardingInfo := msg.DLForwardingUPTNLInformation
		if dlForwardingInfo == nil || dlForwardingInfo.GTPTunnel == nil {
			return fmt.Errorf("DL Forwarding Info not provision")
		}

		if !smCtx.hoCtx.directMode {
			gtpTunnel = dlForwardingInfo.GTPTunnel
			//1. create an Indirect Forwarding Path with a single UPF (the one connected to RAN from existing path)
			if indirectPath, err := tunnel.CreateIndirectPath(curTunnel, ctx.RanTransportNets); err != nil {
				return utils.WrapError("Create Indirect Path", err)
			} else {
				smCtx.hoCtx.indirectPath = indirectPath
			}
			//2. Add PDRs for forwarding returning queued packets from the source gnB toward target gnB
			if err := smCtx.hoCtx.indirectPath.UpdateRanInfo(gtpTunnel.TransportLayerAddress.Bytes, binary.BigEndian.Uint32(gtpTunnel.GTPTEID)); err != nil {
				return utils.WrapError("Update RanInfo for new indirect path", err)
			}
			//3. now change the DLFowardUPTNLInformation to send to the source gnB
			dlForwardingInfo = &ies.UPTransportLayerInformation{
				Choice:    ies.UPTransportLayerInformationPresentGtptunnel,
				GTPTunnel: smCtx.hoCtx.indirectPath.GetDlGtpTunnel(),
			}
		}

		if pdu, err := smCtx.buildHandoverCommandTransfer(dlForwardingInfo); err != nil {
			return fmt.Errorf("Fail to build HandoverCommandTransfer: %+v", err)
		} else if err := smCtx.sendN2SmInfo(pdu, models.N2SMINFOTYPE_HANDOVER_CMD); err != nil {
			return utils.WrapError("Send HandOverCommandTransfer", err)
		}

	} else if ctx.N2SmInfoType == models.N2SMINFOTYPE_HANDOVER_RES_ALLOC_FAIL { //resource allocation failure
		//decode message
		msg := new(ies.HandoverResourceAllocationUnsuccessfulTransfer)
		if err := msg.Decode(n2SmInfo); err != nil {
			return utils.WrapError("Decode HandoverResourceAllocationUnsuccessfulTransfer", err)
		}

		if pdu, err := smCtx.buildHandoverResourceAllocationUnsuccessfulTransfer(&msg.Cause); err != nil {
			return fmt.Errorf("Fail to build HandoverCommandTransfer: %+v", err)
		} else if err := smCtx.sendN2SmInfo(pdu, models.N2SMINFOTYPE_HANDOVER_PREP_FAIL); err != nil {
			utils.WrapError("Send HandOverCommandTransfer", err)
		}
		//release created tunnel
		smCtx.cancelHandover(smCtx.hoCtx)
	} else {
		return fmt.Errorf("Invalid N2SmInfo data in handover prepared")
	}
	return nil
}

// target gnB notify of handover completed
func (smCtx *SmContext) handleHoStateCompleted() error {
	smCtx.Infof("Handle HanoverCompleted")
	state := smCtx.state.CurrentState()
	if state != SM_HANDOVER {
		return fmt.Errorf("Session not in handover")
	}

	//stop timer
	smCtx.hoCtx.hoTimer.Stop()

	//remove indirect path and its rules
	if smCtx.hoCtx.indirectPath != nil {
		smCtx.hoCtx.indirectPath.Release()
	}

	if smCtx.hoCtx.newTunnel != nil { //make before break
		smCtx.tunnel.Release()               //release old tunnel
		smCtx.tunnel = smCtx.hoCtx.newTunnel //replace with the new one
	}
	if smCtx.hoCtx.newUeIp != nil {
		//free old UeIp
		smCtx.ueIp.Free()
		smCtx.ueIp = smCtx.hoCtx.newUeIp
	}
	event := fsm.NewEmptyEventData(HandoverCmplEvent)
	smCtx.state.SetNextEvent(event)
	return nil
}

// source gnB cancel the on-going handover
func (smCtx *SmContext) handleHoStateCancelled() error {
	state := smCtx.state.CurrentState()
	if state != SM_HANDOVER {
		return fmt.Errorf("Session not in handover")
	}

	smCtx.cancelHandover(smCtx.hoCtx)

	event := fsm.NewEmptyEventData(HandoverCmplEvent)
	smCtx.state.SetNextEvent(event)
	return nil
}

func (smCtx *SmContext) cancelHandover(hoCtx *HandoverContext) {

	//stop timer
	if hoCtx.hoTimer != nil {
		hoCtx.hoTimer.Stop()
		hoCtx.hoTimer = nil
	}

	//release the indirect path
	if hoCtx.indirectPath != nil {
		//release indirect path
		hoCtx.indirectPath.Release()
		hoCtx.indirectPath = nil
	}

	//release the new tunnel
	if hoCtx.newTunnel != nil {
		hoCtx.newTunnel.Release()
		hoCtx.newTunnel = nil
	}

	//release new allocated UeIP
	if hoCtx.newUeIp != nil {
		hoCtx.newUeIp.Free()
		hoCtx.newUeIp = nil
	}

	if hoCtx.isPrepared {
		//TODO: notify target gnB to release session resource
	}
}
