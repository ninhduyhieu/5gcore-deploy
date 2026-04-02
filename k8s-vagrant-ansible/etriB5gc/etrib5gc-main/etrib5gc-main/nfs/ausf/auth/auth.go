package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/internal/datastore"
	"etrib5gc/mesh"
	ausfctx "etrib5gc/nfs/ausf/context"
	"fmt"
	"github.com/reogac/sbi/apis/udm/ueauth"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/sec5g"
	"time"
)

type AuthContext struct {
	Supi     string
	Rand     []byte
	XresStar string
	Kseaf    []byte
	SrvNet   string
	EapId    uint8
	AuthType models.AuthType
}

//encode to binary for redis
func encodeAuthCtx(authCtx *AuthContext) ([]byte, error) {
	return json.Marshal(authCtx)
}

//decode from binary
func decodeAuthCtx(buf []byte, authCtx *AuthContext) error {
	return json.Unmarshal(buf, authCtx)
}

type TmpAuthContext struct {
	AuthId string
	Rand   string
}

//encode to binary for redis
func encodeTmpAuthCtx(authCtx *TmpAuthContext) ([]byte, error) {
	return json.Marshal(authCtx)
}

//decode from binary
func decodeTmpAuthCtx(buf []byte, authCtx *TmpAuthContext) error {
	return json.Unmarshal(buf, authCtx)
}

func loadRand(ctx context.Context, supiOrSuci string) (string, string, error) {
	ow := datastore.NewObjectLoader(supiOrSuci, encodeTmpAuthCtx, decodeTmpAuthCtx)
	if v, err := datastore.ReadObject2(_store, ctx, ow); err != nil {
		return "", "", utils.WrapError("Load RAND from data store", err)
	} else {
		log.Tracef("RAND %v is load for %s", *v, supiOrSuci)
		return v.Rand, v.AuthId, nil
	}
}

func loadAuthCtx(ctx context.Context, authId string) (*AuthContext, error) {
	ow := datastore.NewObjectLoader(authId, encodeAuthCtx, decodeAuthCtx)
	if v, err := datastore.ReadObject2(_store, ctx, ow); err != nil {
		return nil, utils.WrapError("Load AuthContext from data store", err)
	} else {
		log.Tracef("AuthContext for %s is loaded", authId)
		return v, nil
	}
}

func createAuthCtx(ctx context.Context, authId string, req *models.AuthenticationInfo, info *models.AuthenticationInfoResult) (*models.UEAuthenticationCtx, error) {
	av, err := decodeAv(info.AuthenticationVector)
	if err != nil {
		return nil, utils.WrapError("Decode authentication vector", err)
	}
	srvNet := req.ServingNetworkName
	rsp := &models.UEAuthenticationCtx{
		ServingNetworkName: srvNet,
		AuthType:           info.AuthType,
		//Links:              make(map[string]models.LinksValueSchema),
	}

	if info.AuthType == models.AUTHTYPE_EAP_AKA_PRIME {
		/*
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

			rsp.FiveGAuthData.EapPayload = ue.eapPayload(ue.network, body.ResynchronizationInfo != nil)
					//rsp.Links["eap-session"] = models.LinksValueSchema{
					//		Href: link + "/eap-session",
					//
		*/
		return nil, fmt.Errorf("EAP-AKA-PRIME not implemented")
	} else {
		// Derive Kseaf from Kausf
		var kSeaf []byte
		P0 := []byte(srvNet)
		if kSeaf, err = sec5g.SeafKey(av.kAusf, P0); err != nil {
			return nil, utils.WrapError("Derive KSEAF from KAUSF", err)
		}
		h := sha256.Sum256(append(av.rand, av.xResStar...))
		rsp.FiveGAuthData.Av5gAka = &models.Av5gAka{
			Rand:      hex.EncodeToString(av.rand),
			Autn:      hex.EncodeToString(av.autn),
			HxresStar: hex.EncodeToString(h[16:]), // last 128 bits
		}
		log.Infof("AuthId %s have auth vector: rand[%x] - autn[%x] - xresstar[%x]", authId, av.rand, av.autn, av.xResStar)
		ueCtx := &AuthContext{
			AuthType: info.AuthType,
			SrvNet:   srvNet,
			Supi:     info.Supi,
			Rand:     av.rand,
			Kseaf:    kSeaf,
			XresStar: hex.EncodeToString(av.xResStar),
		}

		//save RAND by supi
		ow1 := datastore.NewObjectWriter(req.SupiOrSuci, &TmpAuthContext{
			Rand: hex.EncodeToString(av.rand), AuthId: authId,
		}, encodeTmpAuthCtx)
		if err := datastore.WriteObject2(_store, ctx, ow1); err != nil {
			return nil, utils.WrapError("Save RAND to data store", err)
		}
		log.Infof("AuthId %s and Rand %x is save for %s", authId, av.rand, ueCtx.Supi)

		//save AuthCtx by AuthId
		ow2 := datastore.NewObjectWriter(authId, ueCtx, encodeAuthCtx)
		if err := datastore.WriteObject2(_store, ctx, ow2); err != nil {
			return nil, utils.WrapError("Write AuthContext to data store", err)
		}
		log.Infof("AuthContext is save for %s", authId)
		return rsp, nil
	}
}

