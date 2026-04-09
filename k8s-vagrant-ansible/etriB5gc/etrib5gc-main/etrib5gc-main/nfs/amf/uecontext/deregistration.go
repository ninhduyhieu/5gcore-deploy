package uecontext

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/sm"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
)

const (
	TARGET_GPP uint8 = iota
	TARGET_NON_GPP
	TARGET_BOTH
)

const (
	T3522_MAX_CNT uint8 = 1
)

type DeregistrationTrigger struct {
	doneCh    chan struct{}
	targetRan uint8
	retry     bool //re-register after deregistration
}

type DeregistrationContext struct {
	trigger *DeregistrationTrigger

	ranUe    RanUe        //RanUe to execute deregistration procedure
	t3522    common.Timer //for sending deregistration request
	t3522Cnt uint8
}

// an external trigger releases UeContext
func (ueCtx *UeContext) kill() {
	doneCh := make(chan struct{}, 1)
	trigger := &DeregistrationTrigger{
		doneCh:    doneCh,
		targetRan: TARGET_BOTH,
		retry:     false,
	}
	ueCtx.sendEvent(context.TODO(), fsm.NewEventData(DeregisterEvent, trigger))

	<-doneCh
}

// mark UE as deregistered
func (ueCtx *UeContext) setDeregistered(targetRan uint8) {
	if targetRan == TARGET_GPP || targetRan == TARGET_BOTH {
		ueCtx.setRegistrationStatus(true, false)
	}

	if targetRan == TARGET_NON_GPP || targetRan == TARGET_BOTH {
		ueCtx.setRegistrationStatus(false, false)
	}
}

func (ueCtx *UeContext) handleDeregistrationRequest(ctx context.Context, isGpp bool, msg *nas.DeregistrationRequestFromUe, macFailed bool) {
	//TODO: handle mac fail
	ueCtx.Debugf("Receive NAS DeregistrationRequestFromUe")

	if ueCtx.deregCtx != nil {
		ueCtx.Warnf("Network initiated deregistration on-going")
		//there was an on-going network initiated deregistration procedure
		//do nothing, just wait for the original deregistration procedure
		//to complete
	} else {
		//1. get target RAN for deregistration
		targetRan := TARGET_BOTH
		switch msg.DeRegistrationType.GetAccessType() {
		case nas.AccessType3GPP:
			targetRan = TARGET_GPP
		case nas.AccessTypeNon3GPP:
			targetRan = TARGET_NON_GPP
		default:
		}

		//2. mark UE registration status
		ueCtx.setDeregistered(targetRan)

		//3. release pdu sessions and notify PCF/UDM
		ueCtx.notifyDeregistration(targetRan)

		//4. send deregistration accept
		if !msg.DeRegistrationType.GetSwitchOff() {
			if err := ueCtx.sendDeregistrationAccept(isGpp); err != nil {
				ueCtx.Errorf("Fail to send DeregistrationAccept: %+v", err)
			}
		}
		//5. release UeContext at GnB
		ueCtx.releaseUeContext(targetRan, models.N2Cause{})

		//6. remove Ue if both access has been released
		if ueCtx.gpp.ranUe == nil && ueCtx.nonGpp.ranUe == nil {
			_uePool.remove(ueCtx)
		}
	}
}

// network trigger deregistration
func (ueCtx *UeContext) handleDeregistrationTrigger(trigger *DeregistrationTrigger) {
	//TODO: if UE is in attachment procedure -> abort
	//TODO: if UE is in handover procedure -> cancel handover then go to
	//deregistration procedure

	if ueCtx.deregCtx != nil {
		//there is an on-going deregistration procedure
		//TODO:
		ueCtx.Warnf("There is an on-going deregistration")
		if trigger.doneCh != nil {
			trigger.doneCh <- struct{}{}
		}
	} else { //start a new network initiated deregistration procedure

		//1. mark UE registration status
		ueCtx.setDeregistered(trigger.targetRan)

		//2. release Pdu sessions and notify PCF/UDM of deregistration
		ueCtx.notifyDeregistration(trigger.targetRan)

		//4. select RanUe to send deregistration request
		var ranUe RanUe
		if trigger.targetRan == TARGET_GPP || trigger.targetRan == TARGET_BOTH {
			ranUe = ueCtx.gpp.ranUe
		}

		if ranUe == nil && (trigger.targetRan == TARGET_NON_GPP || trigger.targetRan == TARGET_BOTH) {
			ranUe = ueCtx.nonGpp.ranUe
		}

		//5. create deregistration context then move to MM_DEREGISTERING
		deregCtx := &DeregistrationContext{
			trigger: trigger,
			ranUe:   ranUe,
			t3522: common.NewTimer(amfctx.T3522(), func() {
				ueCtx.Warnf("Wating DeregistrationAccept timeout")
				ueCtx.sendEvent(context.Background(), fsm.NewEmptyEventData(DeregTimerEvent))
			}, nil),
		}
		ueCtx.state.SetNextEvent(fsm.NewEventData(DeregisterTriggeredEvent, deregCtx))
	}
}

