package uecontext

import (
	"context"
	amfctx "etrib5gc/nfs/amf/context"
	"fmt"
	"net/http"

	"github.com/reogac/sbi/models"
)

func (ueCtx *UeContext) doPaging(ctx context.Context, n1n2 *N1N2Context) {
	if err := amfctx.SendPaging(ueCtx.tmsi, ueCtx.taList); err != nil {
		n1n2.ErrResponse = &models.N1N2MessageTransferError{
			Error: models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: fmt.Sprintf("Fail to send paging: %+v", err),
			},
		}
	} else {
		n1n2.Response = &models.N1N2MessageTransferRspData{
			Cause: models.N1N2MESSAGETRANSFERCAUSE_ATTEMPTING_TO_REACH_UE,
		}
		//queue n1n2 transfer request
		n1n2.pending.isPaging = true
		ueCtx.addPendingN1N2(n1n2.pending)
	}
}
