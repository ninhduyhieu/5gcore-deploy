package ranue

import (
	"context"
	"github.com/reogac/sbi/models"
)

func (ranUe *RanUe) ReceivePathswitch(ctx context.Context, callback *models.EndpointInfo, msg *models.PathSwitchRequest) (*models.PathSwitchAcknowledge, *models.PathSwitchFailure, error) {
	return ranUe.ue.DoPathSwitch(ranUe.isGpp, msg)
}
