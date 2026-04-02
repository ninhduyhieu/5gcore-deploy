package sm

import (
	"context"
	"etrib5gc/common"
	"time"

	"etrib5gc/internal/fsm"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

const (
	RELEASE_AMF        uint8 = iota // amf request to release [SmContextReleaseRequest]
	RELEASE_AMF_DUP                 // amf request to delete due to duplication [UpdateSmContextRequest]
	RELEASE_FORCE                   // app terminated [OAM]
	RELEASE_DUP                     // found exiting session [while handling PostSmContextCreateRequest]
	RELEASE_UE                      // ue request to release
	RELEASE_GNB                     // GnB request/indicate to release Pdu session resource
	RELEASE_PATHSWITCH              // path switch failed session
)
const (
	RELEASE_TIMEOUT time.Duration = 2 * time.Second
)

// in a release procedure, we ask UPFs and gnB to release resource, and we may
// need to ask UE to deactivate session.
// releaseContext hold a context of a releasing procedure. It tells where the
// release comes from and what to do at the end.
type ReleaseContext struct {
	smCtx     *SmContext
	trigger   *EventData
	notifyAmf bool
	doneChs   []chan struct{}
	t3592     common.Timer //PduSessionReleaseCommand timer
	t3592cnt  int
	n1Sm      []byte //encoded PduSessionReleaseCommand
}

func shouldNotifyAmf(evType uint8) bool {
	switch evType {
	case RELEASE_FORCE, RELEASE_GNB, RELEASE_UE, RELEASE_PATHSWITCH:
		return true
	}
	return false
}

// AMF request to release SmContext
func (smCtx *SmContext) handleReleaseSmContextRequest(task *ReleaseSmContextContext) {
	smCtx.Infof("AMF send ReleaseSmContextRequest to release the Sm Context")
	event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
		evType: RELEASE_AMF,
		dat:    task,
	})
	smCtx.state.SetNextEvent(event)
}

func (smCtx *SmContext) abortHandover() {
	smCtx.Warnf("Handover abort not implemented yet")
}

func (smCtx *SmContext) abortModification() {
	smCtx.Warnf("Session modification abort not implemented yet")
}

func (smCtx *SmContext) handleReleaseTrigger(e *EventData) {
	curState := smCtx.state.CurrentState()
	//if release is trigger by UE, check if the session is in a valid state to
	//release, if it is not send a rejection
	if e.evType == RELEASE_UE {
		switch curState {
		case SM_INACTIVATING, SM_INACTIVE: //reject release request
			//reject N1SM PduSessionReleaseRequest
			req, _ := e.dat.(*nas.PduSessionReleaseRequest)
			pti := req.GetPti()
			n1Cause := nas.Cause5GSMMessageNotCompatibleWithTheProtocolState
			if n1Sm, err := smCtx.buildPduSessionReleaseReject(pti, n1Cause); err != nil {
				smCtx.Errorf("Fail to build N1Sm PduSessionReleaseReject: %+v", err)
				smCtx.sendGsmStatus(req.GetPti(), nas.Cause5GSMNetworkFailure)
			} else {
				if err := smCtx.n1n2.sendN1N2(&N1N2Messages{
					n1Sm: n1Sm,
				}); err != nil {
					smCtx.Errorf("Fail to send N1Sm PduSessionReleaseReject: %+v", err)
				} else {
					smCtx.Infof("N1Sm PduSessionReleaseReject was sent")
				}
			}
			return
		default:
		}
	}
	releaseUe := false
	switch curState {
	case SM_INACTIVATING: //there must be a pending release context
		switch e.evType {
		case RELEASE_FORCE, RELEASE_DUP:
			//if trigger has a signal channel, add it to current release context
			if e.dat != nil {
				if ch, ok := e.dat.(chan struct{}); ok {
					smCtx.relCtx.doneChs = append(smCtx.relCtx.doneChs, ch)
				}
			}
		}
		if !shouldNotifyAmf(e.evType) {
			smCtx.relCtx.notifyAmf = false //overwrite AMF notification flag
		}

	case SM_HANDOVER:
		releaseUe = true
		smCtx.abortHandover()
	case SM_MODIFYING:
		smCtx.abortModification()
		releaseUe = true
	case SM_INACTIVE:
	case SM_ACTIVE:
		releaseUe = true
	}

	switch e.evType {
	case RELEASE_AMF_DUP, RELEASE_DUP, RELEASE_AMF: //do not talk to gnB and UE since AMF don't know about this session
		releaseUe = false
	default:
		smCtx.releaseGnb() //idempotent
	}

	smCtx.releaseSessionResources() //idempotent

	finalize := false
	if releaseUe {
		// build N1Sm PduSessionReleaseCommand
		pti := nas.PtiUnspecified //default value
		cause := nas.Cause5GSMRegularDeactivation
		if e.evType == RELEASE_UE {
			req, _ := e.dat.(*nas.PduSessionReleaseRequest)
			pti = req.GetPti()
			if req.GsmCause != nil {
				cause = uint8(*req.GsmCause)
			}
		}
		if n1Sm, err := smCtx.buildPduSessionReleaseCommand(pti, cause); err != nil {
			smCtx.Errorf("Fail to build N1Sm PduSessionReleaseCommand: %+v", err)
			if e.evType == RELEASE_UE { //get GsmStatus if Ue send release request
				smCtx.sendGsmStatus(pti, nas.Cause5GSMNetworkFailure)
			}
			finalize = true
		} else if err := smCtx.n1n2.sendN1N2(&N1N2Messages{
			n1Sm: n1Sm,
		}); err != nil {
			smCtx.Errorf("Fail to send N1N2 message to release session: %+v", err)
			finalize = true
		} else {
			smCtx.Infof("N1Sm PduSessionReleaseCommand was sent")
			//move to SM_INACTIVATING
			relCtx := &ReleaseContext{
				smCtx:     smCtx,
				n1Sm:      n1Sm,
				trigger:   e,
				notifyAmf: shouldNotifyAmf(e.evType),
				t3592: common.NewTimer(T3592, func() {
					smCtx.Warnf("T3592 timer expires")
					smCtx.sendEvent(context.TODO(), fsm.NewEmptyEventData(T3592Event))
				}, nil),
			}
			relCtx.t3592.Start()
			relCtx.t3592cnt++

			//move to SM_INACTIVATING state
			smCtx.state.SetNextEvent(fsm.NewEventData(ReleaseEvent, relCtx))
		}
	} else {
		finalize = true
	}
	if finalize {
		smCtx.finalizeRelease(e)
	}
}

