package producer

import (
	"context"
	"etrib5gc/nfs/ausf/auth"
	ausfctx "etrib5gc/nfs/ausf/context"
	"github.com/reogac/sbi/models"
	"net/http"
	"time"
)

const (
	SBI_TIMEOUT time.Duration = 5 * time.Second
)

func (p *Producer) HandleEapAuthMethod(authCtxId string, body *models.EapSession) (*models.EapSession, *models.ProblemDetails) {
	p.Debugf("Receive UeAuthenticationsAuthCtxIdEAPConfirmation for authCtxId=%s", authCtxId)
	ctx, cancel := context.WithTimeout(context.Background(), SBI_TIMEOUT)
	defer cancel()
	if rsp, err := auth.HandleEapSession(ctx, authCtxId, body); err != nil {
		p.Errorf("Fail to handle eap session: %+v", err)
		return nil, &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Fail  to handle eap session",
		}

	} else {
		return rsp, nil
	}
}

func (p *Producer) HandleUeAuthentications5gAkaConfirmationPut(authCtxId string, body *models.ConfirmationData) (*models.ConfirmationDataResponse, *models.ProblemDetails) {
	p.Debugf("Receive UeAuthentications5gAkaConfirmationPut for authCtxId=%s", authCtxId)
	ctx, cancel := context.WithTimeout(context.Background(), SBI_TIMEOUT)
	defer cancel()

	if rsp, err := auth.Handle5gAkaConfirmation(ctx, authCtxId, body); err != nil {
		p.Errorf("Fail to handle 5gAkaConfirmation:%+v", err)
		return nil, &models.ProblemDetails{
			Detail: "Fail to handle 5GAkaConfirmation",
			Status: http.StatusInternalServerError,
		}
	} else {
		return rsp, nil
	}
}

func (p *Producer) HandleUeAuthenticationsPost(body *models.AuthenticationInfo) (map[string]string, *models.UEAuthenticationCtx, *models.ProblemDetails) {
	p.Debugf("Receive an UeAuthenticationsPost [supiOrSuci=%s]", body.SupiOrSuci)

	snName := body.ServingNetworkName
	if !ausfctx.IsNetworkAuthorized(snName) {
		p.Errorf("Network %s not authorized", snName)
		return nil, nil, &models.ProblemDetails{
			Detail: "Serving network is not authorized",
			Status: http.StatusForbidden,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), SBI_TIMEOUT)
	defer cancel()

	if authId, rsp, err := auth.HandleAuthenticationPost(ctx, body); err != nil {
		p.Errorf("Fail to handle AuthenticationPost:%+v", err)
		return nil, nil, &models.ProblemDetails{
			Detail: "Fail to handle AuthenticationPost",
			Status: http.StatusInternalServerError,
		}
	} else {
		headers := make(map[string]string)
		headers["Location"] = authId
		return headers, rsp, nil
	}
}

func (p *Producer) HandleDeleteEapAuthenticationResult(authCtxId string) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleRgAuthenticationsPost(body *models.RgAuthenticationInfo) (headers map[string]string, rsp *models.RgAuthCtx, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleProseAuthenticationsPost(body *models.ProSeAuthenticationInfo) (headers map[string]string, rsp *models.ProSeAuthenticationCtx, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleProseAuth(authCtxId string, body *models.ProSeEapSession) (rsp *models.ProseAuthResponse, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUeAuthenticationsDeregisterPost(body *models.DeregistrationInfo) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleDelete5gAkaAuthenticationResult(authCtxId string) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleDeleteProSeAuthenticationResult(authCtxId string) (prob *models.ProblemDetails) {
	return
}
