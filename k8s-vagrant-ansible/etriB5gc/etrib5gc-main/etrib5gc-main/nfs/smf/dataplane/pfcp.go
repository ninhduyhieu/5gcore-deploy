package dataplane

import (
	"etrib5gc/mesh"
	"etrib5gc/nfs/smf/context"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/pfcp/message"
	"github.com/reogac/sbi/apis/upf/n4sbi"
	"github.com/reogac/utils"
)

const (
	MesureMethodVol  = "vol"
	MesureMethodTime = "time"
)

// PFCPSMReqFlags
//
//	Example, if DROBU and SNDEM are enabled, flag is sum of:
//
// PFCPSMReqFlags_DROBU + PFCPSMReqFlags_SNDEM
const (
	PFCPSMReqFlags_DROBU uint8 = 1
	PFCPSMReqFlags_SNDEM uint8 = 1 << 1
	PFCPSMReqFlags_QAURR uint8 = 1 << 2
)

// PDNType definitions.
const (
	_               uint8 = 0
	PDNTypeIPv4     uint8 = 1
	PDNTypeIPv6     uint8 = 2
	PDNTypeIPv4v6   uint8 = 3
	PDNTypeNonIP    uint8 = 4
	PDNTypeEthernet uint8 = 5
)

// Interface definitions.
const (
	SrcInterfaceAccess       uint8 = 0
	SrcInterfaceCore         uint8 = 1
	SrcInterfaceSGiLANN6LAN  uint8 = 2
	SrcInterfaceCPFunction   uint8 = 3
	SrcInterface5GVNInternal uint8 = 4
)

// Interface definitions.
const (
	DstInterfaceAccess       uint8 = 0
	DstInterfaceCore         uint8 = 1
	DstInterfaceSGiLANN6LAN  uint8 = 2
	DstInterfaceCPFunction   uint8 = 3
	DstInterfaceLIFunction   uint8 = 4
	DstInterface5GVNInternal uint8 = 5
)

func (s *PfcpSession) sendDeletionRequest() (rsp *message.PFCPSessionDeletionResponse, err error) {
	if s.remoteSeid == nil {
		//NOTE: session has not been established at UPF
		return //nothing to do
	}
	msg := &message.PFCPSessionDeletionRequest{}

	if rsp, err = n4sbi.SessionDeletion(s.upf.cli, int64(*s.remoteSeid), msg); err != nil {
		err = utils.WrapError("Send PfcpSessionDeletionRequest to UPF", err)
	} else {
		s.Infof("PfcpSessionDeletionRequest was sent to UPF")
	}
	return
}

func (s *PfcpSession) sendEstablishmentRequest() (rsp *message.PFCPSessionEstablishmentResponse, err error) {
	s.Debugf("Send PfcpSessionEstablishmentRequest")
	nodeIP := context.GetPfcpNodeId().IP()
	msg := &message.PFCPSessionEstablishmentRequest{
		NodeID:  *context.GetPfcpNodeId(),
		CPFSEID: *ie.NewFSEID(&nodeIP, nil, s.localSeid),
	}

	for _, flow := range s.qosFlows {
		if flow.state == RULE_INIT {
			s.Tracef("create QER for qfi=%d\n%", flow.qfi)
			msg.CreateQER = append(msg.CreateQER, flow.toCreateQERs()...)
		}
	}

	for _, link := range s.links {
		for _, lp := range link.linkPaths {
			if lp.far.state == RULE_INIT {
				s.Tracef("create FAR %d", lp.far.id)
				msg.CreateFAR = append(msg.CreateFAR, lp.far.toCreateFar())
				for _, pdr := range lp.pdrList {
					//NOTE: no need to check PDR state, they all must be in
					//RULE_INIT state
					s.Tracef("create PDR %d", pdr.id)
					msg.CreatePDR = append(msg.CreatePDR, pdr.toCreatePdr(s))
				}
			}
		}
	}

	//msg.CreateBAR = bar.toCreateBar()
	//msg.CreateURR = append(msg.CreateURR, urr.toCreateUrr())

	msg.PDNType = new(uint8)
	*msg.PDNType = PDNTypeIPv4

	callback := mesh.EndpointInfo()
	if rsp, err = n4sbi.SessionEstablishment(s.upf.cli, callback.Uuid, msg); err != nil {
		err = utils.WrapError("Send PfcpSessionEstablishment", err)
		//TODO: handle error
	} else {
		s.Infof("PcfSessionEstablishment was sent to UPF")
		//set rules to RULE_CREATED state
		for _, flow := range s.qosFlows {
			if flow.state == RULE_INIT {
				flow.state = RULE_CREATED
			}
		}

		for _, link := range s.links {
			for _, lp := range link.linkPaths {
				if lp.far.state == RULE_INIT {
					lp.far.state = RULE_CREATED
					for _, pdr := range lp.pdrList {
						pdr.state = RULE_CREATED
					}
				}
			}
		}

	}
	return
}

