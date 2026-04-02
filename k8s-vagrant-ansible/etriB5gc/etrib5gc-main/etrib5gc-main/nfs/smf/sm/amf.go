package sm

import (
	"github.com/reogac/sbi/apis/amf/callback"
	"github.com/reogac/sbi/models"
)

func (smctx *SmContext) notifyAmf() (err error) {
	smctx.Tracef("Notify AMF of SmContext removal")
	body := &models.SmContextStatusNotification{}
	err = callback.SmContextStatusNotify(smctx.amfCli, callback.SmContextStatusNotifyParams{
		Supi:      smctx.supi,
		SessionId: int16(smctx.pduSessionId),
	}, body)
	return
}
