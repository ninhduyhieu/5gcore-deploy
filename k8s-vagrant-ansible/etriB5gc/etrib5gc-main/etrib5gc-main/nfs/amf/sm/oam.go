package sm

import (
	"etrib5gc/nfs/amf/types"
	"github.com/reogac/sbi/models"
)

// Write OAM SmContext information
func (smCtx *SmContext) WriteInfo() (info types.SmContextInfo) {
	info = types.SmContextInfo{
		Dnn:          smCtx.dnn,
		Slice:        models.SnssaiToString(smCtx.snssai.AllowedSnssai),
		Is3Gpp:       smCtx.isGpp,
		PduSessionId: int(smCtx.id),
		SmCtxRef:     smCtx.ref,
	}

	return
}
