package ue

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"etrib5gc/mesh"
	damfctx "etrib5gc/nfs/damf/context"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/apis/amf/n2nas"
	"github.com/reogac/utils"
)

// forward the InitUeMsg toward the AMF
// Ue in UE_FORWARDING state
func (ueCtx *UeContext) forwardInitUeRequest(ctx context.Context) {
	if rsp, err := n2nas.InitialUeMessage(ueCtx.amfCli, &ueCtx.ranInfo, ueCtx.msg); err != nil {
		ueCtx.Errorf("Fail to forward InitialUeMessage toward AMF: %+v", err)
		//Remove UE context
		ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
			cause: CAUSE_CORE_FAIL,
		}))
	} else {
		ueCtx.Infof("InitialUeMessage is forwarded to an AMF, receive AmfUeId=%d", rsp.AmfUeId)
		//Remove UE context
		ueCtx.state.SetNextEvent(fsm.NewEventData(CloseEvent, &ReleaseContext{
			cause: CAUSE_NORMAL,
		}))

	}
}

func (ueCtx *UeContext) createAmfClient(amfId *nas.AmfId) error {
	//create a Sbi client toward the AMF
	amfSetStr := fmt.Sprintf("%d-%d", amfId.GetRegion(), amfId.GetSet())
	amfService := common.AmfServiceName(damfctx.PlmnId(), amfSetStr)
	amfInsId := common.AmfPointerString(amfId.GetPointer())
	if cli, err := mesh.ConsumerWithInstanceId(amfService, amfInsId); err != nil {
		return utils.WrapError("Create AMF consumer", err)
	} else {
		ueCtx.amfCli = cli
	}
	return nil
}