func (s *PfcpSession) sendModificationRequest() (rsp *message.PFCPSessionModificationResponse, err error) {
	s.Debugf("Send PfcpSessionModificationRequest")
	nodeIP := context.GetPfcpNodeId().IP()
	msg := &message.PFCPSessionModificationRequest{
		NodeID:  context.GetPfcpNodeId(),
		CPFSEID: ie.NewFSEID(&nodeIP, nil, s.localSeid),
	}
	for _, flow := range s.qosFlows {
		switch flow.state {
		case RULE_INIT:
			s.Tracef("create QERs for qfi %d", flow.qfi)
			msg.CreateQER = append(msg.CreateQER, flow.toCreateQERs()...)
		case RULE_UPDATE:
			//create new QERs
			s.Tracef("create QERs for qfi %d", flow.qfi)
			msg.CreateQER = append(msg.CreateQER, flow.toCreateQERs()...)
			//delete old QERs
			s.Tracef("delete QERs for qfi %d", flow.qfi)
			for _, qer := range flow.qerList {
				msg.RemoveQER = append(msg.RemoveQER, ie.RemoveQER{QERID: qer.id})
			}

		case RULE_DELETE:
			s.Tracef("delete QERs for qfi %d", flow.qfi)
			for _, qer := range flow.qerList {
				msg.RemoveQER = append(msg.RemoveQER, ie.RemoveQER{QERID: qer.id})
			}
		}
	}
	for _, link := range s.links {
		for _, lp := range link.linkPaths {
			if lp.far.state != RULE_CREATED {
				switch lp.far.state {
				case RULE_INIT:
					s.Tracef("create FAR %d", lp.far.id)
					msg.CreateFAR = append(msg.CreateFAR, lp.far.toCreateFar())
				case RULE_UPDATE:
					s.Tracef("update FAR %d", lp.far.id)
					msg.UpdateFAR = append(msg.UpdateFAR, lp.far.toUpdateFar())
				case RULE_DELETE: //not handle [for now we do not expect that FAR is deleted as we do not change an existing path]
					s.Tracef("delete FAR %d[not handle]", lp.far.id)
					//msg.RemoveFAR = append(msg.RemoveFAR, ie.RemoveFAR{FARID: lp.far.id})
				}
			}
			for _, pdr := range lp.pdrList {
				switch pdr.state {
				case RULE_INIT:
					s.Tracef("create PDR %d", pdr.id)
					msg.CreatePDR = append(msg.CreatePDR, pdr.toCreatePdr(s))
				case RULE_UPDATE:
					s.Tracef("update PDR %d", pdr.id)
					msg.UpdatePDR = append(msg.UpdatePDR, pdr.toUpdatePdr(s))
				case RULE_DELETE:
					s.Tracef("delete PDR %d", pdr.id)
					msg.RemovePDR = append(msg.RemovePDR, ie.RemovePDR{PDRID: pdr.id})
				}
			}

		}
	}

	if rsp, err = n4sbi.SessionModification(s.upf.cli, int64(*s.remoteSeid), msg); err != nil {
		err = utils.WrapError("Send PfcpSessionModification", err)
		//TODO: handle error
	} else {
		s.Infof("PfcpSessionModificatioRequest was sent to UPF")
		//update rule states
		for _, flow := range s.qosFlows {
			switch flow.state {
			case RULE_INIT:
				flow.state = RULE_CREATED
			case RULE_UPDATE:
				flow.state = RULE_CREATED
				flow.qerList = flow.updateQerList
				flow.updateQerList = []*QER{}
			case RULE_DELETE:
				delete(s.qosFlows, flow.qfi) //remove flow
			}
		}

		for _, link := range s.links {
			for _, lp := range link.linkPaths {
				lp.far.state = RULE_CREATED //NOTE: no FAR deletion
				//remove deleted PDRs and set remaning ones to RULE_CREATED
				//state
				pdrList := []*PDR{}
				for _, pdr := range lp.pdrList {
					if pdr.state != RULE_DELETE {
						pdr.state = RULE_CREATED
						pdrList = append(pdrList, pdr)
					}
				}
				lp.pdrList = pdrList

			}
		}

	}
	return
}