// do things when release comes to end
func (smCtx *SmContext) finalizeRelease(trigger *EventData) {
	smCtx.Debugf("Finalize release SmContext")
	_pool.remove(smCtx)

	evType := trigger.evType

	// notify AMF
	notifyAmf := true
	if smCtx.relCtx != nil {
		notifyAmf = smCtx.relCtx.notifyAmf
	} else {
		notifyAmf = shouldNotifyAmf(trigger.evType)
	}

	if notifyAmf {
		if err := smCtx.notifyAmf(); err != nil {
			smCtx.Warnf("Fail to notify AMF: %+v", err)
		}
	}

	switch evType {
	case RELEASE_AMF:
		info, _ := trigger.dat.(*ReleaseSmContextContext)
		info.Response = &models.SmContextReleasedData{
			//TODO: fill the session release information
		}
		smCtx.Infof("Finalize ReleaseSmContext task")
		info.Finalize(nil)
	default:
	}

	if relCtx := smCtx.relCtx; relCtx != nil {
		if relCtx.t3592 != nil {
			relCtx.t3592.Stop()
		}

		for _, ch := range relCtx.doneChs {
			ch <- struct{}{}
		}

		smCtx.relCtx = nil
	} else {
		switch trigger.evType {
		case RELEASE_FORCE, RELEASE_DUP:
			//notify that release has finalized
			if trigger.dat != nil {
				if ch, ok := trigger.dat.(chan struct{}); ok {
					ch <- struct{}{}
				}
			}
		}
	}
}

// force to release then remove the session
func (smCtx *SmContext) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), RELEASE_TIMEOUT)
	defer cancel()
	doneCh := make(chan struct{}, 1)
	event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
		evType: RELEASE_FORCE,
		dat:    doneCh,
	})

	if err := <-smCtx.sendEvent(ctx, event); err != nil {
		smCtx.Errorf("Fail to send ReleaseTriggerEvent to state machine: %+v", err)
		return
	}

	select {
	case <-ctx.Done():
		_pool.remove(smCtx)
	case <-doneCh:
	}
}

// we found an duplicated SmContext while trying to create a new one, thus we need to remove it.
func (smCtx *SmContext) ForceRelease() (err error) {
	smCtx.Infof("Force release a orphan SmContext")

	doneCh := make(chan struct{}, 1)
	event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
		dat:    doneCh,
		evType: RELEASE_DUP,
	})
	if err := <-smCtx.sendEvent(context.TODO(), event); err != nil {
		return utils.WrapError("Send ReleaseTriggerEvent to state machine", err)
	}

	//wait for release to complete
	<-doneCh
	smCtx.Debug("Force release completed")
	return nil
}

// resend N1Sm PdusessionReleaseCommand
func (smCtx *SmContext) sendPduSessionReleaseCommand() (err error) {
	smCtx.Warnf("Re-send N1Sm Session Release Command")
	return smCtx.n1n2.sendN1N2(&N1N2Messages{
		n1Sm: smCtx.relCtx.n1Sm,
	})
	return
}

//ask gnb to release PduSessionResource
func (smCtx *SmContext) releaseGnb() {
	//send PduSessionResourceReleaseCommand
	//determine if Ran resource need to be released
	if smCtx.isN2Deactivatable() {
		smCtx.upCnxState = UPCNXSTATE_DEACTIVATED
		if n2SmInfo, err := smCtx.buildPduSessionResourceReleaseCommandTransfer(); err != nil {
			smCtx.Errorf("Fail to build N2SmInfo PduSessionResourceReleaseCommand", err)
		} else {
			if err := smCtx.n1n2.sendN1N2(&N1N2Messages{
				n2SmInfo:     n2SmInfo,
				n2SmInfoType: models.N2SMINFOTYPE_PDU_RES_REL_CMD,
			}); err != nil {
				smCtx.Errorf("Fail to send PduSessioNResourceReleaseCommandTransfer to gnB: %+v", err)
			} else {
				smCtx.Infof("PduSessioNResourceReleaseCommandTransfer was sent to gnB")
			}
		}
	}
}
