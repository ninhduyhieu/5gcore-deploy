package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"etrib5gc/common"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/sec5g"
	"github.com/sirupsen/logrus"
	"strings"
	"time"

	"github.com/reogac/nas"
)

const (
	MAX_SYNC_FAILURE_CNT int           = 2
	T3560_DURATION       time.Duration = 5000 //miliseconds
	MAX_T3560_CNT        int           = 4
	MAX_SYNC_CNT         int           = 2
	MAX_NGKSI_CNT        int           = 2
)

type Context struct {
	Log            *logrus.Entry
	SendNasDl      func([]byte) error
	Callback       func(*Result, bool)
	Abba           []byte
	OnT3560        func(*AuthProc)
	PlmnId         *models.PlmnId //serving network
	Suci           string
	IsGpp          bool
	AusfCli        sbi.ConsumerClient
	SelectNewNgKsi func(*models.NgKsi) models.NgKsi //select new ngKsi
}

type AuthProc struct {
	*logrus.Entry
	suci           string
	ausfCli        sbi.ConsumerClient
	selectNewNgKsi func(*models.NgKsi) models.NgKsi
	abba           []byte
	sendNasDl      func([]byte) error
	callback       func(*Result, bool)
	plmnId         models.PlmnId //serving network
	isGpp          bool

	sendEap bool //should eap success be sent with AuthenticationResult?

	syncCnt  int //re-sync counter
	ngKsiCnt int //re-generate ngKsi counter
	t3560Cnt int //t3560 counter
	t3560    common.Timer

	ngKsi models.NgKsi

	authType models.AuthType
	authId   string //from AUSF

	supi      string
	rand      []byte
	autn      []byte
	kamf      []byte
	hxResstar []byte
	eap       []byte

	requested bool //has Ue been requested for authentication?
}

type Result struct {
	Eap      []byte
	Kamf     []byte
	Supi     string
	AuthType models.AuthType
	NgKsi    models.NgKsi
}

func Start(ctx *Context) *AuthProc {
	proc := &AuthProc{
		Entry: ctx.Log.WithFields(logrus.Fields{
			"mod": "auth",
		}),
		suci:           ctx.Suci,
		abba:           ctx.Abba,
		ausfCli:        ctx.AusfCli,
		selectNewNgKsi: ctx.SelectNewNgKsi,
		sendNasDl:      ctx.SendNasDl,
		callback:       ctx.Callback,
		plmnId:         *ctx.PlmnId,
		isGpp:          ctx.IsGpp,
		sendEap:        false,

		ngKsi: ctx.SelectNewNgKsi(nil),
	}
	proc.t3560 = common.NewTimer(T3560_DURATION*time.Millisecond, func() {
		ctx.OnT3560(proc)
	}, nil)

	proc.getUeAuthContext(nil)
	return proc
}

// close all timer
func (proc *AuthProc) Close() {
	proc.t3560.Stop()
}

func (proc *AuthProc) ReceiveN1Mm(ctx context.Context, gmm *nas.DecodedGmmMessage) {
	switch gmm.MsgType {
	case nas.AuthenticationResponseMsgType:
		proc.requested = true
		proc.handleAuthenticationResponse(ctx, gmm.AuthenticationResponse)
	case nas.AuthenticationFailureMsgType:
		proc.requested = true
		proc.handleAuthenticationFailure(ctx, gmm.AuthenticationFailure)
	case nas.GmmStatusMsgType:
		msg := gmm.GmmStatus
		cause := uint8(msg.GmmCause)
		proc.Warnf("Receive a 5GMMStatus message: cause= %s]", nas.Cause5GMMToString(cause))
	default:
		proc.Errorf("Receive Unexpected Nas Message during authentication")
	}
	return

}

