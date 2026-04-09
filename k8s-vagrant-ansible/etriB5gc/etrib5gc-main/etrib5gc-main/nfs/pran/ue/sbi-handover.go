package ue

import (
	"etrib5gc/internal/eventmux"
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

type HandoverRequestContext struct {
	Request     *models.HandoverRequest
	Response    *models.HandoverRequestAcknowledge
	ErrResponse *models.HandoverRequestFailure
	*eventmux.AsyncTask
}

func HandleHandoverRequest(callback *models.EndpointInfo, gnb Ran, hoCtx *HandoverRequestContext) (err error) {
	//1. create UeContext
	if isUePoolClosed() {
		//Context has been closed, don't create any new UeContext
		return fmt.Errorf("Application terminating")
	}
	//find the designate AMF; if not found locate a default one
	var amfCli sbi.ConsumerClient
	if amfCli, err = mesh.ConsumerFromEndpoint(callback); err != nil {
		return utils.WrapError("Create a AMF/DAMF client", err)
	}

	//now create a new UeContext
	//generate identity at CU for the UE
	localId := models.RanUeId{
		Ran: _pool.ranId,
		Id:  _pool.ngapIdGen.Allocate(),
	}

	ueCtx := &UeContext{
		ran:     gnb,
		amfCli:  amfCli,
		amfUeId: hoCtx.Request.AmfUeId,
		localId: localId,
	}
	ueCtx.Entry = log.WithFields(logrus.Fields{
		"ueId": ueCtx.localId.Id,
	})
	log.Infof("Create a new Ue context for handover [ueId = %d]", localId.Id)
	//add UeContext to pool
	_pool.add(ueCtx)

	//2. handle Handover Request now
	ReceiveSbiEvent[HandoverRequestContext](ueCtx, HANDOVER_REQUEST, hoCtx)
	return nil
}

func (ueCtx *UeContext) handleHandoverRequest(info *HandoverRequestContext) {
	ueCtx.Infof("Handle HandoverRequest from AMF")
	if err := ueCtx.sendHandoverRequest(info.Request); err == nil {
		ueCtx.handoverJob = info
		info.SetFinalizer(func(old func(error)) func(error) {
			return func(err error) {
				ueCtx.handoverJob = nil
				old(err)
			}
		})
	} else {
		err = utils.WrapError("Send HandoverRequest to gnB", err)
		info.Finalize(err)
	}
}
