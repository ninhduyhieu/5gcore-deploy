package producer

import (
	"etrib5gc/nfs/amf/ranue"
	"github.com/reogac/sbi/models"
)

func (p *producerImpl) HandleUeContextRelease(ueId int64, msg *models.UeContextReleaseRequest) *models.ProblemDetails {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle UeContextRelease", ueId)
		return notFoundProblem
	} else {
		if err := ranUe.ReceiveUeContextReleaseRequest(msg); err != nil {
			p.Errorf("Fail to handle UeContextReleaseRequest: %+v", err)
			return internalProblem
		} else {
			p.Info("UeContextReleaseRequest is handled")
		}
	}

	return nil
}

func (p *producerImpl) HandleRrcInactivityStatusReport(ueId int64, msg *models.RrcInactivityTransportReport) (prob *models.ProblemDetails) {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [AmfUeId=%d] to handle RrcInactTranReport", ueId)
		prob = notFoundProblem
		return
	} else {
		if err := ranUe.ReceiveRrcInactivityStatusReport(msg); err != nil {
			p.Errorf("Fail to handle RrcInactivityStatusReport: %+v", err)
			return internalProblem
		}
	}
	return nil
}

func (p *producerImpl) HandleN2SmInfoUplink(ueId int64, msg *models.N2SmInfoUplinkTransport) (prob *models.ProblemDetails) {
	if ranUe := ranue.FindRanUe(ueId); ranUe == nil {
		p.Errorf("RanUe not found [amfUeid=%d] to handle N2SmInfoUplink", ueId)
		prob = notFoundProblem
		return
	} else {
		if err := ranUe.ReceiveN2SmInfoUplink(msg); err != nil {
			p.Errorf("Fail to handle N2SmInfoUplink: %+v", err)
			return internalProblem
		}
	}
	return nil
}
