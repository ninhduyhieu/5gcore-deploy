package ngapconv

import (
	"encoding/hex"

	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func SNssaiToModels(ngapSnssai ies.SNSSAI) (modelsSnssai models.Snssai) {
	modelsSnssai.Sst = int(ngapSnssai.SST[0])
	if ngapSnssai.SD != nil {
		modelsSnssai.Sd = hex.EncodeToString(ngapSnssai.SD)
	}
	return
}

func SNssaiToNgap(mId models.Snssai) (ngapId *ies.SNSSAI, err error) {
	ngapId = &ies.SNSSAI{
		SST: []byte{byte(mId.Sst)},
	}
	if len(mId.Sd) > 0 {
		if ngapId.SD, err = hex.DecodeString(mId.Sd); err != nil {
			ngapId = nil
			err = utils.WrapError("Decode snssai.sd failed", err)
			return
		}
	}
	return
}
