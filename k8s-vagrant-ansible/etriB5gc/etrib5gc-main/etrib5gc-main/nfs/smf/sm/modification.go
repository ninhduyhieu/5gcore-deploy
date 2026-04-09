package sm

import "etrib5gc/common"

const (
	MODIFY_N1SM         uint8 = iota //UE request
	MODIFY_RAN_NOTIFY                //RAN notify for modification command
	MODIFY_RAN_INDICATE              //RAN request for modification
	MODIFY_LOCAL                     //SMF (PCF) request
)

type ModificationContext struct {
	trigger  *EventData
	smCtx    *SmContext
	pti      *uint8       //PTI from N1Sm Modification Request, if any
	t3591    common.Timer //waiting for response to Modification Command
	t3591cnt int          //max is 4
	t3593    common.Timer //waiting for next Release Request from UE
}

func (smCtx *SmContext) createModificationContext(e *EventData) *ModificationContext {
	ctx := &ModificationContext{
		smCtx:   smCtx,
		trigger: e,
	}
	return ctx
}

func (ctx *ModificationContext) stop() {
	if ctx.t3591 != nil {
		ctx.t3591.Stop()
	}
	if ctx.t3593 != nil {
		ctx.t3593.Stop()
	}
}

func (ctx *ModificationContext) sendPduSessionModificationCommand() {
	//TODO: send a N1Sm Modification Command
}

func (smCtx *SmContext) handleModificationTrigger(e *EventData) {
	//1. get modification info
	//2. get policy decision from PCF
	//3. modify tunnel + QoS
	//4. update UPF if neccessary
	//5. send modification command
}
