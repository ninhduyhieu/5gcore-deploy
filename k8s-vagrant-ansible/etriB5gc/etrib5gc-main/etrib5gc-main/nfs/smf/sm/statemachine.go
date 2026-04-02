package sm

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	smfctx "etrib5gc/nfs/smf/context"
	"time"

	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

const (
	SM_INACTIVE     fsm.StateType = iota //Session is not active
	SM_MODIFYING                         //Session is active, asking UE to modify session
	SM_ACTIVE                            //Session is active
	SM_INACTIVATING                      //Waiting UE to release session
	SM_HANDOVER                          //Handover on-going
)

const (
	ActCmplEvent fsm.EventType = fsm.EventIndexStart + iota //activation complete, move to SM_ACTIVE
	ReleaseEvent                                            //deactivate session
	ModEvent
	ModCmplEvent //modification complete, move to SM_ACTIVE
	T3591Event   //N1Sm ModificationCommand expires
	T3592Event   //N1Sm ReleaseCommand expires
	ModificationTriggerEvent
	HandoverEvent
	HandoverExpiredEvent
	HandoverCmplEvent

	ReleaseTriggerEvent //Start release SmContext
	N1SmEvent           //Receive N1SM message (from an SBI request)
	N2SmInfoEvent
	SbiEvent //Receive a SBI request
)

var _sm *fsm.Fsm

func initFsm() {
	if _sm != nil {
		return
	}
	transitions := fsm.Transitions{
		fsm.Tuple(SM_INACTIVE, ActCmplEvent): SM_ACTIVE,

		fsm.Tuple(SM_MODIFYING, ModCmplEvent): SM_ACTIVE,
		fsm.Tuple(SM_MODIFYING, T3591Event):   SM_MODIFYING,

		fsm.Tuple(SM_INACTIVATING, T3592Event): SM_INACTIVATING,

		fsm.Tuple(SM_ACTIVE, ReleaseEvent):             SM_INACTIVATING,
		fsm.Tuple(SM_ACTIVE, ModificationTriggerEvent): SM_ACTIVE,
		fsm.Tuple(SM_ACTIVE, ModEvent):                 SM_MODIFYING,
		fsm.Tuple(SM_ACTIVE, HandoverEvent):            SM_HANDOVER,

		fsm.Tuple(SM_HANDOVER, HandoverCmplEvent):    SM_ACTIVE,
		fsm.Tuple(SM_HANDOVER, HandoverExpiredEvent): SM_ACTIVE,
	}

	callbacks := fsm.Callbacks{
		SM_INACTIVE:     sm_inactive,     // not activated
		SM_ACTIVE:       sm_active,       // activated
		SM_INACTIVATING: sm_inactivating, //inactivating
		SM_MODIFYING:    sm_modifying,    //session modifying
		SM_HANDOVER:     sm_handover,     //session handovering
	}

	commonEvents := []fsm.EventType{
		ReleaseTriggerEvent,
		N1SmEvent,
		N2SmInfoEvent,
		SbiEvent,
	}

	_sm = fsm.NewFsm(fsm.Options{
		Transitions:    transitions,
		Callbacks:      callbacks,
		CommonEvents:   commonEvents,
		CommonCallback: sm_all,
	}, smfctx.MaxNumSmContexts(), 128)
}

// no session, or session is being established
func sm_inactive(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	smCtx, _ := fsm.GetStateOwner[*SmContext](state)
	switch evType {
	case fsm.EntryEvent:
		smCtx.Debug("Enter SM_INACTIVE")
	}
}

// session established
func sm_active(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	smCtx, _ := fsm.GetStateOwner[*SmContext](state)

	switch evType {
	case fsm.EntryEvent:
		smCtx.Debug("Enter SM_ACTIVE state")

	case ModificationTriggerEvent:
		e := payload.(*EventData)
		smCtx.handleModificationTrigger(e)
	}
}

// ask UE to modify session (PduSessionModificationCommand to UE is sent to UE)
// invariant: smCtx.modCtx must be non nil. It must be reset to nil when exit
func sm_modifying(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	smCtx, _ := fsm.GetStateOwner[*SmContext](state)
	switch evType {
	case fsm.EntryEvent:
		smCtx.Debug("Enter SM_MODIFYING state")
		smCtx.modCtx = payload.(*ModificationContext)
		smCtx.modCtx.sendPduSessionModificationCommand()

	case fsm.ExitEvent:
		smCtx.modCtx.stop()
		smCtx.modCtx = nil

	case T3591Event:
		//try re-send PduSessionModificationCommand a few time before go back to
		//SM_ACTIVE
		smCtx.modCtx.t3591cnt++
		if smCtx.modCtx.t3591cnt >= MAX_T3591_CNT {
			smCtx.Errorf("No response for N1SM PduSessionModificationCommand")
			//modification fails, go back to SM_ACTIVE
			smCtx.state.SetNextEvent(fsm.NewEmptyEventData(ModCmplEvent))
		} else {
			smCtx.modCtx.sendPduSessionModificationCommand()
		}

	}
}

