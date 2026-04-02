package producer

import (
	"etrib5gc/nfs/nsm/context"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *Producer) HandleAmfRegister(req *models.AmfRegistrationRequest) (rsp *models.AmfRegistrationResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive a AMF registration request")
	var err error
	if rsp, err = context.AmfRegister(req); err != nil {
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		}
	}
	return
}

func (p *Producer) HandleGetSupportedSlices(body *models.GetSupportedSlicesRequest) (rsp *models.GetSupportedSlicesResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive a Supported Plmn List update")
	rsp = &models.GetSupportedSlicesResponse{
		Slices: context.GetSupportedSlices(body.AmfRegion),
	}
	return
}
