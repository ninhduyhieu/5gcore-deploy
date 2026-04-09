package producer

import (
	"context"
	"etrib5gc/nfs/amf/ranue"
	"etrib5gc/nfs/amf/uecontext"
	"github.com/reogac/sbi/models"
	"net/http"

	"github.com/reogac/nas"
)

func (p *producerImpl) HandleNasUl(ueId int64, msg *models.NasUplinkTransport) *models.ProblemDetails {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle NasUl", ueId)
		return notFoundProblem
	} else {
		p.Tracef("Receive a NasUl message for ueId=%d", ueId)

		if err := ranUe.ReceiveNasUplink(context.TODO(), msg.NasPdu); err != nil {
			ranUe.Errorf("Fail to handle NasUl: %+v", err)
			return internalProblem
		}
	}

	return nil
}

func (p *producerImpl) HandleNasErr(ueId int64, msg *models.UplinkNasError) *models.ProblemDetails {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle NasErr", ueId)
		return notFoundProblem
	} else {
		p.Trace("Receive Nas Non Delivery Indication for ueId=%d", ueId)
		ranUe.ReceiveNasErr(context.TODO(), msg)
	}

	return nil
}

func (p *producerImpl) HandleInitialUeMessage(callback *models.EndpointInfo, msg *models.InitialUeMessage) (*models.InitialUeMessageResponse, *models.ProblemDetails) {
	var nasMsg nas.NasMessage
	var err error
	var ranUe *ranue.RanUe
	var ueCtx *uecontext.UeContext

	p.Infof("Receive InitialUeMessage [RanUeId=%s]", msg.RanUeId.String())

	//1. decode plain text content
	if nasMsg, err = nas.Decode(nil, msg.NasPdu, true); err != nil {
		p.Errorf("Decode Nas failed: %+v", err)
		return nil, &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Detail: "Fail to decode NAS pdu",
		}
	}
	if nasMsg.Gmm == nil {
		p.Errorf("Decoded Nas has no N1Mm") //malicious RAN?
		return nil, &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Detail: "Empty N1Mm",
		}
	}

	//2. Find/Create UeContext
	if msg.AuthCtx != nil { //already authenticated at DAMF
		//2.a find with SUPI
		if ueCtx = uecontext.FindUeBySupi(msg.AuthCtx.Supi); ueCtx != nil {
			p.Infof("Found UeContext with supi[%s] received from default AMF", msg.AuthCtx.Supi)
		}
	}
	// 2.b find/create with Mobile Identity
	if ueCtx == nil {
		if ueCtx, err = uecontext.GetUeContext(nasMsg.Gmm); err != nil {
			p.Errorf("Fail to get UeContext: %+v", err)
			return nil, &models.ProblemDetails{
				Status: http.StatusNotFound,
				Detail: "Fail to get UeContext",
			}
		}

	}

	//3. create a new RanUe, it refers to the UeContext. However, the
	//UeContext has not attached the RanUe yet (util it grants the RanUe a
	//permission to procceed a registration/deregistration/service request procedure)

	if ranUe, err = ranue.CreateRanUe(ueCtx, msg, callback, nasMsg.Gmm); err != nil {
		p.Errorf("Create RanUe Context failed: %+v", err)
		return nil, &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Failed to create RanUe",
		}
	} else if err = ueCtx.BindRanUe(ranUe, msg, nasMsg.Gmm); err != nil {
		p.Errorf("Fail to attach RanUe: %+v", err)
		return nil, &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Failed to bind RanUe",
		}
	} else {

		//now that RanUE context has been initialized, we should create a response
		//regardless of how the inner NAS message is handled
		return &models.InitialUeMessageResponse{
			AmfUeId: ranUe.AmfUeId(),
		}, nil
	}

}
