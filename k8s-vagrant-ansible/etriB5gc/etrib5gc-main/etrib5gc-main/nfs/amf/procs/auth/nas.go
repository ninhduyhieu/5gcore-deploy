package auth

import (
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func (proc *AuthProc) sendAuthenticationRequest() error {
	proc.Debugf("Send AuthenticationRequest to UE")
	msg := &nas.AuthenticationRequest{
		Ngksi: *proc.ngKsi.NasType(),
		Abba:  proc.abba,
	}
	msg.SetSecurityHeader(nas.NasSecNone)

	switch proc.authType {
	case models.AUTHTYPE_5G_AKA:
		msg.AuthenticationParameterRand = proc.rand
		msg.AuthenticationParameterAutn = proc.autn

	case models.AUTHTYPE_EAP_AKA_PRIME:
		msg.EapMessage = proc.eap
	}

	if pdu, err := nas.EncodeMm(nil, msg, proc.isGpp); err == nil {
		if err = proc.sendNasDl(pdu); err != nil {
			return utils.WrapError("Send N1MM downlink", err)
		}
	} else {
		return utils.WrapError("Encode N1MM message", err)
	}
	return nil
}

// Eap success message is sent through AuthenticationResult if Authentication
// procedure is not followed by a SecurityModeCommand or a
// RegistrationAccept/ServiceAccept
func (proc *AuthProc) sendAuthenticationResult() error {
	proc.Debugf("Send NAS AuthenticationResult to UE")
	msg := &nas.AuthenticationResult{
		Ngksi:      *proc.ngKsi.NasType(),
		EapMessage: proc.eap,
		Abba:       proc.abba,
	}
	msg.SetSecurityHeader(nas.NasSecNone)

	if pdu, err := nas.EncodeMm(nil, msg, proc.isGpp); err == nil {
		if err = proc.sendNasDl(pdu); err != nil {
			return utils.WrapError("Send N1MM downlink", err)
		}
	} else {
		return utils.WrapError("Encode N1MM mesage", err)
	}

	return nil
}

// Eap failure message is sent through an AuthenticationReject if it is not sent
// by a RegistrationReject or ServiceReject
func (proc *AuthProc) sendAuthenticationReject() (err error) {
	proc.Debugf("Send NASAuthenticationReject to UE")
	msg := &nas.AuthenticationReject{}
	msg.SetSecurityHeader(nas.NasSecNone)

	if len(proc.eap) > 0 {
		msg.EapMessage = proc.eap
	}

	var pdu []byte
	if pdu, err = nas.EncodeMm(nil, msg, proc.isGpp); err == nil {
		err = utils.WrapError("Send N1MM message", proc.sendNasDl(pdu))
	} else {
		err = utils.WrapError("Encode N1MM message", err)
	}

	return
}
