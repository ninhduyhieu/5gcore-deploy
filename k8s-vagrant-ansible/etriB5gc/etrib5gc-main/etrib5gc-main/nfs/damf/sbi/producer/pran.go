package producer

import (
	"context"
	"etrib5gc/nfs/damf/ue"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (prod *producerImpl) HandleInitialUeMessage(callback *models.EndpointInfo, body *models.InitialUeMessage) (*models.InitialUeMessageResponse, *models.ProblemDetails) {
	prod.Debugf("Receive an InitialUeMessage request")
	prod.Tracef("Callback to proxy RAN: %s", models.EndpointInfoToString(*callback))
	if !ue.HasUe(body.RanUeId) {
		if amfUeId, err := ue.CreateUeContext(context.TODO(), *callback, body); err != nil {
			prod.Errorf("Create UeContext [ranUeId=%s] failed: %s", body.RanUeId.String(), err.Error())
			return nil, &models.ProblemDetails{
				Detail: "Can not create UeContext",
				Status: http.StatusInternalServerError,
			}
		} else {
			//set AmfUeId in the response
			return &models.InitialUeMessageResponse{
				AmfUeId: amfUeId,
			}, nil
		}
	} else {
		prod.Errorf("UeContext[ranUeId=%s] existed", body.RanUeId.String())
		return nil, &models.ProblemDetails{
			Detail: "UeContext existed",
			Status: http.StatusConflict,
		}
	}
}

func (prod *producerImpl) HandleNasUl(ueId int64, body *models.NasUplinkTransport) *models.ProblemDetails {
	prod.Debugf("Receive a NasUL request")
	if ueCtx := ue.FindUe(ueId); ueCtx == nil {
		prod.Errorf("UeContext not found [ueId=%d]", ueId)
		return notFoundProblem
	} else {
		return ueCtx.HandleNasUl(context.TODO(), body)
	}
}

func (prod *producerImpl) HandleNasErr(ueId int64, body *models.UplinkNasError) *models.ProblemDetails {
	prod.Debugf("Receive a NasNonDelivery request")
	if ueCtx := ue.FindUe(ueId); ueCtx == nil {
		prod.Errorf("UeContext not found [ueId=%d]", ueId)
		return notFoundProblem
	} else {
		return ueCtx.HandleNasErr(context.TODO(), body)
	}
}
