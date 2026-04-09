package uecontext

import (
	"context"
	//"etrib5gc/internal/eventmux"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/procs/auth"
	"etrib5gc/nfs/amf/procs/iden"
	"etrib5gc/nfs/amf/procs/secmode"
	"github.com/reogac/nas"
	//	"time"
)

const (
	MM_IDLE          fsm.StateType = iota //UE not in any procedure
	MM_REGISTERING                        //UE is in a registration procedure
	MM_DEREGISTERING                      //UE is in a deregistration procedure
)

const (
	RegisterEvent     fsm.EventType = fsm.EventIndexStart + iota //entering MM_REGISTERING
	RegisterFailEvent                                            //any failure go back to MM_DEREGISTERED
	RegisterDoneEvent                                            //registration completed

	AttachEvent
	N1MmEvent
	N1ErrEvent

	ReleaseEvent      //deregistration triggred
	DeregisteredEvent //deregistration completed

	DeregisterEvent          // network trigger for deregistration
	DeregisterTriggeredEvent //enter deregistering

	RegCmplTimerEvent       // registration complete timer expires
	CfgUpdateCmplTimerEvent // configuration update complete timer expires
	SecmodeTimerEvent       // security mode command timer expires
	AuthTimerEvent          // authentication request timer expires
	IdenTimerEvent          // identification request expires expires
	T3513Event              //paging timer expires
	DeregTimerEvent         //T3522 deregistration accept timer expires
	T3565Event              //notification response timer expires

	ReleaseContextEvent //oam UeContext release
)

var _sm *fsm.Fsm

// call once to initizlize state machine
func initFsm() *fsm.Fsm {
	if _sm == nil {
		transitions := fsm.Transitions{
			fsm.Tuple(MM_IDLE, RegisterEvent):            MM_REGISTERING,
			fsm.Tuple(MM_IDLE, ReleaseEvent):             MM_DEREGISTERING,
			fsm.Tuple(MM_IDLE, DeregisterTriggeredEvent): MM_DEREGISTERING,
			fsm.Tuple(MM_IDLE, AttachEvent):              MM_IDLE,
			fsm.Tuple(MM_IDLE, CfgUpdateCmplTimerEvent):  MM_IDLE,

			fsm.Tuple(MM_REGISTERING, RegisterDoneEvent): MM_IDLE,
			fsm.Tuple(MM_REGISTERING, RegisterFailEvent): MM_IDLE,
			fsm.Tuple(MM_REGISTERING, ReleaseEvent):      MM_DEREGISTERING,
			fsm.Tuple(MM_REGISTERING, AuthTimerEvent):    MM_REGISTERING,
			fsm.Tuple(MM_REGISTERING, SecmodeTimerEvent): MM_REGISTERING,
			fsm.Tuple(MM_REGISTERING, IdenTimerEvent):    MM_REGISTERING,
			fsm.Tuple(MM_REGISTERING, RegCmplTimerEvent): MM_REGISTERING,

			fsm.Tuple(MM_DEREGISTERING, DeregisteredEvent): MM_IDLE,
			fsm.Tuple(MM_DEREGISTERING, DeregTimerEvent):   MM_DEREGISTERING,
		}

		callbacks := fsm.Callbacks{
			MM_IDLE:          mm_idle,
			MM_REGISTERING:   mm_registering,
			MM_DEREGISTERING: mm_deregistering,
		}

		commonEvents := []fsm.EventType{
			//N1N2Event,
			N1MmEvent,
			N1ErrEvent,
			ReleaseContextEvent,
			DeregisterEvent,
		}

		_sm = fsm.NewFsm(fsm.Options{
			Transitions:    transitions,
			Callbacks:      callbacks,
			CommonEvents:   commonEvents,
			CommonCallback: commonHandler,
		}, amfctx.MaxNumUes(), 128)
	}
	return _sm
}

type RanSbiEventData struct {
	isGpp bool
	evDat any
}

