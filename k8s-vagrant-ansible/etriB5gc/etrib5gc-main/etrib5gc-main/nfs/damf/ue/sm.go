package ue

import (
	"context"
	"etrib5gc/internal/fsm"
	"etrib5gc/nfs/amf/procs/auth"
	"etrib5gc/nfs/amf/procs/iden"
	damfctx "etrib5gc/nfs/damf/context"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/apis/pran/uectx"
	"github.com/reogac/sbi/models"
)

const (
	UE_AUTHENTICATING fsm.StateType = iota //authenticating
	UE_FORWARDING                          //authentication completed, forward InitUeMsg toward an AMF
	UE_REMOVING                            //clean up to remove UeContext from the pool
)

const (
	AuthEvent      fsm.EventType = fsm.EventIndexStart + iota //start authentication
	AuthTimerEvent                                            //T3560 timer expires
	IdenTimerEvent                                            //T3570 timer expires
	NasUlEvent                                                //receive a Nas Uplink message from PRAN
	NasErrEvent                                               //receive a Nas Error Message from PRAN
	CloseEvent                                                //force close
	ForwardEvent                                              //start forwarding InitUeMsg toward a designated AMF
)

// a single finite state machine shared by all UeContexts
var _sm *fsm.Fsm

// singleton to create a finite state machine for UeContext
func initFsm() {
	if _sm == nil {
		transitions := fsm.Transitions{
			fsm.Tuple(UE_AUTHENTICATING, AuthEvent):      UE_AUTHENTICATING,
			fsm.Tuple(UE_AUTHENTICATING, AuthTimerEvent): UE_AUTHENTICATING,
			fsm.Tuple(UE_AUTHENTICATING, IdenTimerEvent): UE_AUTHENTICATING,
			fsm.Tuple(UE_AUTHENTICATING, NasUlEvent):     UE_AUTHENTICATING,
			fsm.Tuple(UE_AUTHENTICATING, NasErrEvent):    UE_AUTHENTICATING,
			fsm.Tuple(UE_AUTHENTICATING, CloseEvent):     UE_REMOVING,
			fsm.Tuple(UE_AUTHENTICATING, ForwardEvent):   UE_FORWARDING,

			fsm.Tuple(UE_FORWARDING, CloseEvent): UE_REMOVING,

			fsm.Tuple(UE_REMOVING, CloseEvent): UE_REMOVING,
		}

		callbacks := fsm.Callbacks{
			UE_REMOVING:       ue_removing,
			UE_FORWARDING:     ue_forwarding,
			UE_AUTHENTICATING: ue_authenticating,
		}

		_sm = fsm.NewFsm(fsm.Options{
			Transitions: transitions,
			Callbacks:   callbacks,
		}, damfctx.MaxNumUes(), 128)
	}
}

func ue_authenticating(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	switch evType {
	case AuthEvent:
		ueCtx.handleAttachment(ctx)

	case AuthTimerEvent:
		proc, _ := payload.(*auth.AuthProc)
		proc.HandleT3560()

	case IdenTimerEvent:
		proc, _ := payload.(*iden.IdentificationProcedure)
		proc.HandleT3570()

	case NasUlEvent:
		n2msg, _ := payload.(*models.NasUplinkTransport)
		if nasMsg, err := nas.Decode(nil, n2msg.NasPdu, ueCtx.isGpp); err != nil {
			ueCtx.Warnf("Fail to decode a nas uplink message: %+v", err)
		} else {
			if nasMsg.Gmm == nil {
				ueCtx.Warnf("N1Mm in decoded Nas message is empty")
			} else {
				ueCtx.proc.ReceiveN1Mm(ctx, nasMsg.Gmm)
			}
		}

	case NasErrEvent:
		//TODO: handle nas non delivery indication
		//Note: UeContext will be removed anyway (due to timeout) if authentication and AMF finding procedures do not finish.
		ueCtx.Warnf("Receive NasErrMsg from PRan")

	case CloseEvent:
		if ueCtx.proc != nil {
			ueCtx.proc.Close()
		}
	}

}

func ue_forwarding(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	switch evType {
	case fsm.EntryEvent:
		ueCtx.Debugf("IN FORWARD")
		if ueCtx.amfCli != nil { //amf found with GUTI
			//forward initial Ue message
			ueCtx.forwardInitUeRequest(ctx) //move to removing state after forwarding
			return
		}

		//get AmData from UDM
		if err := ueCtx.getAmData(); err != nil {
			ueCtx.Errorf("Fail to get AmData from UDM: %+v", err)
			//fail to get AmData for Ue
			ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
				cause: CAUSE_CORE_FAIL,
			}))
		}

		//get AMF by asking NSSF
		if err := ueCtx.findAmf(); err != nil {
			//fail to get AMF, go to removing state
			ueCtx.Errorf("Fail to create AMF consumer: %+v", err)
			ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
				cause: CAUSE_CORE_FAIL,
			}))
		} else {
			//forward initial Ue message
			ueCtx.forwardInitUeRequest(ctx) //move to removing state after forwarding
		}
	}
}

// in removing state
func ue_removing(ctx context.Context, state *fsm.State, evType fsm.EventType, payload any) {
	ueCtx, _ := fsm.GetStateOwner[*UeContext](state)
	switch evType {
	case fsm.EntryEvent:
		ueCtx.Debugf("IN REMOVING")
		_pool.removeUe(ueCtx) //remove UeContext from the UE context pool
		ueCtx.Infof("Ue is removed")
		relCtx, _ := payload.(*ReleaseContext)

		if relCtx.cause != CAUSE_NORMAL {
			switch ueCtx.gmm.MsgType {
			case nas.RegistrationRequestMsgType:
				//send a registration reject
				if relCtx.cause != CAUSE_AUTH_RJT {
					var eap []byte
					if authCtx := ueCtx.msg.AuthCtx; authCtx != nil {
						eap = authCtx.Eap
					}
					if err := ueCtx.sendRegistrationReject(toNasMmCause(relCtx.cause), eap); err != nil {
						ueCtx.Errorf("Fail to send registration reject: %+v", err)
					}
				}

			case nas.ServiceRequestMsgType:
				//send service reject
				if err := ueCtx.sendServiceReject(toNasMmCause(relCtx.cause)); err != nil {
					ueCtx.Errorf("Fail to send service reject: %+v", err)
				}
			default: //do nothing
			}
			//ask RAN to release UeContext
			if _, err := uectx.UeContextRelease(ueCtx.ranCli, ueCtx.ranUeId.Id, &models.UeContextReleaseCommand{
				Cause: models.N2Cause{
					//TODO: assign cause
				},
			}); err == nil {
				ueCtx.Infof("UeContextReleaseCommand was sent to gnB, receive a UeContextReleaseResponse")
			} else {
				ueCtx.Errorf("Fail to send UeContextRelease to gnB: %+v", err)
			}
		}

		if relCtx.doneCh != nil {
			relCtx.doneCh <- struct{}{}
		}

	case CloseEvent:
		relCtx, _ := payload.(*ReleaseContext)
		if relCtx.doneCh != nil {
			relCtx.doneCh <- struct{}{}
		}
	}
}
