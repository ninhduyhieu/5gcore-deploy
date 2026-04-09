package auth

import (
	"encoding/hex"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

type AuthVector struct {
	rand, xRes, autn, ckPrime, ikPrime, xResStar, kAusf []byte
}

func decodeAv(dat *models.AuthenticationVector) (*AuthVector, error) {
	var err error
	av := new(AuthVector)
	if info := dat.Av5GHeAka; info != nil {
		if av.rand, err = hex.DecodeString(info.Rand); err != nil {
			return nil, utils.WrapError("Decode RAND", err)
		} else if len(av.rand) == 0 {
			return nil, fmt.Errorf("Empty RAND")
		}

		if av.autn, err = hex.DecodeString(info.Autn); err != nil {
			return nil, utils.WrapError("Decode AUTN", err)
		} else if len(av.autn) == 0 {
			return nil, fmt.Errorf("Empty RAND")
		}

		if av.xResStar, err = hex.DecodeString(info.XresStar); err != nil {
			return nil, utils.WrapError("Decode XRESSTAR", err)
		} else if len(av.xResStar) == 0 {
			return nil, fmt.Errorf("Empty XRESSTAR")
		}

		if av.kAusf, err = hex.DecodeString(info.Kausf); err != nil {
			return nil, utils.WrapError("Decode KAUSF", err)
		} else if len(av.kAusf) == 0 {
			return nil, fmt.Errorf("Empty KAUSF")
		}
	} else if info := dat.AvEapAkaPrime; info != nil {
		if av.rand, err = hex.DecodeString(info.Rand); err != nil {
			return nil, utils.WrapError("Decode RAND", err)
		} else if len(av.rand) == 0 {
			return nil, fmt.Errorf("Empty RAND")
		}
		if av.xRes, err = hex.DecodeString(info.Xres); err != nil {
			return nil, utils.WrapError("Decode XRES", err)
		} else if len(av.xRes) == 0 {
			return nil, fmt.Errorf("Empty XRES")
		}

		if av.autn, err = hex.DecodeString(info.Autn); err != nil {
			return nil, utils.WrapError("Decode AUTN", err)
		} else if len(av.autn) == 0 {
			return nil, fmt.Errorf("Empty RAND")
		}

		if av.ckPrime, err = hex.DecodeString(info.CkPrime); err != nil {
			return nil, utils.WrapError("Decode CKPRIME", err)
		} else if len(av.ckPrime) == 0 {
			return nil, fmt.Errorf("Empty CKPRIME")
		}

		if av.ikPrime, err = hex.DecodeString(info.IkPrime); err != nil {
			return nil, utils.WrapError("Decode IKPRIME", err)
		} else if len(av.ikPrime) == 0 {
			return nil, fmt.Errorf("Empty IKPRIME")
		}
	} else {
		return nil, fmt.Errorf("Empty authentication vector")
	}

	return av, nil
}
