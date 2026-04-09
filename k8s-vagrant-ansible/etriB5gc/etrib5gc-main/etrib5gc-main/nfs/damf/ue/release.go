package ue

import (
	"context"
	"etrib5gc/internal/fsm"
	"github.com/reogac/nas"
	"time"
)

const (
	CAUSE_ID_FAIL   uint8 = iota // identification failure
	CAUSE_AUTH_FAIL              // authentication failure, not reject yet
	CAUSE_AUTH_RJT               // authentication was rejected
	CAUSE_CORE_FAIL              // fail to connect to other core NFs
	CAUSE_N1MM                   // unexpected intitial nas message
	CAUSE_NORMAL                 // success
	CAUSE_FORCE                  // DAMF is terminated
	CAUSE_NO_AMFID               // can't get AmfId from service request/deregistration request
)

type ReleaseContext struct {
	doneCh chan struct{}
	cause  uint8
}

// send a close event to release all on-going procedures of UeContext
func (ueCtx *UeContext) kill() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	doneCh := make(chan struct{}, 1)
	ueCtx.sendEvent(ctx, fsm.NewEventData(CloseEvent, &ReleaseContext{
		doneCh: doneCh,
		cause:  CAUSE_FORCE,
	}))
	select {
	case <-doneCh:
	case <-ctx.Done():
	}
}

func toNasMmCause(cause uint8) uint8 {
	switch cause {
	case CAUSE_ID_FAIL:
		return nas.Cause5GMMUEIdentityCannotBeDerivedByTheNetwork
	case CAUSE_FORCE:
		return nas.Cause5GMMImplicitlyDeregistered
	default:
		return nas.Cause5GMMProtocolErrorUnspecified
	}
}
