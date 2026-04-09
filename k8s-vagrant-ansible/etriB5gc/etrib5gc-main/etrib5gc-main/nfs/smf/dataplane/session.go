package dataplane

import (
	"encoding/json"
	"fmt"
	"github.com/reogac/pfcp/message"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"net"
)

type SmContext interface {
	Dnn() string
	UeIp() net.IP
	WithFields(logrus.Fields) *logrus.Entry
}

//a Pfcp session toward a UPF
type PfcpSession struct {
	*logrus.Entry
	ctx        SmContext
	upf        *Upf
	localSeid  uint64             // session id allocated by SMF
	remoteSeid *uint64            //session id allocated by UPF
	links      map[string]*Link   //all links to other neighboring UPFs in the datapaths
	qosFlows   map[uint8]*QosFlow //map QFI (of a Qos flow) to a list of QERs
}

func (s *PfcpSession) GetUpf() *Upf {
	return s.upf
}

func (s *PfcpSession) GetUpfId() string {
	return s.upf.Id()
}

//get or create a link to a next/prev UPF in a datapath
func (s *PfcpSession) GetLink(direction uint8, localIp, remoteIp net.IP) *Link {
	//create link ID
	linkId := createLinkId(localIp, remoteIp, direction)
	//get Link if it existed
	if l, ok := s.links[linkId]; ok {
		return l
	}
	//otherwise, create a new link

	l := &Link{
		id:        linkId,
		session:   s,
		direction: direction,
		localIp:   localIp,
		remoteIp:  remoteIp,
		linkPaths: make(map[uint8]*LinkPath),
	}

	if len(remoteIp) > 0 || direction == DIRECTION_UPLINK { //do not create teid for downlink at anchor UPF
		l.teid = s.upf.teidGen.Allocate()
	}
	s.Tracef("Create link %s with TEID=%d", linkId, l.teid)
	//cache it
	s.links[linkId] = l
	return l
}

// ask UPFs to release session
func (s *PfcpSession) Release() (err error) {
	s.Debug("Release session")
	//remove QERs
	for qfi, flow := range s.qosFlows {
		s.Tracef("Free QER Ids for qfi %d", qfi)
		for _, qer := range flow.qerList {
			s.Tracef("Free QER Id %d of %d", qer.id, qfi)
			s.upf.qerIdGen.Free(qer.id)
		}
	}

	var rsp *message.PFCPSessionDeletionResponse
	rsp, err = s.sendDeletionRequest()
	if err != nil {
		err = utils.WrapError("Build then send PfcpSessionDeleteSession", err)
		return
	}

	if rsp != nil && rsp.PFCPSDRspFlags != nil {
		var seid uint64
		if err = json.Unmarshal(rsp.PFCPSDRspFlags, &seid); err != nil {
			err = utils.WrapError("Parse SEID from SessionDeletionResponse", err)
		}
	}
	s.remoteSeid = nil
	s.upf.deleteSession(s)
	_pool.remove(s)
	return
}

func (s *PfcpSession) Update() (err error) {
	s.Debugf("Update pfcp session %d", s.localSeid)
	if s.remoteSeid == nil { //session has not been created at UPF
		var rsp *message.PFCPSessionEstablishmentResponse
		if rsp, err = s.sendEstablishmentRequest(); err != nil {
			err = utils.WrapError("Build then send PfcpSessionEstablishment", err)
		} else if rsp.UPFSEID != nil {
			seid := rsp.UPFSEID.Seid()
			s.remoteSeid = &seid
		} else {
			err = fmt.Errorf("Receive PfcpSessionEstablishment response without SEID")
		}
	} else {
		if _, err = s.sendModificationRequest(); err != nil {
			err = utils.WrapError("Build then send PfcpSessionModification", err)
		}
	}
	return
}

func (s *PfcpSession) DeleteQosFlow(qfi uint8) {
	if flow, ok := s.qosFlows[qfi]; ok {
		flow.state = RULE_DELETE

		//release QER IDs
		for _, qer := range flow.qerList {
			s.upf.qerIdGen.Free(qer.id)
		}
	}
	//NOTE: don't worry about PDRs, they should be deleted in advance.
}

func (s *PfcpSession) UpdateQosFlow(qfi uint8, builder QERBuilder) (err error) {
	s.Debugf("Update Qos flow [qfi=%d]", qfi)
	newQerList := builder.buildQERs(qfi, s)

	if flow, ok := s.qosFlows[qfi]; ok {
		//NOTE: when updating a Qos flow, we delete old associated QERs and
		//create new ones to reflect updates

		//release QER IDs of the old QER list
		for _, qer := range flow.qerList {
			s.upf.qerIdGen.Free(qer.id)
		}

		flow.state = RULE_UPDATE
		flow.updateQerList = newQerList //link to new QERs
	} else { //create new flow
		s.qosFlows[qfi] = &QosFlow{
			qfi:     qfi,
			qerList: newQerList,
			state:   RULE_INIT,
		}
	}
	return
}
