package iden

import (
	"context"
	"etrib5gc/common"
	"github.com/reogac/nas"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	T3570_DURATION time.Duration = 5000 //miliseconds
	MAX_T3570_CNT  int           = 4
)

type Context struct {
	Log       *logrus.Entry
	OnT3570   func(*IdentificationProcedure)
	SendNasDl func([]byte) error
	NasCtx    *nas.NasContext
	Callback  func(*nas.MobileIdentity, error)
	IdType    uint8
	IsGpp     bool
}

type IdentificationProcedure struct {
	*logrus.Entry
	idType        uint8
	t3570         common.Timer
	t3570Cnt      int
	callback      func(*nas.MobileIdentity, error)
	sendIdRequest func() error //to send IdentityRequest
}

func Start(ctx *Context) (proc *IdentificationProcedure) {
	proc = &IdentificationProcedure{
		Entry: ctx.Log.WithFields(logrus.Fields{
			"mod": "iden",
		}),
		idType:   ctx.IdType,
		callback: ctx.Callback,
	}
	proc.makeSendIdRequestFunction(ctx.NasCtx, ctx.IsGpp, ctx.SendNasDl)

	proc.t3570 = common.NewTimer(T3570_DURATION*time.Millisecond, func() {
		//IdentityRequest expired
		ctx.OnT3570(proc)
	}, nil)
	proc.requestId()
	return
}

// close all timer
func (proc *IdentificationProcedure) Close() {
	proc.t3570.Stop()
}

func (proc *IdentificationProcedure) ReceiveN1Mm(ctx context.Context, gmm *nas.DecodedGmmMessage) {
	if gmm.MsgType == nas.IdentityResponseMsgType {
		proc.handleIdentityResponse(ctx, gmm.IdentityResponse)
	} else {
		proc.Warnf("Unexpected message in Identification Procedure (msgtype= %d", gmm.MsgType)
	}
}

func (proc *IdentificationProcedure) HandleT3570() {
	proc.Warnf("UE did not send IdentityResponse, T3570 expired")
	if proc.t3570Cnt >= MAX_T3570_CNT {
		proc.callback(nil, ErrNoResponse)
	} else {
		proc.requestId()
	}
}

func (proc *IdentificationProcedure) handleIdentityResponse(ctx context.Context, msg *nas.IdentityResponse) {
	proc.Debug("Receive IdentityResponse from UE")
	//reset timer and its counter
	proc.t3570.Stop()
	proc.t3570Cnt = 0

	id := msg.MobileIdentity
	idType := id.GetType()
	if idType != proc.idType {
		proc.callback(nil, ErrMismatchedIdType)
		return
	}

	proc.callback(&id, nil)
	return
}
func (proc *IdentificationProcedure) requestId() {
	if err := proc.sendIdRequest(); err != nil {
		proc.callback(nil, err)
	} else {
		proc.t3570.Start()
		proc.t3570Cnt++
	}
}
