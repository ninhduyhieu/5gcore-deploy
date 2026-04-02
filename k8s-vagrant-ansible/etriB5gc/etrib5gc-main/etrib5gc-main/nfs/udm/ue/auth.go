package ue

import (
	"context"
	"encoding/hex"
	"etrib5gc/internal/datastore"
	udmcontext "etrib5gc/nfs/udm/context"
	"etrib5gc/util/suci"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/sec5g"
)

type AuthContext struct {
	Amf, Enc, Sqn, Op []byte
	IsOpc             bool
	Type              models.AuthType
	milenage          *sec5g.Milenage
	sqn               Sqn
	sub               *models.AuthenticationSubscription
}

func GenerateAuthData(ctx context.Context, suciOrSupi string, req *models.AuthenticationInfoRequest) (rsp *models.AuthenticationInfoResult, err error) {
	var supi string
	//NOTE: handle the case where AUSF send a supi in the form that is not
	//in SUCI-IMSI for mat
	supi = suciOrSupi
	if !isSupi(suciOrSupi) {
		if supi, err = suci.RecoverSupi(suciOrSupi, udmcontext.SuciProfiles()); err != nil {
			err = utils.WrapError("Recovering SUPI", err)
			return
		}
	}
	//load AuthContext
	ow := datastore.NewObjectLoader[AuthContext](supi, nil, nil)

	var ueCtx *AuthContext
	if ueCtx, err = datastore.ReadObject(_store, ctx, ow, newUe(supi, nil).readAuthContext); err != nil {
		err = utils.WrapError("Load authentication context for "+supi, err)
		return
	}

	if reSync := req.ResynchronizationInfo; reSync != nil {
		log.Tracef("SQN of %s before sync: %s", ueCtx.sub.Supi, ueCtx.sqn.String())

		//NOTE: rand is set during resync
		if err = ueCtx.reSync(reSync.Auts, reSync.Rand); err != nil {
			err = utils.WrapError("SQN resynchronize", err)
			return
		}
		log.Tracef("SQN for %s is resynchronized", ueCtx.sub.Supi)
	} else {
		//update RAND
		ueCtx.milenage.Refresh()
		//increate sqn
		ueCtx.sqn.inc()
	}
	var v *models.AuthenticationVector
	if v, err = ueCtx.createAuthenticationVector(req.ServingNetworkName); err != nil {
		err = utils.WrapError("Generate authentication vector", err)
		return
	}
	rsp = &models.AuthenticationInfoResult{
		AuthType:             ueCtx.Type,
		Supi:                 supi,
		AuthenticationVector: v,
	}
	log.Infof("Authentication data is generated for %s", supi)

	if err = datastore.WriteObject(_store, ctx, ow, newUe(supi, nil).writeAuthContext); err != nil {
		err = utils.WrapError("Write authentication context", err)
	}
	return
}

func HandleConfirmAuth(supi string, dat *models.AuthEvent) (rsp *models.AuthEvent, err error) {
	rsp = dat
	if dat.Success {
		log.Infof("Authentication for %s succeeded", supi)
	} else {
		log.Infof("Authentication for %s failed", supi)
	}
	return
}

func (ueCtx *AuthContext) reSync(austStr string, randStr string) (err error) {
	var auts, ueRand []byte
	if auts, err = hex.DecodeString(austStr); err != nil {
		return utils.WrapError("Decode AUTS", err)
	}

	if ueRand, err = hex.DecodeString(randStr); err != nil {
		return utils.WrapError("Decode UE's RAND", err)
	}
	var sqn [6]byte
	if sqn, err = ueCtx.milenage.ValidateAuts(auts, ueRand); err != nil {
		return utils.WrapError("Validate AUTS", err)
	} else {
		ueCtx.sqn.reset(sqn[:])
	}
	return
}

