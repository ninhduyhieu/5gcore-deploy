package ue

import (
	"encoding/hex"
	"fmt"
	"github.com/reogac/sbi/models"
)

type AuthVector struct {
	rand, xRes, autn, ckPrime, ikPrime, xResStar, kAusf []byte
}

func (av *AuthVector) decode(dat *models.AuthenticationVector) (err error) {
	if info := dat.Av5GHeAka; info != nil {
		if len(info.Rand) > 0 {
			if av.rand, err = hex.DecodeString(info.Rand); err != nil {
				return
			}
		}

		if len(info.Autn) > 0 {
			if av.autn, err = hex.DecodeString(info.Autn); err != nil {
				return
			}
		}

		if len(info.XresStar) > 0 {
			if av.xResStar, err = hex.DecodeString(info.XresStar); err != nil {
				return
			}
		}

		if len(info.Kausf) > 0 {
			if av.kAusf, err = hex.DecodeString(info.Kausf); err != nil {
				return
			}
		}

	} else if info := dat.AvEapAkaPrime; info != nil {
		if len(info.Rand) > 0 {
			if av.rand, err = hex.DecodeString(info.Rand); err != nil {
				return
			}
		}
		if len(info.Xres) > 0 {
			if av.xRes, err = hex.DecodeString(info.Xres); err != nil {
				return
			}
		}

		if len(info.Autn) > 0 {
			if av.autn, err = hex.DecodeString(info.Autn); err != nil {
				return
			}
		}

		if len(info.CkPrime) > 0 {
			if av.ckPrime, err = hex.DecodeString(info.CkPrime); err != nil {
				return
			}
		}

		if len(info.IkPrime) > 0 {
			if av.ikPrime, err = hex.DecodeString(info.IkPrime); err != nil {
				return
			}
		}
	} else {
		err = fmt.Errorf("Empty authentication vector")
	}

	return
}