// try to deregister UE
func (ueCtx *UeContext) deregisterUe() {
	if ueCtx.deregCtx.ranUe == nil {
		//no access
		ueCtx.completeDeregistration()
	} else if err := ueCtx.sendDeregistrationRequest(); err != nil {
		//complete deregistration now
		ueCtx.Errorf("Fail to send DeregistrationRequest: %+v", err)
		ueCtx.completeDeregistration()
	}
}

// handle timeout for waiting the deregistration accept from UE
func (ueCtx *UeContext) handleT3522() {
	ctx := ueCtx.deregCtx

	if ctx.t3522Cnt >= T3522_MAX_CNT { //tired now, finalize deregistration
		ueCtx.Warnf("UE did not send DeregistrationAccept")
		ueCtx.completeDeregistration()
	} else {
		//try deregister Ue again
		ueCtx.deregisterUe()
	}
}

func (ueCtx *UeContext) handleDeregistrationAccept(ctx context.Context, isGpp bool, msg *nas.DeregistrationAcceptFromUe) {
	ueCtx.Debugf("Handle NAS DeregistrationAccept from UE")
	if ueCtx.deregCtx != nil {
		ueCtx.deregCtx.t3522.Stop()

		ueCtx.completeDeregistration()
	}
}

// release Pdu Sessions and notify PCF/UDM of deregistration
func (ueCtx *UeContext) notifyDeregistration(targetRan uint8) {
	//1. release sessions
	smCtxs := []*sm.SmContext{}
	switch targetRan {
	case TARGET_BOTH:
		smCtxs = ueCtx.sessions.listAll()
	case TARGET_GPP:
		smCtxs = ueCtx.sessions.getSessionsForAccess(true)
	case TARGET_NON_GPP:
		smCtxs = ueCtx.sessions.getSessionsForAccess(false)
	}

	tasks := []func(){}
	for _, smCtx := range smCtxs {
		tasks = append(tasks, func() {
			ueCtx.releaseSmContext(smCtx, "")
		})
	}
	//2. terminate AmPol with PCF if any
	if !ueCtx.gpp.registered && !ueCtx.nonGpp.registered {
		tasks = append(tasks, func() {
			ueCtx.terminateAmPol()
		})
	}

	//3. deregister with UDM
	if !ueCtx.gpp.registered && !ueCtx.nonGpp.registered {
		tasks = append(tasks, func() {
			ueCtx.notifyUdmOfDeregistration()
		})
	}
	executeTasks(tasks)
}

// in MM_DEREGISTERING, after receive DeregistrationAccept or max timeouts
func (ueCtx *UeContext) completeDeregistration() {
	deregCtx := ueCtx.deregCtx

	if !deregCtx.trigger.retry { // no re-registration
		ueCtx.releaseUeContext(deregCtx.trigger.targetRan, models.N2Cause{})
		_uePool.remove(ueCtx)
	} else { // ue migh register again
		//TODO: a mechanism to remove UeContext if Ue does not register again
		if ueCtx.gpp.ranUe == nil && ueCtx.nonGpp.ranUe == nil { //both access has been released -> delete security context
			ueCtx.deleteSecurityContext()
		}
		//MM_IDLE
		ueCtx.state.SetNextEvent(fsm.NewEmptyEventData(DeregisteredEvent))
	}

	if deregCtx.trigger.doneCh != nil {
		deregCtx.trigger.doneCh <- struct{}{}
	}

	ueCtx.Infof("Deregistration completed")
}