// handle an authentication rsponse from the UE. The handling is dependent on
// the authentication method (5g-aka or eap-aka-prime).
func (proc *AuthProc) handleAuthenticationResponse(ctx context.Context, msg *nas.AuthenticationResponse) {
	proc.Debugf("Handle NAS AuthenticationResponse")
	//reset T3560
	proc.t3560.Stop()
	proc.t3560Cnt = 0
	//reset other counter
	proc.syncCnt = 0
	proc.ngKsiCnt = 0

	var err error
	switch proc.authType {
	case models.AUTHTYPE_5G_AKA:
		resstar := msg.AuthenticationResponseParameter
		if len(resstar) == 0 {
			proc.complete(fmt.Errorf("Empty ResStar in AuthenticationResponse"))
			return
		}

		hx := sha256.Sum256(append(proc.rand, resstar[:]...))
		hresstar := hx[16:]

		//check resstar
		if !bytes.Equal(hresstar, proc.hxResstar) {
			proc.complete(fmt.Errorf("Mismatched ResStar in AuthenticationResponse: %x vs %x [xresstar=%x]", hresstar, proc.hxResstar, resstar))
			return
		}
		proc.Infof("Receive valid AuthenticationResponse from UE")
		var rsp *models.ConfirmationDataResponse
		if rsp, err = proc.put5gAkaConfirm(resstar[:]); err != nil {
			err = utils.WrapError("Put5gAkaConfirmation to AUSF", err)
			proc.complete(err)
			return
		}
		switch rsp.AuthResult {
		case models.AUTHRESULT_AUTHENTICATION_SUCCESS:
			proc.Infof("Receive authentication success from AUSF [SUPI=%s]", rsp.Supi)
			if err = proc.createKamf(rsp.Supi, rsp.Kseaf); err != nil {
				err = utils.WrapError("Derive KAMF key", err)
				proc.complete(err)
				return
			}
			proc.complete(nil)
		case models.AUTHRESULT_AUTHENTICATION_FAILURE:
			proc.complete(fmt.Errorf("Authentication failure received from AUSF"))
		default:
			proc.complete(fmt.Errorf("Unknown authentication result received from AUSF"))
		}

	case models.AUTHTYPE_EAP_AKA_PRIME:
		if msg.EapMessage == nil {
			proc.complete(fmt.Errorf("Empty EAP message in AuthenticationResponse"))
			return
		}
		//send eap rsponse toward AUSF
		var rsp *models.EapSession
		eapMsg := base64.StdEncoding.EncodeToString(msg.EapMessage)
		if rsp, err = proc.sendEapResponse(eapMsg); err != nil {
			err = utils.WrapError("Send Eap message to AUSF", err)
			proc.complete(err)
			return
		}

		proc.Infof("Receive valid AuthenticationResponse from UE")
		//inteprete the rsponse
		switch rsp.AuthResult {
		case models.AUTHRESULT_AUTHENTICATION_SUCCESS:
			proc.Infof("Receive authentication success from AUSF [SUPI=%s]", rsp.Supi)
			if err = proc.createKamf(rsp.Supi, rsp.KSeaf); err == nil {
				if proc.eap, err = base64.StdEncoding.DecodeString(rsp.EapPayload); err == nil {
					proc.complete(nil)
					return
				} else {
					err = utils.WrapError("Decode base64 eap message", err)
				}
			} else {
				err = utils.WrapError("Derive KAMF", err)
			}
			proc.complete(err)

		case models.AUTHRESULT_AUTHENTICATION_FAILURE:
			proc.complete(fmt.Errorf("Eap failure received from AUSF"))

		case models.AUTHRESULT_AUTHENTICATION_ONGOING:
			proc.Infof("Receive eap message from AUSF [Authentication on-going]]")
			if proc.eap, err = base64.StdEncoding.DecodeString(rsp.EapPayload); err == nil {
				//receive a EAP request from AUSF, send it to UE
				proc.challenge()
			} else {
				err = utils.WrapError("Decode base64 eap message", err)
				proc.complete(err)
			}
		}
	}
}

func (proc *AuthProc) handleAuthenticationFailure(ctx context.Context, msg *nas.AuthenticationFailure) {
	proc.Debugf("Handle NAS AuthenticationFailure")
	//reset T3560
	proc.t3560.Stop()
	proc.t3560Cnt = 0

	cause := msg.GmmCause
	if proc.authType == models.AUTHTYPE_5G_AKA {
		switch cause {
		case nas.Cause5GMMMACFailure:
			proc.complete(fmt.Errorf("Mac Failure"))

		case nas.Cause5GMMNon5GAuthenticationUnacceptable:
			proc.complete(fmt.Errorf("Non-5G Authentication Unacceptable"))

		case nas.Cause5GMMngKSIAlreadyInUse:
			proc.Warnf("Authentication Failure Cause: NgKSI Already In Use")
			proc.ngKsiCnt++
			if proc.ngKsiCnt >= MAX_NGKSI_CNT {
				proc.complete(fmt.Errorf("Max NgKsi already-in-use failures"))
				return
			}
			// select new ngKsi
			proc.ngKsi = proc.selectNewNgKsi(&proc.ngKsi)
			//re-send authentication request
			proc.challenge()

		case nas.Cause5GMMSynchFailure: // TS 24.501 5.4.1.3.7 case f
			proc.Warnf("Authentication Failure 5GMM Cause: Synch Failure")
			proc.syncCnt++
			if proc.syncCnt >= MAX_SYNC_CNT {
				proc.complete(fmt.Errorf("Max sync failure"))
				return
			}
			if len(msg.AuthenticationFailureParameter) == 0 {
				proc.complete(fmt.Errorf("NAS AuthenticationFailure with empty parameter"))
				return
			}

			reSync := &models.ResynchronizationInfo{
				Auts: hex.EncodeToString(msg.AuthenticationFailureParameter),
				Rand: hex.EncodeToString(proc.rand),
			}
			//request authentication context with reSynchronized SQN
			proc.getUeAuthContext(reSync)
		default:
			proc.complete(fmt.Errorf("NAS AuthenticationFailure with unknown cause: %d", cause))
		}
	} else { //EAP-AKA-PRIME
		switch cause {
		case nas.Cause5GMMngKSIAlreadyInUse:
			proc.Warnf("Authentication Failure Cause: NgKSI Already In Use")
			proc.ngKsiCnt++
			if proc.ngKsiCnt >= MAX_NGKSI_CNT {
				proc.complete(fmt.Errorf("Max ngKsit already-in-use failures"))
				return
			}
			// select new ngKsi
			proc.ngKsi = proc.selectNewNgKsi(&proc.ngKsi)
			//re-send authentication request
			proc.challenge()

		default:
			proc.complete(fmt.Errorf("NAS AuthenticationFailure with unknown cause: %d", cause))
		}
	}
}

