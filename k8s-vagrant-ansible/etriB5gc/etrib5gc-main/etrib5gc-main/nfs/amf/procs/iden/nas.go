package iden

import (
	"github.com/reogac/nas"
	"github.com/reogac/utils"
)

func (proc *IdentificationProcedure) makeSendIdRequestFunction(nasCtx *nas.NasContext, isGpp bool, sendNasDl func([]byte) error) {
	proc.sendIdRequest = func() error {
		proc.Infof("Request Ue Id of type %d", proc.idType)
		msg := &nas.IdentityRequest{
			IdentityType: proc.idType,
		}
		if nasCtx != nil {
			msg.SetSecurityHeader(nas.NasSecBoth)
		} else {
			msg.SetSecurityHeader(nas.NasSecNone)
		}

		if pdu, err := nas.EncodeMm(nasCtx, msg, isGpp); err != nil {
			return utils.WrapError("Encode IdentityRequest", err)
		} else {
			if err = sendNasDl(pdu); err != nil {
				return utils.WrapError("Send IdentityRequest to UE", err)
			}
		}
		return nil
	}
}
