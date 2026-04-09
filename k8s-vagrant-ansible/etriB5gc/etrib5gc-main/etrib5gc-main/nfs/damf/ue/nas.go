package ue

import (
	"etrib5gc/internal/nasbuilder"
	"etrib5gc/nfs/damf/context"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/apis/pran/nasdl"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func (ueCtx *UeContext) sendNasDl(pdu []byte) error {
	return nasdl.NasDl(ueCtx.ranCli, ueCtx.ranUeId.Id, &models.NasDownlinkTransport{
		NasPdu: pdu,
	})
}

func (ueCtx *UeContext) sendRegistrationReject(n1Cause uint8, eap []byte) error {
	ueCtx.Debugf("Send RegistrationReject")
	builder := func(msg *nas.RegistrationReject) error {
		msg.GmmCause = n1Cause
		msg.T3502Value = &nas.GprsTimer2{
			Value: context.GetT3502(),
		}
		msg.SetSecurityHeader(nas.NasSecNone)
		return nil
	}

	if pdu, err := nasbuilder.NewGmmComposer(nil, ueCtx.isGpp, new(nas.RegistrationReject)).Build(builder); err != nil {
		return utils.WrapError("Build RegistrationReject", err)
	} else if err = ueCtx.sendNasDl(pdu); err != nil {
		ueCtx.Errorf("Send RegistrationReject failed: %+v", err)
	}
	return nil
}

func (ueCtx *UeContext) sendServiceReject(cause uint8) error {
	ueCtx.Debugf("Send ServiceReject")
	msg := &nas.ServiceReject{
		GmmCause: cause,
	}

	msg.SetSecurityHeader(nas.NasSecNone)

	if pdu, err := nas.EncodeMm(nil, msg, ueCtx.isGpp); err == nil {
		if err = ueCtx.sendNasDl(pdu); err != nil {
			return utils.WrapError("Send NasDl", err)
		}
	} else {
		return utils.WrapError("Encode Service Request", err)
	}
	return nil
}