// send an event to the UeContext state machine
func (ueCtx *UeContext) sendEvent(ctx context.Context, ev *fsm.EventData) chan error {
	return _sm.SendEvent(ctx, ueCtx.state, ev)
}

func mm_idle(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	switch evType {
	case fsm.EntryEvent:
		ueCtx.Trace("Enter MM_IDLE")
		if payload != nil {
			//send configuration update command to UE
			if flags, ok := payload.(*ConfigUpdateFlags); ok {
				ctx := &ConfigUpdateContext{
					flags: flags,
				}
				ctx.t3555 = common.NewTimer(amfctx.T3555(), func() {
					//Timer waiting for RegistrationComplete expired
					ueCtx.sendEvent(context.TODO(), fsm.NewEmptyEventData(CfgUpdateCmplTimerEvent))
				}, nil)

				ueCtx.configCtx = ctx
				if err := ueCtx.sendConfigurationUpdateCommand(); err != nil {
					ueCtx.Errorf("Fail to send ConfigurationUpdateCommand: %+v", err)
					ueCtx.configCtx = nil
					//TODO: trigger deregistration
				}
			}
		}

	case CfgUpdateCmplTimerEvent: //timer to wait for ConfigurationUpdateComplete expires
		ueCtx.handleT3555()

	case AttachEvent:
		info, _ := payload.(*AttachmentInfo)
		ueCtx.handleAttachment(ctx, info)

	case T3565Event: //notification time out
	default:
	}
}

func mm_registering(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	attCtx := ueCtx.attCtx

	switch evType {
	case fsm.EntryEvent:
		ueCtx.Tracef("Enter MM_REGISTERING")
		if payload != nil {
			ranUe, _ := payload.(RanUe)
			//bind RanUe to the UeContext (and remove old one)
			ueCtx.attachRanUe(ranUe)
			//ranUe.metrics.regStart(time.Now() /*TODO*/)
		}

		//go to next common procedure
		ueCtx.goNextRegistrationStep()

	case RegCmplTimerEvent: //timer to wait for RegistrationComplete expires
		ueCtx.handleT3550()

	case AuthTimerEvent:
		proc := payload.(*auth.AuthProc)
		proc.HandleT3560()

	case SecmodeTimerEvent:
		proc := payload.(*secmode.SecmodeProcedure)
		proc.HandleT3560()

	case IdenTimerEvent:
		proc := payload.(*iden.IdentificationProcedure)
		proc.HandleT3570()

	case RegisterDoneEvent: //mark Ue is registered on current access
		//ranUe := ueCtx.getRanUe(attCtx.isGpp)
		//ranUe.metrics.regComplete()
		ueCtx.setRegistrationStatus(attCtx.isGpp, true)

	case RegisterFailEvent:
		// //ueCtx.setRegistrationStatus(attCtx.isGpp, false)

	case fsm.ExitEvent:
		if attCtx.registrationRequest != nil {
			attCtx.t3550.Stop() //stop timer
		}
		ueCtx.attCtx = nil

	default:
	}
}

func mm_deregistering(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	switch evType {
	case fsm.EntryEvent:
		ueCtx.Trace("Enter MM_DEREGISTERING")
		ueCtx.deregCtx, _ = payload.(*DeregistrationContext)
		ueCtx.deregisterUe()

	case DeregTimerEvent:
		ueCtx.handleT3522()

	case DeregisteredEvent:
		ueCtx.deregCtx = nil
	}
}

// handle registered non-transitional events
func commonHandler(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	switch evType {
	case N1MmEvent:
		e := payload.(*RanSbiEventData)
		msg, _ := e.evDat.(*nas.DecodedGmmMessage)
		ueCtx.handleN1Mm(ctx, e.isGpp, msg, nil)

	case N1ErrEvent:
		ueCtx.Warnf("NasErr not handled")

	case ReleaseContextEvent:
		isGpp := payload.(bool)
		ueCtx.releaseUeContextForAccess(ctx, isGpp)

	case DeregisterEvent:
		trigger := payload.(*DeregistrationTrigger)
		ueCtx.handleDeregistrationTrigger(trigger)

	}
}
