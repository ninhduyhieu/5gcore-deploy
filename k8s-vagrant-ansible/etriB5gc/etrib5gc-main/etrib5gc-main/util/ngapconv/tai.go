package ngapconv

/*
import (
	"encoding/hex"

	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func TaiToModels(tai ies.TAI) models.Tai {
	var modelsTai models.Tai

	plmnID := PlmnIdToModels(*tai.PLMNIdentity)
	modelsTai.PlmnId = plmnID
	modelsTai.Tac = hex.EncodeToString(tai.TAC.Value)

	return modelsTai
}

func TaiToNgap(tai models.Tai) (ngapTai ies.TAI, err error) {

	if ngapTai.PLMNIdentity, err = PlmnIdToNgap(tai.PlmnId); err != nil {
		return
	}
	var tac []byte
	if tac, err = hex.DecodeString(tai.Tac); err != nil {
		err = utils.WrapError("Decode TAC failed", err)
		return
	} else {
		ngapTai.TAC.Value = tac
	}
	return
}
*/
