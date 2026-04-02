package dataplane

import (
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/sbi/models"
)

type QERBuilder interface {
	buildQERs(uint8, *PfcpSession) []*QER
}

type qosQERBuilder struct {
	qosData *models.QosData
}

func (b *qosQERBuilder) buildQERs(qfi uint8, s *PfcpSession) (qerList []*QER) {
	newQer := &QER{
		qfi:        qfi,
		id:         s.upf.qerIdGen.Allocate(),
		gateStatus: ie.NewGateStatus(ie.GateStatusOpen, ie.GateStatusOpen),
		state:      RULE_INIT,
	}
	s.Tracef("QER id %d is allocate", newQer.id)

	ul := BitRateToKbps(b.qosData.MaxbrUl)
	dl := BitRateToKbps(b.qosData.MaxbrDl)
	newQer.mbr = ie.NewMBR(uint32(ul), uint32(dl))
	if isGBRFlow(b.qosData) {
		ul = BitRateToKbps(b.qosData.GbrUl)
		dl = BitRateToKbps(b.qosData.GbrDl)
		newQer.gbr = ie.NewGBR(uint32(ul), uint32(dl))

	}
	qerList = append(qerList, newQer)
	return
}

type defaultQosQERBuilder struct {
	sessionRule *models.SessionRule
}

func (b *defaultQosQERBuilder) buildQERs(qfi uint8, s *PfcpSession) (qerList []*QER) {
	newQer := &QER{
		qfi:        qfi,
		id:         s.upf.qerIdGen.Allocate(),
		gateStatus: ie.NewGateStatus(ie.GateStatusOpen, ie.GateStatusOpen),
		state:      RULE_INIT,
	}
	s.Tracef("QER id %d is allocate", newQer.id)
	qerList = append(qerList, newQer)
	return
}

func NewQosQERBuilder(dat *models.QosData) QERBuilder {
	return &qosQERBuilder{
		qosData: dat,
	}
}

func NewDefaultQosQERBuilder(rule *models.SessionRule) QERBuilder {
	return &defaultQosQERBuilder{
		sessionRule: rule,
	}
}

type QER struct {
	id         uint32 //generate per UPF
	qfi        uint8  // QoSflow identifier
	gateStatus *ie.GateStatus
	mbr        *ie.MBR // MaximumBitrate
	gbr        *ie.GBR // GuaranteedBitrate
	state      RuleState
}

func (qer *QER) toCreateQer() ie.CreateQER {
	cQer := ie.CreateQER{
		QERID:             qer.id,
		QoSflowidentifier: &qer.qfi,
		GateStatus:        qer.gateStatus,
		MaximumBitrate:    qer.mbr,
		GuaranteedBitrate: qer.gbr,
	}
	return cQer
}

func (qer *QER) toUpdateQer() ie.UpdateQER {
	uQer := ie.UpdateQER{
		QERID:             qer.id,
		QoSflowidentifier: &qer.qfi,
		GateStatus:        qer.gateStatus,
		MaximumBitrate:    qer.mbr,
		GuaranteedBitrate: qer.gbr,
	}
	return uQer
}

//mapping a QFI (QosFlow) to a list of QERs
type QosFlow struct {
	qfi           uint8
	qerList       []*QER
	updateQerList []*QER //new qer list in case of updating
	state         RuleState
}

func (f *QosFlow) toCreateQERs() (qerList []ie.CreateQER) {
	list := f.qerList
	if f.state == RULE_UPDATE {
		list = f.updateQerList
	}
	for _, qer := range list {
		qerList = append(qerList, qer.toCreateQer())
	}
	return
}
