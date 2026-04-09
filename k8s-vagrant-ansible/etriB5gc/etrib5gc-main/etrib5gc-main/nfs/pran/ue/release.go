package ue

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/eventmux"
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/apis/amf/uectx"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"time"
)

const (
	RELEASE_TIMEOUT = 5 * time.Second //time for release procedure to complete (miliseconds)
)

type ReleaseContext struct {
	doneCh   chan struct{}
	amfTimer common.Timer //wait for AMF
	gnbTimer common.Timer //wait for gnB

	job *UeCtxReleaseContext //pending ue context release request
}

// force to release Ue. Call once, no more
func (ueCtx *UeContext) Kill() {
	ueCtx.Tracef("Kill UeContext")
	doneCh := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ev := eventmux.NewEventData[chan struct{}](ForceCloseEvent, &doneCh)
	if err := _exec.Send(ctx, ueCtx.execSlot, ev); err != nil {
		ueCtx.Errorf("Fail to send event: %+v", err)
	}
	select {
	case <-ctx.Done():
	case <-doneCh:
	}
}
func (ueCtx *UeContext) forceClose(ctx context.Context, doneCh chan struct{}) {
	if ueCtx.releaseCtx != nil {
		if ueCtx.releaseCtx.doneCh == nil {
			ueCtx.releaseCtx.doneCh = doneCh
		} else { //should never happen
			doneCh <- struct{}{}
		}
	} else {
		req := &models.UeContextReleaseRequest{
			Cause: models.N2Cause{
				CausePresent: int16(ies.CausePresentMisc),
				CauseValue:   int16(ies.CauseMiscUnspecified),
			},
		}
		ueCtx.requestUeContextRelease(ctx, req, doneCh)
	}
}

//notify AMF to trigger UeContext Release; create release context
func (ueCtx *UeContext) requestUeContextRelease(ctx context.Context, req *models.UeContextReleaseRequest, doneCh chan struct{}) {
	if err := uectx.UeContextRelease(ueCtx.amfCli, ueCtx.amfUeId, req); err != nil {
		ueCtx.Errorf("Fail to notify AMF to release UeContext: %+v", err)
		ueCtx.releaseCtx = &ReleaseContext{
			doneCh: doneCh,
		}
		ueCtx.releaseUeContext() //command gnB to release UeContext
	} else {
		ueCtx.Tracef("AMF is notified to release UeContext request")
		//create a timer to wait for AMF to send release command
		ueCtx.releaseCtx = &ReleaseContext{
			doneCh: doneCh,
			amfTimer: common.NewTimer(RELEASE_TIMEOUT, func() {
				_exec.Send(context.Background(), ueCtx.execSlot, eventmux.NewEmptyEventData(AmfUeCtxReleaseTimeoutEvent))
			}, nil),
		}
		ueCtx.releaseCtx.amfTimer.Start()
	}
}

//command gnB to release UeContext
func (ueCtx *UeContext) releaseUeContext() {
	//incase time out and release has been gone
	if ueCtx.releaseCtx == nil {
		//nothing to do
		return
	}

	var cmd *models.UeContextReleaseCommand
	info := ueCtx.releaseCtx.job
	if info != nil {
		cmd = info.Command
	} else {
		cmd = &models.UeContextReleaseCommand{
			//TODO: add cause
		}
	}
	//send UeContext release command to gnB
	if err := ueCtx.sendUeContextReleaseCommand(cmd); err == nil {
		ueCtx.Infof("UeContextReleaseCommand from core is forwarded to gnB")
		ueCtx.releaseCtx.gnbTimer = common.NewTimer(RELEASE_TIMEOUT, func() {
			ueCtx.Warnf("GnB did not send release complete, timeouted")
			_exec.Send(context.Background(), ueCtx.execSlot, eventmux.NewEmptyEventData(GnbUeCtxReleaseTimeoutEvent))
		}, nil)
		ueCtx.releaseCtx.gnbTimer.Start()
	} else { //fail to command gnB, release local UeContext
		if info != nil {
			info.Finalize(utils.WrapError("Forward UeContextReleaseCommand to gnB", err))
		} else {
			ueCtx.clean()
		}
	}
}

func (ueCtx *UeContext) clean() {
	ueCtx.Warnf("Clean up UeContext")
	//in case time out and release has been gone
	if ueCtx.releaseCtx == nil {
		//nothing to do
		return
	}

	//clean up sbi pending tasks
	if ueCtx.modifyJob != nil {
		ueCtx.modifyJob.Finalize(fmt.Errorf("Ue is released"))
	}

	for _, smCtx := range ueCtx.sessions {
		if smCtx != nil {
			smCtx.clean()
		}
	}

	//remove UeContext
	_pool.remove(ueCtx)

	if ueCtx.releaseCtx.doneCh != nil {
		ueCtx.releaseCtx.doneCh <- struct{}{}
	}

	ueCtx.releaseCtx = nil
}
