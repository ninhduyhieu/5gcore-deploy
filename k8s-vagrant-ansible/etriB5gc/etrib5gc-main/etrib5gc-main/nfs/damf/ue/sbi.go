package ue

import (
	"context"
	"etrib5gc/internal/fsm"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

// Handle IntialUeMessage from PRAN, create and add UeContext to the pool
func CreateUeContext(ctx context.Context, ranInfo models.EndpointInfo, msg *models.InitialUeMessage) (int64, error) {
	// check if DAMF is terminated
	if _pool.isClosed() {
		return 0, fmt.Errorf("UePool already closed")
	}

	// creat new UeContext
	if ueCtx, err := newUeContext(ranInfo, msg); err == nil {
		_pool.add(ueCtx)
		ueCtx.Infof("Ue context is created")
		ueCtx.sendEvent(context.Background(), fsm.NewEmptyEventData(AuthEvent))
		return ueCtx.amfUeId, nil
	} else {
		return 0, utils.WrapError("Create UeContext", err)
	}
}

// Handle NasUl from PRAN
func (ueCtx *UeContext) HandleNasUl(ctx context.Context, msg *models.NasUplinkTransport) (prob *models.ProblemDetails) {
	return ueCtx.sendEventEx(ctx, fsm.NewEventData(NasUlEvent, msg))
}

// Handle NasErr indication from PRAN
func (ueCtx *UeContext) HandleNasErr(ctx context.Context, msg *models.UplinkNasError) (prob *models.ProblemDetails) {
	return ueCtx.sendEventEx(ctx, fsm.NewEventData(NasErrEvent, msg))
}
