package sm

import (
	"context"
	"etrib5gc/internal/eventmux"
	"etrib5gc/internal/fsm"
	"time"

	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

const (
	SBI_RELEASE_SMCONTEXT uint8 = iota
	SBI_UPDATE_SMCONTEXT
)

const (
	SBI_TIMEOUT time.Duration = 5 * time.Second
)

type ReleaseSmContextContext struct {
	Request  *models.ReleaseSmContextRequest
	Response *models.SmContextReleasedData
	*eventmux.AsyncTask
}

func NewReleaseSmContextContext(req *models.ReleaseSmContextRequest) *ReleaseSmContextContext {
	return &ReleaseSmContextContext{
		Request:   req,
		AsyncTask: eventmux.NewAsyncTask(),
	}
}

type AsyncTask interface {
	Wait() chan error
	Finalize(error)
}

type PtrTo[T any] interface{ ~*T }

type EventData struct {
	evType uint8
	dat    any
}

func ReceiveSbiEvent[T any, PT interface {
	PtrTo[T]
	AsyncTask
}](smCtx *SmContext, evType uint8, dat PT) error {
	ctx, cancel := context.WithTimeout(context.Background(), SBI_TIMEOUT)
	defer cancel()
	if err := <-smCtx.sendEvent(ctx, fsm.NewEventData(SbiEvent, &EventData{
		evType: evType,
		dat:    dat,
	})); err == nil {
		smCtx.Infof("Receive SBI EVENT")
		select {
		case <-ctx.Done():
			dat.Finalize(nil)
			return ctx.Err()
		case err = <-dat.Wait():
			if err != nil {
				return utils.WrapError("Handle SbiEvent", err)
			}
		}
	} else {
		return utils.WrapError("Send SbiEvent to state machine", err)
	}

	return nil
}

func (smCtx *SmContext) handleSbiEvent(ctx context.Context, e *EventData) {
	switch e.evType {
	case SBI_RELEASE_SMCONTEXT:
		task, _ := e.dat.(*ReleaseSmContextContext)
		smCtx.handleReleaseSmContextRequest(task)

	case SBI_UPDATE_SMCONTEXT:
		smCtx.Infof("Start handling UpdateSmContextRequest from AMF")
		task, _ := e.dat.(*UpdateSmContextContext)
		smCtx.handleUpdateSmContextRequest(task)
		smCtx.Infof("UpdateSmContextRequest is handled")
	default:
	}
}
