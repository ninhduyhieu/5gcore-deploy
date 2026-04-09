package ue

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"etrib5gc/logctx"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/udm/ueauth"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/sec5g"
	"github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"time"

	"github.com/bronze1man/radius"
)

type UeContext struct {
	*logrus.Entry
	authId   string //supi or suci
	supi     string
	network  string
	kAusf    []byte
	kSeaf    []byte
	rand     []byte
	xResStar []byte
	autn     []byte
	xRes     []byte
	kAut     []byte
	eapId    uint8
	ckPrime  string
	ikPrime  string

	authType models.AuthType
	udmcli   sbi.ConsumerClient
	amData   *models.AccessAndMobilitySubscriptionData
}

func New(authId string, snName string, cli sbi.ConsumerClient) *UeContext {
	return &UeContext{
		Entry: logctx.Entry(logrus.Fields{
			"authId": authId,
			"mod":    "ue",
		}),
		authId:  authId,
		network: snName,
		udmcli:  cli,
	}
}

func (ue *UeContext) update(info *models.AuthenticationInfoResult) (err error) {
	ue.authType = info.AuthType
	var av AuthVector
	if err = av.decode(info.AuthenticationVector); err != nil {
		return
	}
	if info.AuthType == models.AUTHTYPE_5G_AKA {
		err = ue.update5gAka(&av)
	} else if info.AuthType == models.AUTHTYPE_EAP_AKA_PRIME {
		err = ue.updateEapAka(&av, info.Supi)
	} else {
		//unknown type
		err = fmt.Errorf("Not supported authentication method %s", info.AuthType)
	}
	if err == nil {
		ue.supi = info.Supi
		ue.amData = info.AmData
		ue.Entry = ue.Entry.WithFields(logrus.Fields{
			"supi": ue.supi,
		})
	}
	return
}

func (ue *UeContext) AmData() *models.AccessAndMobilitySubscriptionData {
	return ue.amData
}

func (ue *UeContext) update5gAka(av *AuthVector) (err error) {
	// Derive Kseaf from Kausf
	var kSeaf []byte
	P0 := []byte(ue.network)
	if kSeaf, err = sec5g.SeafKey(av.kAusf, P0); err != nil {
		ue.Errorf("GetKDFValue failed: %+v", err)
		return
	}
	//ue.Info("KSEAF is generated")
	ue.xResStar = av.xResStar
	ue.kAusf = av.kAusf
	ue.kSeaf = kSeaf
	ue.rand = av.rand
	ue.autn = av.autn
	ue.Tracef("autn=%s", hex.EncodeToString(ue.autn))
	return
}

func (ue *UeContext) updateEapAka(info *AuthVector, supi string) (err error) {
	//change 20dec
	identity := supi
	_, K_aut, _, _, EMSK := eapAkaPrimePrf(hex.EncodeToString(info.ikPrime), hex.EncodeToString(info.ckPrime), identity)
	ue.kAut = K_aut
	Kausf := EMSK[0:32]
	ue.kAusf = Kausf
	var kSeaf []byte
	P0 := []byte(ue.network)
	if kSeaf, err = sec5g.SeafKey(ue.kAusf, P0); err != nil {
		ue.Errorf("GetKDFValue failed: %+v", err)
		return
	}
	ue.kSeaf = kSeaf
	ue.xRes = info.xRes
	ue.rand = info.rand
	ue.autn = info.autn
	ue.ikPrime = hex.EncodeToString(info.ikPrime)
	ue.ckPrime = hex.EncodeToString(info.ckPrime)
	return
}

func (ue *UeContext) Supi() string {
	return ue.supi
}

func (ue *UeContext) AuthId() string {
	return ue.authId
}
func (ue *UeContext) Kseaf() string {
	return hex.EncodeToString(ue.kSeaf)
}

func (ue *UeContext) Xres() string {
	return hex.EncodeToString(ue.xRes)
}

func (ue *UeContext) Autn() string {
	return hex.EncodeToString(ue.autn)
}

func (ue *UeContext) Rand() string {
	return hex.EncodeToString(ue.rand)
}

func (ue *UeContext) Kaut() string {
	return hex.EncodeToString(ue.kAut)
}

func (ue *UeContext) EapId() uint8 {
	return 0
}

