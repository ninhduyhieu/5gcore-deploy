package ranue

import (
	"context"
	"etrib5gc/mesh"
	amfctx "etrib5gc/nfs/amf/context"
	"fmt"
	"github.com/reogac/sbi/apis/pran/handover"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
)

func (ranUe *RanUe) createHandoverTarget(msg *models.HandoverRequired) *RanUe {
	var gnb *amfctx.Ran
	if gnb = amfctx.FindRan(&msg.TargetId); gnb == nil {
		ranUe.Errorf("Fail to find target Ran")
		return nil
	}

	return &RanUe{
		ue:      ranUe.ue,
		isGpp:   ranUe.isGpp,
		ranCli:  gnb.Client(),
		ranInfo: gnb.RanInfo(),
		metrics: copyRanUeMetrics(ranUe.metrics),
		Entry:   log,
	}
}

func (ranUe *RanUe) ReceiveHandoverRequired(ctx context.Context, msg *models.HandoverRequired) (*models.HandoverCommand, *models.HandoverPreparationFailure, error) {
	if target := ranUe.createHandoverTarget(msg); target == nil {
		return nil, nil, fmt.Errorf("Invalid target for handover")
	} else {
		return ranUe.ue.HandleHandoverRequired(ranUe.isGpp, target, msg)
	}
}

func (ranUe *RanUe) ReceiveHandoverNotify(ctx context.Context, msg *models.HandoverNotify) error {
	return ranUe.ue.HandleHandoverNotify(ranUe.isGpp, msg)
}

func (ranUe *RanUe) ReceiveHandoverCancel(ctx context.Context, msg *models.HandoverCancel) (*models.HandoverCancelAcknowledge, error) {
	return ranUe.ue.HandleHandoverCancel(ranUe.isGpp, msg)
}

func (ranUe *RanUe) SendHandoverRequest(req *models.HandoverRequest) (*models.HandoverRequestAcknowledge, *models.HandoverRequestFailure, error) {
	rsp, ersp, err := handover.HandoverRequest(ranUe.ranCli, mesh.EndpointInfo(), req)
	if rsp != nil {
		//update RanUe with information from the target gnB
		ranUe.ranNets = rsp.RanNets
		ranUe.nasSplit = rsp.NasSplit
		ranUe.ranUeId = rsp.RanUeId
		ranUe.Entry = ranUe.Entry.WithFields(logrus.Fields{
			"ranUeId": rsp.RanUeId.String(),
		})
	}
	return rsp, ersp, err
}
