package ue

import (
	"context"
	"etrib5gc/internal/eventmux"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"time"
)

const (
	SBI_TIMEOUT time.Duration = 5000 * time.Millisecond //Millisecond
)

type UeCtxSetupContext struct {
	Request  *models.UeContextSetupRequest
	Response *models.UeContextSetupResponse
	Failure  *models.UeContextSetupFailure
	*eventmux.AsyncTask
}

func CreateUeCtxSetupContext(req *models.UeContextSetupRequest) *UeCtxSetupContext {
	return &UeCtxSetupContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

type UeCtxModifyContext struct {
	Request  *models.UeContextModifyRequest
	Response *models.UeContextModifyResponse
	Failure  *models.UeContextModifyFailure
	*eventmux.AsyncTask
}

func CreateUeCtxModifyContext(req *models.UeContextModifyRequest) *UeCtxModifyContext {
	return &UeCtxModifyContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

type UeCtxReleaseContext struct {
	Command  *models.UeContextReleaseCommand
	Complete *models.UeContextReleaseComplete
	*eventmux.AsyncTask
}

func CreateUeCtxReleaseContext(cmd *models.UeContextReleaseCommand) *UeCtxReleaseContext {
	return &UeCtxReleaseContext{
		Command:   cmd,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

func ReceiveAsyncSbiEvent[T any, PT interface {
	PtrTo[T]
	LongEvent
}](ueCtx *UeContext, evType uint8, dat PT) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), SBI_TIMEOUT)
	defer cancel()

	evData := eventmux.NewEventData(evType, dat)
	ev := eventmux.NewEventData(SbiEvent, evData)
	if err := _exec.Send(ctx, ueCtx.execSlot, ev); err != nil {
		ueCtx.Errorf("Fail to send event: %+v", err)
	}

	select {
	case <-ctx.Done():
		dat.Finalize(ctx.Err())
		return
	case err = <-dat.Wait():
		if err != nil {
			return utils.WrapError("Handle SbiEvent", err)
			/*
				if evType == UECTX_SETUP {
					//fail to create UeContext, clear UeContext
					ueCtx.clean(false)
				} else if evType == HANDOVER_REQUEST {
					//fail to handle handover request, clear UeContext
					ueCtx.clean(false)
				}
			*/
		}
	}
	return
}

func ReceiveSbiEvent[T any, PT PtrTo[T]](ueCtx *UeContext, evType uint8, dat PT) {
	evData := eventmux.NewEventData[T](evType, dat)
	ev := eventmux.NewEventData(SbiEvent, evData)
	if err := _exec.Send(context.Background(), ueCtx.execSlot, ev); err != nil {
		ueCtx.Errorf("Fail to send event: %+v", err)
	}
}

// handler of the SbiEvent that will be called by the state machine
func (ueCtx *UeContext) handleSbiEvent(ctx context.Context, e *eventmux.EventData) {
	switch e.Type() {
	case PDU_SETUP:
		task := eventmux.GetEventData[SessionResourceSetupContext](e)
		ueCtx.createSessionResource(task)
	case PDU_MODIFY:
		task := eventmux.GetEventData[SessionResourceModifyContext](e)
		ueCtx.modifySessionResource(task)
	case PDU_RELEASE:
		task := eventmux.GetEventData[SessionResourceReleaseContext](e)
		ueCtx.releaseSessionResource(task)

	case N2SMINFO_DOWNLINK:
		msg := eventmux.GetEventData[models.N2SmInfoDownlinkTransport](e)
		ueCtx.handleN2SmInfoDownlink(msg)

	case UPDATE_AMF_INFO:
		msg := eventmux.GetEventData[models.AmfUeContextInfo](e)
		ueCtx.updateAmfUeId(msg)

	case NAS_DOWNLINK:
		msg := eventmux.GetEventData[models.NasDownlinkTransport](e)
		ueCtx.handleNasDownlink(msg)

	case UECTX_SETUP:
		task := eventmux.GetEventData[UeCtxSetupContext](e)
		ueCtx.handleSetupContext(task)

	case UECTX_MODIFY:
		task := eventmux.GetEventData[UeCtxModifyContext](e)
		ueCtx.handleModifyContext(task)

	case UECTX_RELEASE:
		task := eventmux.GetEventData[UeCtxReleaseContext](e)
		ueCtx.handleReleaseContext(ctx, task)

	case HANDOVER_REQUEST:
		task := eventmux.GetEventData[HandoverRequestContext](e)
		ueCtx.handleHandoverRequest(task)
	}
}

func (ueCtx *UeContext) updateAmfUeId(msg *models.AmfUeContextInfo) {
	//update UeContext
	if err := ueCtx.updateAmfClient(msg.AmfSet, uint8(msg.AmfPointer), msg.PlmnId, msg.AmfUeId); err != nil {
		ueCtx.Errorf("Fail to update AMF info: %+v", err)
	} else {
		ueCtx.Infof("Update AMF information: amfId=%s:%d, amfUeId=%d", msg.AmfSet, msg.AmfPointer, msg.AmfUeId)
	}
}

func (ueCtx *UeContext) handleNasDownlink(msg *models.NasDownlinkTransport) {
	if msg.AmfUeInfo != nil {
		ueCtx.updateAmfUeId(msg.AmfUeInfo)
	}

	if err := ueCtx.sendDownlinkNasTransport(msg.NasPdu); err != nil {
		ueCtx.Errorf("Fail to forward NasDL from AMF  to gnB: %+v", err)
		//TODO: send NasErrMsg toward AMF
	} else {
		ueCtx.Infof("NasDL message from AMF is forwarded to gnB")
	}
}

func (ueCtx *UeContext) handleN2SmInfoDownlink(msg *models.N2SmInfoDownlinkTransport) {
	//TODO: send in go routines
	for _, transfer := range msg.Transfers {
		switch transfer.N2SmInfoType {
		case models.N2SMINFOTYPE_PDU_RES_SETUP_REQ:
			if err := ueCtx.sendPduSessionResourceSetupRequest(uint8(transfer.SessionId), transfer.N2SmInfo, transfer.NasPdu, transfer.Snssai); err != nil {
				ueCtx.Errorf("Fail to forward PduSessionResourceSetup Request to gNB: %+v", err)
			} else {
				ueCtx.Infof("PduSessionResourceSetupRequest from core is forwarded to gnB")
			}
		case models.N2SMINFOTYPE_PDU_RES_REL_CMD:
			if err := ueCtx.sendPduSessionResourceReleaseCommand(uint8(transfer.SessionId), transfer.N2SmInfo, transfer.NasPdu); err != nil {
				ueCtx.Errorf("Fail to forward PduSessionResourceSetup Request to gNB: %+v", err)
			} else {
				ueCtx.Infof("PduSessionResourceReleaseCommand from core is forwarded to gnB")
			}
		case models.N2SMINFOTYPE_PDU_RES_MOD_REQ:
			ueCtx.Warnf("Forwarding PduSessionResourceModificationRequest is not implemented")
			//case N2SMINFOTYPE_PDU_RES_NTY_REL:
		case models.N2SMINFOTYPE_PDU_RES_MOD_CFM:
			ueCtx.Warnf("Forwarding PduSessionResourceModifyConfirmation is not implemented")
		case models.N2SMINFOTYPE_PDU_RES_MOD_IND_FAIL:
			ueCtx.Warnf("Forwarding PduSessionResourceModificationIndication is not implemented")
		default:
			ueCtx.Warnf("N2SmInfoDownlink from core is not forwarded to gnB because of unknown N2SmInfoType[%s]", transfer.N2SmInfoType)
		}
	}

}

func (ueCtx *UeContext) handleSetupContext(info *UeCtxSetupContext) {
	if info.Request.AmfUeInfo != nil {
		ueCtx.updateAmfUeId(info.Request.AmfUeInfo)
	}

	if ueCtx.setupJob != nil {
		info.Finalize(fmt.Errorf("UeContextSetup request is not handled becuase there is a UeContext setup pending"))
		return
	}

	if err := ueCtx.sendInitialContextSetupRequest(info.Request); err == nil {
		ueCtx.Info("InitialUeContextSetupRequest from core is forwarded to gNB")
		ueCtx.setupJob = info
		info.SetFinalizer(func(old func(error)) func(error) {
			return func(err error) {
				ueCtx.setupJob = nil
				old(err)
			}
		})
	} else {
		err = utils.WrapError("Forward InitialUeContextSetupRequest to gNB", err)
		info.Finalize(err)
	}
}

func (ueCtx *UeContext) handleModifyContext(info *UeCtxModifyContext) {
	if err := ueCtx.sendUeContextModificationRequest(info.Request); err == nil {
		ueCtx.Infof("UeContextModificationRequest from core is forwarded to gNB")
		ueCtx.modifyJob = info
		info.SetFinalizer(func(old func(error)) func(error) {
			return func(err error) {
				ueCtx.modifyJob = nil
				old(err)
			}
		})
	} else {
		err = utils.WrapError("Forward UeContextModificationRequest to gnB", err)
		info.Finalize(err)
	}
}

func (ueCtx *UeContext) handleReleaseContext(ctx context.Context, info *UeCtxReleaseContext) {
	info.SetFinalizer(func(old func(error)) func(error) {
		return func(err error) {
			ueCtx.clean()
			old(err)
		}
	})
	if ueCtx.releaseCtx == nil {
		ueCtx.releaseCtx = &ReleaseContext{
			job: info,
		}
	} else {
		if ueCtx.releaseCtx.amfTimer != nil {
			ueCtx.releaseCtx.amfTimer.Stop() //stop AMF waiting timer
			ueCtx.releaseCtx.job = info      //attach pending sbi release command
		}
	}
	//command gnB to release UeContext
	ueCtx.releaseUeContext()
}