func (ueCtx *AuthContext) createAuthenticationVector(servingNetwork string) (*models.AuthenticationVector, error) {
	sqn := ueCtx.sqn.bytes()
	ueRand := ueCtx.milenage.GetRand()

	maca, _, _ := ueCtx.milenage.F1(sqn, ueCtx.Amf[:])
	res, ak := ueCtx.milenage.F2F5()
	ck := ueCtx.milenage.F3()
	ik := ueCtx.milenage.F4()

	//calculate autn
	sqnXORak := make([]byte, 6)
	for i := 0; i < 6; i++ {
		sqnXORak[i] = sqn[i] ^ ak[i]
	}
	autn := append(append(sqnXORak, ueCtx.Amf[:]...), maca...)

	key := append(ck, ik...)

	var xresStar []byte
	var err error
	v := new(models.AuthenticationVector)
	if ueCtx.Type == models.AUTHTYPE_5G_AKA {
		av := new(models.Av5GHeAka)
		v.Av5GHeAka = av

		av.Rand = hex.EncodeToString(ueRand)
		av.Autn = hex.EncodeToString(autn)

		if _, xresStar, err = sec5g.ResstarXresstar(key, []byte(servingNetwork), ueRand, res); err != nil {
			return nil, utils.WrapError("Generate XRESSTAR", err)
		}
		var kAusf []byte
		if kAusf, err = sec5g.KAUSF(key, []byte(servingNetwork), sqnXORak); err != nil {
			return nil, utils.WrapError("Generate KAUSF", err)
		}

		av.XresStar = hex.EncodeToString(xresStar)
		av.Kausf = hex.EncodeToString(kAusf)
		av.AvType = models.AVTYPE_5G_HE_AKA

	} else if ueCtx.Type == models.AUTHTYPE_EAP_AKA_PRIME {
		av := new(models.AvEapAkaPrime)
		v.AvEapAkaPrime = av

		var ckPrime, ikPrime []byte
		if ckPrime, ikPrime, err = sec5g.CkPrimeIkPrime(key, []byte(servingNetwork), sqnXORak); err != nil {
			return nil, utils.WrapError("Generate Ck' and Ik'", err)
		}
		av.Rand = hex.EncodeToString(ueRand)
		av.CkPrime = hex.EncodeToString(ckPrime)
		av.IkPrime = hex.EncodeToString(ikPrime)
		av.AvType = models.AVTYPE_EAP_AKA_PRIME
		av.Xres = hex.EncodeToString(res)
	} else {
		//should never happen
		return nil, fmt.Errorf("Unsupported authentication type")
	}
	return v, nil
}

func createAuthenticationContext(sub *models.AuthenticationSubscription) (ueCtx *AuthContext, err error) {
	var k, opopc, amf, sqn []byte
	if k, err = hex.DecodeString(sub.EncPermanentKey); err != nil {
		return
	}
	if amf, err = hex.DecodeString(sub.AuthenticationManagementField); err != nil {
		return
	}

	if sqn, err = hex.DecodeString(sub.SequenceNumber.Sqn); err != nil {
		return
	}
	var isOpc bool = false
	if len(sub.EncOpcKey) > 0 {
		if opopc, err = hex.DecodeString(sub.EncOpcKey); err != nil {
			return
		}
		isOpc = true
	} else if len(sub.EncTopcKey) > 0 {
		if opopc, err = hex.DecodeString(sub.EncTopcKey); err != nil {
			return
		}
	} else {
		err = fmt.Errorf("OP/OPC is missing from subscription data")
		return
	}

	if len(sqn) != 6 || len(amf) != 2 {
		err = fmt.Errorf("Input size for SQN or AMF is incorrect")
		return
	}
	var authType models.AuthType
	switch sub.AuthenticationMethod {
	case models.AUTHMETHOD_5G_AKA:
		authType = models.AUTHTYPE_5G_AKA
	case models.AUTHMETHOD_EAP_AKA_PRIME:
		authType = models.AUTHTYPE_EAP_AKA_PRIME
	case models.AUTHMETHOD_EAP_TLS:
		authType = models.AUTHTYPE_EAP_TLS
	default:
		err = fmt.Errorf("Unsupported Auth method")
		return
	}

	log.Infof("Create Authentication Context for %s", sub.Supi)
	ueCtx = &AuthContext{
		Enc:   k,
		Op:    opopc,
		Sqn:   sqn,
		IsOpc: isOpc,
		Amf:   amf,
		Type:  authType,
		sub:   sub,
	}
	if ueCtx.milenage, err = sec5g.NewMilenage(k, opopc, isOpc); err != nil {
		return
	}
	ueCtx.sqn.set(sqn)
	return ueCtx, nil
}
