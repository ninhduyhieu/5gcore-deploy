package secmode

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/nfs/amf/secctx"
	"fmt"
	"time"

	"github.com/reogac/nas"
	//"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

const (
	T3560_DURATION time.Duration = 50000 //miliseconds
	MAX_T3560_CNT  int           = 4
)

type Context struct {
	Log       *logrus.Entry
	SendNasDl func([]byte) error
	Callback  func(*Result, error)
	SecCtx    *secctx.SecurityContext
	Builder   func(*nas.SecurityModeCommand) error
	IsGpp     bool
	OnT3560   func(*SecmodeProcedure)
}

type Result struct {
	Id        *nas.MobileIdentity
	SecCtx    *secctx.SecurityContext
	Container []byte
}

type SecmodeProcedure struct {
	*logrus.Entry
	t3560     common.Timer
	t3560Cnt  int
	callback  func(*Result, error)
	sendNasDl func([]byte) error
	secCtx    *secctx.SecurityContext
	builder   func(*nas.SecurityModeCommand) error
	isGpp     bool
}

// create and start the procedure
func Start(ctx *Context) (proc *SecmodeProcedure) {
	proc = &SecmodeProcedure{
		isGpp:     ctx.IsGpp,
		builder:   ctx.Builder,
		secCtx:    ctx.SecCtx,
		sendNasDl: ctx.SendNasDl,
		callback:  ctx.Callback,
		Entry: ctx.Log.WithFields(logrus.Fields{
			"mod": "secmod",
		}),
	}
	proc.t3560 = common.NewTimer(T3560_DURATION*time.Millisecond, func() {
		ctx.OnT3560(proc)
	}, nil)

	proc.sendSecmodeCommand()
	return
}

// close all timer
func (proc *SecmodeProcedure) Close() {
	proc.t3560.Stop()
}

// handle received N1Mm messages
func (proc *SecmodeProcedure) ReceiveN1Mm(ctx context.Context, gmm *nas.DecodedGmmMessage) {
	switch gmm.MsgType {
	case nas.SecurityModeCompleteMsgType:
		proc.handleSecurityModeComplete(ctx, gmm.SecurityModeComplete)
	case nas.SecurityModeRejectMsgType:
		proc.handleSecurityModeReject(ctx, gmm.SecurityModeReject)
	default: //ignore
		proc.Warnf("Unknown Nas message to security mode establishment")
	}
	return
}

// handle T3560 timer expiration
func (proc *SecmodeProcedure) HandleT3560() {
	proc.Warn("T3560 expired")
	if proc.t3560Cnt >= MAX_T3560_CNT {
		//stop security mode establishment procedure
		proc.callback(nil, ErrNoResponse)
	} else {
		//re-send security mode command
		proc.sendSecmodeCommand()
	}
}

func (proc *SecmodeProcedure) handleSecurityModeComplete(ctx context.Context, msg *nas.SecurityModeComplete) {
	proc.t3560.Stop()
	proc.t3560Cnt = 0
	proc.callback(&Result{
		SecCtx:    proc.secCtx,
		Container: msg.NasMessageContainer,
		Id:        msg.Imeisv,
	}, nil)
}

func (proc *SecmodeProcedure) handleSecurityModeReject(ctx context.Context, msg *nas.SecurityModeReject) {
	proc.Debug("Handle NAS SecurityModeReject")
	proc.t3560.Stop()
	proc.t3560Cnt = 0

	cause := msg.GmmCause
	proc.callback(nil, fmt.Errorf("UE reject the security mode command:%s", nas.Cause5GMMToString(cause)))

}