func (proc *AuthProc) complete(err error) {
	var rst *Result
	rejected := false
	if err != nil { //authentication fail
		proc.Errorf("Authentication fails with error: %+v", err)
		if proc.requested { //send an authentication reject if UE was requested
			if err = proc.sendAuthenticationReject(); err != nil {
				proc.Errorf("Send authentication reject fail: %+v", err)
			} else {
				rejected = true
			}
		}

	} else { //authentication success
		rst = &Result{
			Kamf:     proc.kamf,
			Supi:     proc.supi,
			AuthType: proc.authType,
			NgKsi:    proc.ngKsi,
		}

		if proc.authType == models.AUTHTYPE_EAP_AKA_PRIME {
			if proc.sendEap {
				//send eap success now
				if err = proc.sendAuthenticationResult(); err != nil {
					proc.Errorf("Send authentication result fail: %+v", err)
					rst.Eap = proc.eap //send EAP message later
				}
			} else { //otherwise, eapMsg should be sent with SecurityModeCommand or RegistrationAccept/ServiceAccept
				rst.Eap = proc.eap
			}
		}
	}
	//execute the callback on authentication completion
	proc.callback(rst, rejected)
}

// get authentication context from AUSF
func (proc *AuthProc) getUeAuthContext(reSync *models.ResynchronizationInfo) {
	if authId, authctx, err := proc.getAuthenticationContext(reSync); err != nil {
		err = utils.WrapError("Get AuthenticationContext from AUSF", err)
		proc.complete(err)
		return
	} else {
		proc.Infof("Receive authentication context from the AUSF: %s", authId)
		if reSync == nil {
			proc.authId = authId
		}
		if err = proc.loadAuthCtx(authctx); err != nil {
			err = utils.WrapError("Load received authentication context", err)
			proc.complete(err)
			return
		}
		//then send Downlink AuthenticationRequest to UE
		proc.challenge() //send challening
	}
	return
}

// send nas authentication request toward UE
func (proc *AuthProc) challenge() {
	if err := proc.sendAuthenticationRequest(); err != nil {
		err = utils.WrapError("Send NAS AuthenticationRequest", err)
		proc.complete(err)
		return
	} else {
		proc.Infof("AuthenticationRequest is sent to UE")
		proc.t3560.Start()
		proc.t3560Cnt++
	}
}

// handle AuthenticationRequest timer expiring
func (proc *AuthProc) HandleT3560() {
	if proc.t3560Cnt >= MAX_T3560_CNT {
		//stop authentication
		proc.complete(fmt.Errorf("T3560 expired"))
	} else {
		proc.Warnf("T3560 expired")
		//re-send authentication request
		proc.challenge()
	}
}

// read authentication context returned from AUSF
func (proc *AuthProc) loadAuthCtx(ctx *models.UEAuthenticationCtx) (err error) {
	proc.authType = ctx.AuthType
	if proc.authType == models.AUTHTYPE_5G_AKA {
		dat := ctx.FiveGAuthData.Av5gAka
		if dat == nil {
			return fmt.Errorf("Empty 5G AKA authentivation vector")
		}
		if proc.rand, err = hex.DecodeString(dat.Rand); err != nil {
			err = utils.WrapError("Decode RAN from hex string", err)
			return
		}

		if proc.autn, err = hex.DecodeString(dat.Autn); err != nil {
			err = utils.WrapError("Decode AUTN from hex string", err)
			return
		}

		if proc.hxResstar, err = hex.DecodeString(dat.HxresStar); err != nil {
			err = utils.WrapError("Decode hased RessStar from hex string", err)
			return
		}
	} else {
		eapMsg := ctx.FiveGAuthData.EapPayload
		if len(eapMsg) == 0 {
			return fmt.Errorf("Empty 5G EAP payload")
		}
		if proc.eap, err = base64.StdEncoding.DecodeString(eapMsg); err != nil {
			err = utils.WrapError("Decode base64 EAP string", err)
			return
		}
	}
	return
}

// Kamf Derivation function defined in TS 33.501 Annex A.7
func (proc *AuthProc) createKamf(receivedSupi string, kseaf string) (err error) {
	//decode KSEAF
	var kseafBytes []byte
	if kseafBytes, err = hex.DecodeString(kseaf); err != nil {
		return utils.WrapError("Decode KSEAF", err)
	}

	//check SUPI validity
	parts := strings.Split(receivedSupi, "-")
	if len(parts) != 2 || parts[0] != "imsi" {
		return fmt.Errorf("Wrong supi format")
	}
	proc.supi = receivedSupi

	//derive KAMF
	if proc.kamf, err = sec5g.KAMF(kseafBytes, []byte(proc.supi[5:]), proc.abba); err != nil {
		return utils.WrapError("Derive KAMF", err)
	}
	proc.Infof("KAMF is derived")
	return
}
