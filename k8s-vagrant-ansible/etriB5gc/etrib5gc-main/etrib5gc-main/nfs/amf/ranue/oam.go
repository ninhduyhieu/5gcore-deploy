package ranue

import (
	"etrib5gc/nfs/amf/types"
	"time"
)

func (ranUe *RanUe) WriteInfo() (info *types.RanUeInfo) {
	m := ranUe.metrics
	info = &types.RanUeInfo{
		Access:                   ranUe.accessType(),
		RanUeId:                  ranUe.ranUeId,
		LocalId:                  ranUe.localId,
		CreatedTime:              m.createdTime.Format(time.RFC1123),
		LastRegistrationTime:     m.lastRegTime.Format(time.RFC1123),
		LastRegistrationDuration: m.lastRegDuration,
	}
	return
}