func HandleAuthenticationPost(ctx context.Context, body *models.AuthenticationInfo) (string, *models.UEAuthenticationCtx, error) {
	req := models.AuthenticationInfoRequest{
		ServingNetworkName: body.ServingNetworkName,
	}
	sid := common.UdmServiceName(ausfctx.PlmnId())
	udmcli, err := mesh.Consumer(sid, nil)
	if err != nil {
		return "", nil, utils.WrapError("Create UDM client", err)
	}
	var authId, rand string
	if reSync := body.ResynchronizationInfo; reSync != nil {
		if rand, authId, err = loadRand(ctx, body.SupiOrSuci); err != nil {
			return "", nil, utils.WrapError("Load previous RAND for SQN resynchronization", err)
		} else if len(rand) == 0 {
			return "", nil, fmt.Errorf("Last authentication context not found for SQN re-synchronization")
		} else {
			if len(reSync.Rand) == 0 {
				reSync.Rand = rand
			}
		}
		req.ResynchronizationInfo = reSync
	}

	log.Debugf("Request authentication vector from UDM for %s", body.SupiOrSuci)
	if info, err := ueauth.GenerateAuthData(udmcli, body.SupiOrSuci, &req); err != nil {
		return "", nil, utils.WrapError("Get authentication vector from UDM", err)
	} else {
		log.Debugf("Receive authentication vector from UDM for %s", body.SupiOrSuci)
		if len(authId) == 0 {
			authId = fmt.Sprintf("auth-%d", ausfctx.AllocateAuthId())
		}
		if rsp, err := createAuthCtx(ctx, authId, body, info); err != nil {
			return "", nil, utils.WrapError("Create authentication context", err)
		} else {
			return authId, rsp, nil
		}
	}
}

func HandleEapSession(ctx context.Context, authId string, body *models.EapSession) (*models.EapSession, error) {
	if ue, err := loadAuthCtx(ctx, authId); err != nil {
		return nil, utils.WrapError("Load authentication context", err)
	} else if ue == nil {
		return nil, fmt.Errorf("Authentication context not found for %s", authId)
	} else {
		//TODO: handle eap
		return nil, nil
	}
}

func Handle5gAkaConfirmation(ctx context.Context, authId string, body *models.ConfirmationData) (*models.ConfirmationDataResponse, error) {
	if ue, err := loadAuthCtx(ctx, authId); err != nil {
		return nil, utils.WrapError("Load authentication context", err)
	} else if ue == nil {
		return nil, fmt.Errorf("Authentication context not found for %s", authId)
	} else {
		if body.ResStar == ue.XresStar {
			log.Infof("UE %s is authenticated", ue.Supi)
			sid := common.UdmServiceName(ausfctx.PlmnId())
			udmcli, err := mesh.Consumer(sid, nil)
			if err != nil {
				return nil, utils.WrapError("Create UDM client", err)
			}

			req := &models.AuthEvent{
				TimeStamp:          time.Now().Format(time.UnixDate),
				AuthType:           ue.AuthType,
				Success:            true,
				ServingNetworkName: ue.SrvNet,
			}
			if _, _, err := ueauth.ConfirmAuth(udmcli, ue.Supi, req); err != nil {
				return nil, fmt.Errorf("Send confirmation to UDM", err)
			} else {
				return &models.ConfirmationDataResponse{
					AuthResult: models.AUTHRESULT_AUTHENTICATION_SUCCESS,
					Supi:       ue.Supi,
					Kseaf:      hex.EncodeToString(ue.Kseaf),
				}, nil
			}
		} else {
			return nil, fmt.Errorf("Mismatched resstar")
		}
	}
}
