package dataplane

import (
	"fmt"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/sbi/models"
	"net"
)

const (
	DIRECTION_UPLINK uint8 = iota
	DIRECTION_DOWNLINK
)

type GtpFace interface {
	GetIp() net.IP
	GetTeid() uint32
}

//how link content in a given data path
type LinkPath struct {
	qfi     uint8 //qos flow id
	pdrList []*PDR
	far     *FAR
}

type Link struct {
	id        string              //id composed by direction and remote face IP
	direction uint8               //uplink or downlink
	remoteIp  net.IP              //remote face IP
	localIp   net.IP              //loca face IP
	teid      uint32              //local TEID
	session   *PfcpSession        //session owns this link
	linkPaths map[uint8]*LinkPath //link paths on different data paths
}

func (l *Link) GetTeid() uint32 {
	return l.teid
}

func (l *Link) freeTeid() {
	//only free allocated TEID
	if l.direction == DIRECTION_UPLINK || len(l.remoteIp) > 0 {
		l.session.upf.teidGen.Free(l.teid)
	}
}

func (l *Link) GetIp() net.IP {
	return l.localIp
}

//NOTE: panic if link is not activated (PDRs have not been allocated)
func (l *Link) UpdateFAR(pathId uint8, ip net.IP, teid uint32) (err error) {
	if lp, ok := l.linkPaths[pathId]; !ok {
		err = fmt.Errorf("Path %d not found", pathId)
		return
	} else {
		far := lp.far //should be non-nil
		l.session.Infof("Update FAR (id=%d) for link %s", far.id, l.id)
		if far.fwdParams.OuterHeaderCreation != nil {
			far.fwdParams.SendEndMarker = true
		}

		//Edit FAR to tell UPF to create GTP header to forward packets to gnB (ip and teid
		//from gnB are used to set the FAR's parameters
		far.action = ie.NewApplyAction(ie.APPLY_ACT_FORW)
		far.fwdParams.OuterHeaderCreation = ie.NewOuterHeaderCreationGtpIpv4(ip, teid)

		//mark the rule to be updated
		far.state = RULE_UPDATE
	}
	return
}

//NOTE: panic if link is not activated (PDRs have not been allocated)
func (l *Link) DeactivateFAR(pathId uint8) (err error) {
	if lp, ok := l.linkPaths[pathId]; !ok {
		err = fmt.Errorf("Path %d not found", pathId)
		return
	} else {
		far := lp.far //should be non-nil

		far.action = ie.NewApplyAction(ie.APPLY_ACT_BUFF | ie.APPLY_ACT_NOCP)
		//mark the rule to be updated
		far.state = RULE_UPDATE
	}
	return
}

