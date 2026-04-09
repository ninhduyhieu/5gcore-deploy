package tunnel

import (
	"etrib5gc/nfs/smf/dataplane"
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
)

const (
	DefaultNonGBR5QI int64 = 9
)

type QosConverter interface {
	toNasQosFlowDesc(uint8) QosFlowDesc
	toNgapQosFlowSetupRequestItem(uint8) ies.QosFlowSetupRequestItem
}

type defaultQosConverter struct {
	rule *models.SessionRule
}

func newDefaultQosConverter(r *models.SessionRule) *defaultQosConverter {
	return &defaultQosConverter{
		rule: r,
	}
}

func (c *defaultQosConverter) getFiveQi() int64 {
	fiveQi := DefaultNonGBR5QI
	if c.rule != nil && c.rule.AuthDefQos != nil && c.rule.AuthDefQos.FiveQi != nil {
		fiveQi = int64(*c.rule.AuthDefQos.FiveQi)
	}
	return fiveQi
}

func (c *defaultQosConverter) toNasQosFlowDesc(qfi uint8) (desc QosFlowDesc) {
	desc = QosFlowDesc{
		QFI:           qfi,
		OperationCode: OperationCodeCreateNewQosFlowDescription,
	}
	desc.Parameters = append(desc.Parameters, &QosFlow5QI{
		FiveQI: uint8(c.getFiveQi()),
	})

	return
}

func (c *defaultQosConverter) toNgapQosFlowSetupRequestItem(qfi uint8) (item ies.QosFlowSetupRequestItem) {
	var arp *models.Arp
	if c.rule != nil && c.rule.AuthDefQos != nil {
		arp = c.rule.AuthDefQos.Arp
	}
	item = ies.QosFlowSetupRequestItem{
		QosFlowIdentifier: int64(qfi),
		QosFlowLevelQosParameters: ies.QosFlowLevelQosParameters{
			QosCharacteristics: ies.QosCharacteristics{
				Choice: ies.QosCharacteristicsPresentNondynamic5Qi,
				NonDynamic5QI: &ies.NonDynamic5QIDescriptor{
					FiveQI: c.getFiveQi(),
				},
			},
			AllocationAndRetentionPriority: buildNgapArp(arp),
		},
	}

	return
}

type qosConverter struct {
	qos *models.QosData
}

func newQosConverter(dat *models.QosData) *qosConverter {
	return &qosConverter{
		qos: dat,
	}
}

func (c *qosConverter) getFiveQi() int64 {
	fiveQi := DefaultNonGBR5QI
	if c.qos.FiveQi != nil {
		fiveQi = int64(*c.qos.FiveQi)
	}
	return fiveQi
}

func (c *qosConverter) toNasQosFlowDesc(qfi uint8) (desc QosFlowDesc) {
	//TODO:
	desc = QosFlowDesc{
		QFI:           qfi,
		OperationCode: OperationCodeCreateNewQosFlowDescription,
	}
	desc.Parameters = append(desc.Parameters, &QosFlow5QI{
		FiveQI: uint8(c.getFiveQi()),
	})

	/* TODO: set GBR parameters
	if q.IsGBRFlow() && q.QoSProfile != nil {
		gbrDlParameter := new(nasType.QoSFlowGFBRDownlink)
		gbrDlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		gbrDlParameter.Value = util.BitRateTombps(q.QoSProfile.GbrDl)
		qosDesc.Parameters = append(qosDesc.Parameters, gbrDlParameter)
		gbrUlParameter := new(nasType.QoSFlowGFBRUplink)
		gbrUlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		gbrUlParameter.Value = util.BitRateTombps(q.QoSProfile.GbrUl)
		qosDesc.Parameters = append(qosDesc.Parameters, gbrUlParameter)
		mbrDlParameter := new(nasType.QoSFlowMFBRDownlink)
		mbrDlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		mbrDlParameter.Value = util.BitRateTombps(q.QoSProfile.MaxbrDl)
		qosDesc.Parameters = append(qosDesc.Parameters, mbrDlParameter)
		mbrUlParameter := new(nasType.QoSFlowMFBRUplink)
		mbrUlParameter.Unit = nasType.QoSFlowBitRateUnit1Mbps
		mbrUlParameter.Value = util.BitRateTombps(q.QoSProfile.MaxbrUl)
		qosDesc.Parameters = append(qosDesc.Parameters, mbrUlParameter)
	}
	*/
	return
}

func (c *qosConverter) toNgapQosFlowSetupRequestItem(qfi uint8) (item ies.QosFlowSetupRequestItem) {

	item = ies.QosFlowSetupRequestItem{
		QosFlowIdentifier: int64(qfi),
		QosFlowLevelQosParameters: ies.QosFlowLevelQosParameters{
			QosCharacteristics: ies.QosCharacteristics{
				Choice: ies.QosCharacteristicsPresentNondynamic5Qi,
				NonDynamic5QI: &ies.NonDynamic5QIDescriptor{
					FiveQI: c.getFiveQi(),
				},
			},
			AllocationAndRetentionPriority: buildNgapArp(c.qos.Arp),
		},
	}
	return
}

type QosFlow struct {
	qfi        uint8
	qerBuilder dataplane.QERBuilder //building QERs from qos flow
	converter  QosConverter         //build Qos Rule/Description for NAS/NGAP
	paths      map[uint8]*Path      //data paths linked to this Qos Flow
}

func newDefaultQosFlow(sessionRule *models.SessionRule) *QosFlow {
	return &QosFlow{
		qfi:        1,
		qerBuilder: dataplane.NewDefaultQosQERBuilder(sessionRule),
		converter:  newDefaultQosConverter(sessionRule),
		paths:      make(map[uint8]*Path),
	}
}

func newQosFlow(qfi uint8, dat *models.QosData) *QosFlow {
	return &QosFlow{
		qfi:        qfi,
		qerBuilder: dataplane.NewQosQERBuilder(dat),
		converter:  newQosConverter(dat),
		paths:      make(map[uint8]*Path),
	}
}

func (t *Tunnel) deleteQosFlow(id string) (err error) {
	if flow, ok := t.qosFlows[id]; ok {
		//if there is no path link to this session we can safely delete the Qos
		//Flow
		if len(flow.paths) == 0 {
			for _, session := range t.sessions {
				session.DeleteQosFlow(flow.qfi)
			}
			delete(t.qosFlows, id)
			t.qfiGen.Free(flow.qfi)
		} else {
			err = fmt.Errorf("Can't delete Qos Flow; there is an attached Pcc rule")
		}
	}
	return
}

func (t *Tunnel) updateQosFlow(dat *models.QosData) {
	var ok bool
	var flow *QosFlow
	if flow, ok = t.qosFlows[dat.QosId]; ok { //existing flow, need to update
		flow.qerBuilder = dataplane.NewQosQERBuilder(dat)
		for _, path := range flow.paths {
			path.updateQosFlow()
		}
	} else { //create a new QosFlow
		qfi := t.qfiGen.Allocate()
		flow = newQosFlow(qfi, dat)
		t.qosFlows[dat.QosId] = flow
	}
}

//pick a QosFlow for a given PccRule
func (t *Tunnel) pickQosFlow(rule *models.PccRule) (flow *QosFlow) {
	var ok bool
	for _, id := range rule.RefQosData {
		//NOTE: for now, just take the first one
		if flow, ok = t.qosFlows[id]; ok {
			return
		}
	}
	return
}
