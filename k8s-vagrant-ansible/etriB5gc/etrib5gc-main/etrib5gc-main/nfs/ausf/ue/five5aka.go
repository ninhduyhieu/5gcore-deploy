package ue

import (
	"fmt"
	"github.com/reogac/sbi/models"
)

func (ue *UeContext) Handle5gAkaConfirmation(body *models.ConfirmationData) (rsp *models.ConfirmationDataResponse, err error) {
	if ue.checkResStar(body.ResStar) {
		ue.Info("UE is authenticated")
		if err = ue.confirmAuth2Udm(true); err == nil {
			rsp = &models.ConfirmationDataResponse{
				AuthResult: models.AUTHRESULT_AUTHENTICATION_SUCCESS,
				Supi:       ue.Supi(),
				Kseaf:      ue.Kseaf(),
				AmData:     ue.AmData(),
			}
		}
	} else {
		err = fmt.Errorf("Mismatched resstar")
	}
	return
}