func (l *Link) CreatePDRs(pathId uint8, precedence uint32, qfi uint8, flowInfos []models.FlowInformation, fwdFace GtpFace) (err error) {
	l.session.Debugf("Create PDRs for link : %s", l.id)
	upf := l.session.upf
	dnn := l.session.ctx.Dnn()
	ueIp := l.session.ctx.UeIp()
	farId := l.session.upf.farIdGen.Allocate()
	pdrList := []*PDR{}

	//1. create FAR
	var far *FAR
	if l.direction == DIRECTION_UPLINK { //it is an uplink
		far = &FAR{
			id:     farId,
			action: ie.NewApplyAction(ie.APPLY_ACT_FORW),
			fwdParams: &ForwardingParameters{
				NetworkInstance: dnn,
			},
		}
		if fwdFace == nil { //its next link is connecting to a data network
			far.fwdParams.DestinationInterface = DstInterfaceSGiLANN6LAN //Data network
		} else { //its next link is connect to an UPF
			far.fwdParams.DestinationInterface = DstInterfaceCore //next destination is UPF
			//TODO: set Forwarding Parameter to the next UPF
		}

		l.session.Debugf("Create uplink FAR (id=%d)", far.id)
	} else { //it is a downlink
		far = &FAR{
			id:     farId,
			action: ie.NewApplyAction(ie.APPLY_ACT_FORW),
			fwdParams: &ForwardingParameters{
				DestinationInterface: DstInterfaceAccess, //TODO: how about the next hop is I-UPF
				NetworkInstance:      dnn,                //NOTE: only at UPF facing RAN?
			},
		}
		l.session.Debugf("Create downlink FAR (id=%d)", far.id)
	}
	far.state = RULE_INIT
	//2. create PDRs and attach the created FAR
	if l.direction == DIRECTION_UPLINK { //it is an uplink
		srcFace := SrcInterfaceAccess //previous node is gnB

		if len(l.remoteIp) > 0 {
			srcFace = SrcInterfaceCore //previous node is another UPF
		}

		tmpPdi := ie.PDI{
			LocalFTEID:      ie.NewFTEID(&l.localIp, nil, l.teid),
			SourceInterface: srcFace,
			NetworkInstance: []byte(dnn),
			UEIPaddress:     ie.NewUEIPAddress(&ueIp, nil),
		}

		tmpPdr := PDR{
			precedence:         precedence,
			far:                far,
			pdi:                &tmpPdi,
			outerHeaderRemoval: ie.NewOuterHeaderRemoval(ie.OuterHeaderRemovalGtpUUdpIpv4), //always for uplink
			qfi:                qfi,                                                        //link to qer list
			//urr list
			state: RULE_INIT,
		}

		if len(flowInfos) == 0 {
			tmpPdr.id = upf.pdrIdGen.Allocate()
			l.session.Debugf("Create uplink PDR (id=%d)", tmpPdr.id)
			pdrList = append(pdrList, &tmpPdr)
		} else {
			for _, flowInfo := range flowInfos {
				if filters := flowInfo2SDFFilters(l.direction, &flowInfo); len(filters) > 0 {
					pdi := tmpPdi                    //copy PDI
					pdi.SDFFilter = filters          //set filters
					pdr := tmpPdr                    //copy PDR
					pdr.pdi = &pdi                   //set new PDI
					pdr.id = upf.pdrIdGen.Allocate() //allocate PDR id

					l.session.Debugf("Create uplink PDR (id=%d)", pdr.id)
					pdrList = append(pdrList, &pdr)
				}
			}
		}

	} else { //it is an downlink
		srcFace := SrcInterfaceCore
		var outerHeaderRemoval *ie.OuterHeaderRemoval
		if len(l.remoteIp) == 0 { //downlink at anchoring UPF
			srcFace = SrcInterfaceSGiLANN6LAN
		} else { //need to remove gtp header
			outerHeaderRemoval = ie.NewOuterHeaderRemoval(ie.OuterHeaderRemovalGtpUUdpIpv4)
		}
		tmpPdi := ie.PDI{
			SourceInterface: srcFace,
			//			LocalFTEID:      ie.NewFTEID(&l.localIp, nil, l.teid),
			NetworkInstance: []byte(dnn),
			UEIPaddress:     ie.NewUEIPAddress(&ueIp, nil),
		}
		if len(l.localIp) > 0 {
			tmpPdi.LocalFTEID = ie.NewFTEID(&l.localIp, nil, l.teid)
		}

		tmpPdr := PDR{
			precedence:         precedence,
			far:                far,
			pdi:                &tmpPdi,
			outerHeaderRemoval: outerHeaderRemoval,
			qfi:                qfi, //link to qer list
			//urr list
			state: RULE_INIT,
		}

		if len(flowInfos) == 0 {
			tmpPdr.id = upf.pdrIdGen.Allocate()
			l.session.Debugf("Create downlink PDR (id=%d)", tmpPdr.id)
			pdrList = append(pdrList, &tmpPdr)
		} else {
			for _, flowInfo := range flowInfos {
				if filters := flowInfo2SDFFilters(l.direction, &flowInfo); len(filters) > 0 {
					pdi := tmpPdi                    //copy PDI
					pdi.SDFFilter = filters          //set filters
					pdr := tmpPdr                    //copy PDR
					pdr.pdi = &pdi                   //set new PDI
					pdr.id = upf.pdrIdGen.Allocate() //allocate PDR id
					l.session.Debugf("Create downlink PDR (id=%d)", pdr.id)
					pdrList = append(pdrList, &pdr)
				}
			}
		}

	}
	l.linkPaths[pathId] = &LinkPath{
		far:     far,
		qfi:     qfi,
		pdrList: pdrList,
	}
	return
}

func (l *Link) DeletePath(pathId uint8) {
	l.session.Infof("Delete path %d on link %s", pathId, l.id)
	upf := l.session.upf
	//remote PDRs and FAR; free TEID if applicable
	if lp, ok := l.linkPaths[pathId]; ok {
		delete(l.linkPaths, pathId)
		if len(l.linkPaths) == 0 { //no more path on this link
			delete(l.session.links, l.id)
			//free TEID
			l.freeTeid()
		}

		//mark FAR and PDRs to be deleted
		lp.far.state = RULE_DELETE
		//log.Infof("Free FAR id: %d", lp.far.id)
		upf.farIdGen.Free(lp.far.id)
		for _, pdr := range lp.pdrList {
			//log.Infof("Free PDR id: %d", pdr.id)
			upf.pdrIdGen.Free(pdr.id)
			pdr.state = RULE_DELETE
		}
	}
}
