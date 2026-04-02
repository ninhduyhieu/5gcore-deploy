package producer

import (
	"context"
	"etrib5gc/nfs/udm/ue"
	"github.com/reogac/sbi/apis/udm/ueauth"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *Producer) HandleConfirmAuth(supi string, body *models.AuthEvent) (headers map[string]string, rsp *models.AuthEvent, prob *models.ProblemDetails) {
	p.Debugf("Receive a ConfirmAuth request")
	var err error
	if rsp, err = ue.HandleConfirmAuth(supi, body); err != nil {
		p.Errorf("Fail to confirm authentication for %s: %+v", supi, err)
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		}
	}

	return
}

func (p *Producer) HandleGenerateAuthData(supiOrSuci string, body *models.AuthenticationInfoRequest) (rsp *models.AuthenticationInfoResult, prob *models.ProblemDetails) {
	p.Debugf("Receive a GenerateAuthData request [SuciOrSupi=%s", supiOrSuci)
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if rsp, err = ue.GenerateAuthData(ctx, supiOrSuci, body); err != nil {
		p.Errorf("GenerateAuthData for %s failed: %+v", supiOrSuci, err)
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		}
	}

	return
}

func (p *Producer) HandleGenerateAv(params *ueauth.GenerateAvParams, body *models.HssAuthenticationInfoRequest) (rsp *models.HssAuthenticationInfoResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleDeleteAuth(params *ueauth.DeleteAuthParams, body *models.AuthEvent) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGenerateGbaAv(supi string, body *models.GbaAuthenticationInfoRequest) (rsp *models.GbaAuthenticationInfoResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGenerateProseAV(supiOrSuci string, body *models.ProSeAuthenticationInfoRequest) (rsp *models.ProSeAuthenticationInfoResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetRgAuthData(params *ueauth.GetRgAuthDataParams) (rsp *models.RgAuthCtx, prob *models.ProblemDetails) {
	return
}
