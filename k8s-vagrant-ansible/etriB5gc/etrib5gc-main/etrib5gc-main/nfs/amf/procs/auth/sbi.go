package auth

import (
	"encoding/hex"
	"etrib5gc/common"
	"fmt"
	"github.com/reogac/sbi/apis/ausf/ueauth"
	"github.com/reogac/sbi/models"
)

func (proc *AuthProc) put5gAkaConfirm(resstar []byte) (rsp *models.ConfirmationDataResponse, err error) {
	proc.Debugf("Send 5GAKA confirmation to AUSF")
	body := &models.ConfirmationData{
		ResStar: hex.EncodeToString(resstar[:]),
	}

	rsp, err = ueauth.UeAuthentications5gAkaConfirmationPut(proc.ausfCli, proc.authId, body)
	return
}

func (proc *AuthProc) sendEapResponse(eapmsg string) (rsp *models.EapSession, err error) {
	proc.Debugf("Send Eap response to AUSF")
	body := &models.EapSession{
		EapPayload: eapmsg,
	}
	rsp, err = ueauth.EapAuthMethod(proc.ausfCli, proc.authId, body)
	return
}

func (proc *AuthProc) getAuthenticationContext(reSync *models.ResynchronizationInfo) (authId string, rsp *models.UEAuthenticationCtx, err error) {
	proc.Debugf("Request Authentication Context from AUSF")
	info := &models.AuthenticationInfo{
		ServingNetworkName:    common.ServingNetworkName(&proc.plmnId),
		ResynchronizationInfo: reSync,
	}
	//just use SUCI (as supi in UDR is still in wrong format (IMSI-PLMNMSIN)
	if len(proc.supi) > 0 {
		proc.Tracef("Using supi for authentication")
		info.SupiOrSuci = proc.supi
	} else {
		info.SupiOrSuci = proc.suci
	}

	var headers map[string]string
	headers, rsp, err = ueauth.UeAuthenticationsPost(proc.ausfCli, info)
	if reSync == nil {
		if loc, ok := headers["Location"]; ok {
			authId = loc
		} else {
			err = fmt.Errorf("AUSF did not send authentication id")
		}
	}
	return
}