func sm_handover(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	smCtx, _ := fsm.GetStateOwner[*SmContext](state)
	switch evType {
	case fsm.EntryEvent:
		smCtx.Debug("Enter SM_HANDOVER state")
		hoCtx := payload.(*HandoverContext)
		smCtx.hoCtx = hoCtx
		var err error
		var pdu []byte
		if pdu, err = smCtx.buildPduSessionResourceSetupRequestTransfer(); err != nil {
			err = utils.WrapError("Build N2SmInfo PduSessionResourceSetupRequestTransfer", err)
		} else if err = smCtx.sendN2SmInfo(pdu, models.N2SMINFOTYPE_PDU_RES_SETUP_REQ); err != nil {
			err = utils.WrapError("Send N2SmInfo PduSessionResourceSetupRequestTransfer", err)
		}
		if err != nil { //fail to notify target Ran
			smCtx.Errorf(err.Error())
			smCtx.cancelHandover(hoCtx)
			//move back to SM_ACTIVE
			event := fsm.NewEventData(HandoverCmplEvent, hoCtx)
			smCtx.state.SetNextEvent(event)
		} else { //create a handover timer
			smCtx.hoCtx.hoTimer = common.NewTimer(HO_TIMER*time.Millisecond, func() {
				smCtx.Warnf("Handover expired")
				smCtx.sendEvent(context.TODO(), fsm.NewEmptyEventData(HandoverExpiredEvent))
			}, nil)

		}
	case HandoverExpiredEvent:
		//cancel handover
		smCtx.cancelHandover(smCtx.hoCtx)

	case fsm.ExitEvent:
		smCtx.hoCtx = nil
	}
}

// wait for UE to release session (PduSessionReleaseCommand is sent to UE)
// invariant 1: smCtx.relCtx must be non nil. It must be reset to nil when exit
// invariant 2: release procedure must be finalized when exit
func sm_inactivating(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	smCtx, _ := fsm.GetStateOwner[*SmContext](state)
	switch evType {
	case fsm.EntryEvent:
		//attach release context
		smCtx.Debug("Enter SM_INACTIVATING")
		smCtx.relCtx = payload.(*ReleaseContext)

	case T3592Event:
		if smCtx.relCtx.t3592cnt >= MAX_T3592_CNT {
			smCtx.Errorf("No response for N1SM PduSessionReleaseCommand")
			smCtx.finalizeRelease(smCtx.relCtx.trigger)
		} else {
			if err := smCtx.sendPduSessionReleaseCommand(); err != nil {
				smCtx.Errorf("Fail to send N1SM PduSessionReleaseCommand: %+v", err)
				smCtx.finalizeRelease(smCtx.relCtx.trigger)
			} else {
				smCtx.relCtx.t3592cnt++
				smCtx.relCtx.t3592.Start()
			}
		}
	}
}

// common callback for non-transitional events
func sm_all(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	smCtx, _ := fsm.GetStateOwner[*SmContext](state)
	switch evType {
	case ReleaseTriggerEvent:
		info := payload.(*EventData)
		smCtx.handleReleaseTrigger(info)

	case SbiEvent:
		info := payload.(*EventData)
		smCtx.handleSbiEvent(ctx, info)

	case N1SmEvent:
		msg := payload.(*nas.DecodedGsmMessage)
		smCtx.handleN1Sm(msg)

	case N2SmInfoEvent:
		info, _ := payload.(*N2SmInfo)
		smCtx.handleN2SmInfo(info.infoType, info.dat)
	}
}

func stateString(s fsm.StateType) string {
	switch s {
	case SM_INACTIVE:
		return "SM_INACTIVE"
	case SM_MODIFYING:
		return "SM_MODIFYING"
	case SM_ACTIVE:
		return "SM_ACTIVE"
	case SM_INACTIVATING:
		return "SM_INACTIVATING"
	case SM_HANDOVER:
		return "SM_HANDOVER"
	}
	return "SM_UNKNOWN"
}