func (ue *UeContext) eapPayload(snname string, resync bool) (dat string) {
	if ue.authType == models.AUTHTYPE_5G_AKA {
		return ""
	}
	var eapPkt radius.EapPacket
	eapPkt.Code = radius.EapCode(1)
	if resync {
		rand.Seed(time.Now().Unix())
		randIdentifier := rand.Intn(256)
		ue.eapId = uint8(randIdentifier)
	} else {
		ue.eapId = ue.EapId() + 1
	}
	eapPkt.Identifier = ue.eapId
	eapPkt.Type = radius.EapType(50) // according to RFC5448 6.1
	var eapAKAHdr, atRand, atAutn, atKdf, atKdfInput, atMAC string
	if atRandTmp, err := EapEncodeAttribute("AT_RAND", ue.Rand()); err != nil {
		ue.Errorf("EAP encode RAND failed: %+v", err)
	} else {
		atRand = atRandTmp
	}
	if atAutnTmp, err := EapEncodeAttribute("AT_AUTN", ue.Autn()); err != nil {
		ue.Errorf("EAP encode AUTN failed: %+v", err)
	} else {
		atAutn = atAutnTmp
	}
	if atKdfTmp, err := EapEncodeAttribute("AT_KDF", snname); err != nil {
		ue.Errorf("EAP encode KDF failed: %+v", err)
	} else {
		atKdf = atKdfTmp
	}
	if atKdfInputTmp, err := EapEncodeAttribute("AT_KDF_INPUT", snname); err != nil {
		ue.Errorf("EAP encode KDF failed: %+v", err)
	} else {
		atKdfInput = atKdfInputTmp
	}
	if atMACTmp, err := EapEncodeAttribute("AT_MAC", ""); err != nil {
		ue.Errorf("EAP encode MAC failed: %+v", err)
	} else {
		atMAC = atMACTmp
	}
	eapAKAHdrBytes := make([]byte, 3) // RFC4187 8.1
	eapAKAHdrBytes[0] = AKA_CHALLENGE_SUBTYPE
	eapAKAHdr = string(eapAKAHdrBytes)
	dataArrayBeforeMAC := eapAKAHdr + atRand + atAutn + atMAC + atKdfInput + atKdf
	eapPkt.Data = []byte(dataArrayBeforeMAC)
	encodedPktBeforeMAC := eapPkt.Encode()
	MacValue := ue.calculateAtMAC(ue.kAut, encodedPktBeforeMAC)
	MacLen, _ := EapEncodeAttribute("MAC_LEN", "")
	atMAC = MacLen[:4] + string(MacValue)
	//ue.Infof("Mac value before send: %s",hex.EncodeToString([]byte(atMAC)))
	dataArrayAfterMAC := eapAKAHdr + atRand + atAutn + atKdf + atKdfInput + atMAC
	eapPkt.Data = []byte(dataArrayAfterMAC)
	encodedPktAfterMAC := eapPkt.Encode()
	dat = base64.StdEncoding.EncodeToString(encodedPktAfterMAC)
	return dat
}

func (ue *UeContext) var5gAuthData(snname string, resync bool) *models.Av5gAka {
	if ue.authType != models.AUTHTYPE_5G_AKA {
		return nil
	}
	var dat models.Av5gAka
	dat.Rand = hex.EncodeToString(ue.rand)
	dat.Autn = hex.EncodeToString(ue.autn)
	// Derive HXRES* from XRES*
	ue.Tracef("value Xresstar : %s", hex.EncodeToString(ue.xResStar))
	h := sha256.Sum256(append(ue.rand, ue.xResStar...))
	dat.HxresStar = hex.EncodeToString(h[16:]) // last 128 bits
	return &dat
}

func (ue *UeContext) checkResStar(resstar string) bool {
	return strings.Compare(resstar, hex.EncodeToString(ue.xResStar)) == 0
}

func (ue *UeContext) HandleAuthenticationPost(body *models.AuthenticationInfo) (rsp *models.UEAuthenticationCtx, err error) {

	req := models.AuthenticationInfoRequest{
		ServingNetworkName: ue.network,
		FetchUeAmData:      body.FetchUeAmData,
	}
	if resync := body.ResynchronizationInfo; resync != nil {
		if len(resync.Rand) == 0 {
			resync.Rand = hex.EncodeToString(ue.rand)
		}
		req.ResynchronizationInfo = resync
	}
	ue.Debugf("Request authentication vector from UDM")
	authId := ue.authId
	if len(ue.supi) > 0 {
		ue.Tracef("Request authentication vector with SUPI")
		authId = ue.supi
	}
	var info *models.AuthenticationInfoResult
	if info, err = ueauth.GenerateAuthData(ue.udmcli, authId, &req); err != nil {
		ue.Errorf("Fails to get authentication vector: %+v", err)
		err = utils.WrapError("Get authentication vector", err)
		return
	} else {
		ue.Debugf("Receive authentication vector")
		if err = ue.update(info); err != nil {
			ue.Errorf("Failed to update UeContext: %+v", err)
			return
		} else {
			ue.Infof("Authentication vector generated")
			rsp = &models.UEAuthenticationCtx{
				ServingNetworkName: ue.network,
				AuthType:           info.AuthType,
				//Links:              make(map[string]models.LinksValueSchema),
			}
			//there are only 2 auth types (constrained by the previous code
			//block)
			//link := p.ctx.Url() + "/nausf-auth/v1/ue-authentications/" + ueid
			if info.AuthType == models.AUTHTYPE_EAP_AKA_PRIME {
				rsp.FiveGAuthData.EapPayload = ue.eapPayload(ue.network, body.ResynchronizationInfo != nil)
				//rsp.Links["eap-session"] = models.LinksValueSchema{
				//		Href: link + "/eap-session",
				//	}
			} else {
				rsp.FiveGAuthData.Av5gAka = ue.var5gAuthData(ue.network, body.ResynchronizationInfo != nil)
				//	rsp.Links["5g-aka"] = models.LinksValueSchema{
				//		Href: link + "/5g-aka-confirmation",
				//	}
			}
		}
	}

	return
}

func (ue *UeContext) confirmAuth2Udm(success bool) (err error) {
	dat := &models.AuthEvent{
		TimeStamp:          time.Now().Format(time.UnixDate),
		AuthType:           ue.authType,
		Success:            success,
		ServingNetworkName: ue.network,
	}
	_, _, err = ueauth.ConfirmAuth(ue.udmcli, ue.supi, dat)
	return
}
