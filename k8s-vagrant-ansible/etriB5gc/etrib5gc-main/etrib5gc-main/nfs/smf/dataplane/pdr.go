package dataplane

import (
	"github.com/reogac/pfcp/ie"
)

type PDR struct {
	id                 uint16
	precedence         uint32
	pdi                *ie.PDI
	outerHeaderRemoval *ie.OuterHeaderRemoval
	far                *FAR
	urrList            []*URR
	qfi                uint8
	state              RuleState
}

func (pdr *PDR) toCreatePdr(s *PfcpSession) ie.CreatePDR {
	cPdr := ie.CreatePDR{
		PDRID:              pdr.id,
		Precedence:         pdr.precedence,
		OuterHeaderRemoval: pdr.outerHeaderRemoval,
	}
	if pdr.far != nil {
		cPdr.FARID = &pdr.far.id
	}
	if pdr.pdi != nil {
		cPdr.PDI = pdr.pdi
	}

	for _, urr := range pdr.urrList {
		if urr != nil {
			cPdr.URRID = append(cPdr.URRID, urr.id)
		}
	}
	if flow, ok := s.qosFlows[pdr.qfi]; ok {
		for _, qer := range flow.qerList {
			//TODO: CreatePDR should have a list of QerIDs, not a single one
			cPdr.QERID = &qer.id
			break
		}
	}

	return cPdr
}

func (pdr *PDR) toUpdatePdr(s *PfcpSession) ie.UpdatePDR {
	uPdr := ie.UpdatePDR{
		PDRID:              pdr.id,
		Precedence:         &pdr.precedence,
		PDI:                pdr.pdi,
		OuterHeaderRemoval: pdr.outerHeaderRemoval,
	}
	if pdr.far != nil {
		uPdr.FARID = &pdr.far.id
	}
	for _, urr := range pdr.urrList {
		if urr != nil {
			uPdr.URRID = &urr.id
			break
		}
	}
	if flow, ok := s.qosFlows[pdr.qfi]; ok {
		qerList := flow.qerList

		if flow.state == RULE_UPDATE {
			qerList = flow.updateQerList
		}

		for _, qer := range qerList {
			uPdr.QERID = &qer.id
			break
		}
	}

	return uPdr
}
