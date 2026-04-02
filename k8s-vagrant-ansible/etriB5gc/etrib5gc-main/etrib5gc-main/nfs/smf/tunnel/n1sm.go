package tunnel

import (
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
)

func (t *Tunnel) FillN1SmPduSessionEstablishmentAccept(msg *nas.PduSessionEstablishmentAccept) {
	var qosRules QosRules
	var qosDescs QosFlowDescs
	for _, p := range t.getAllPaths() {
		/*
			if !t.isDefaultPath(p) {
				continue
			}
		*/
		qosRules = append(qosRules, p.buildQosRule(t.isDefaultPath(p)))
		qosDescs = append(qosDescs, p.buildQosFlowDesc())
	}
	qosRulesBytes, _ := qosRules.MarshalBinary()
	qosFlowDescsBytes, _ := qosDescs.MarshalBinary()

	msg.AuthorizedQosRules = nas.QosRules{
		Bytes: qosRulesBytes,
	}

	msg.AuthorizedQosFlowDescriptions = &nas.QosFlowDescriptions{
		Bytes: qosFlowDescsBytes,
	}

	var ambr *models.Ambr
	if sr, ok := t.sessionRules[t.selectedSessionRule]; ok {
		if ambr = sr.AuthSessAmbr; ambr != nil {
			msg.SessionAmbr = *ambr.NasSessionAmbr()
		}
	}

	if ambr == nil {
		msg.SessionAmbr = *nas.NewSessionAmbr(nas.SessionAMBRUnit1Mbps, 100, nas.SessionAMBRUnit1Mbps, 100)
	}

}
