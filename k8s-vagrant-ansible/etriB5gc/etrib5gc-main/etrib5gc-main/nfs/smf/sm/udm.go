package sm

import (
	"fmt"
	"github.com/reogac/sbi/apis/udm/sdm"
	"github.com/reogac/utils"
)

func (smCtx *SmContext) getSmData() error {
	smCtx.Debug("Get SM data from UDM")
	params := sdm.GetSmDataParams{
		SingleNssai: smCtx.snssai,
		Dnn:         smCtx.dnn,
		PlmnId:      smCtx.homePlmnId,
		Supi:        smCtx.supi,
	}
	if _, rsp, err := sdm.GetSmData(smCtx.udmCli, params); err != nil {
		return utils.WrapError("GetSmData", err)
	} else {
		if len(rsp.SessionManagementSubscriptionData) > 0 {
			smCtx.smData = &rsp.SessionManagementSubscriptionData[0] //just take the first item
		} else {
			return fmt.Errorf("No SessionSubscription data found")
		}
	}
	return nil
}
