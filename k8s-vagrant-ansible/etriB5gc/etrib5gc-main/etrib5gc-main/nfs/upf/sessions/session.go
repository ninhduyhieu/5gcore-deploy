package sessions

import (
	"encoding/json"
	"etrib5gc/nfs/upf/report"
	"fmt"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/sbi"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"net"
	"sync"

	"github.com/pkg/errors"
)

const (
	BUFFQ_LEN = 512
)

type PDRInfo struct {
	RelatedurrIdList map[uint32]struct{}
}

type URRInfo struct {
	removed bool
	SEQN    uint32
	ie.MeasureMethod
	ie.MeasurementInformation
	refPdrNum uint16
}

type PfcpSession struct {
	*logrus.Entry
	smfCli sbi.ConsumerClient

	localID  uint64 //local SEID
	remoteID uint64 //remote SEID

	pdrList   map[uint16]*PDRInfo    // key: PDR_ID
	farIdList map[uint32]struct{}    // key: FAR_ID
	qerIdList map[uint32]struct{}    // key: QER_ID
	urrIdList map[uint32]*URRInfo    // key: URR_ID
	barIdList map[uint8]struct{}     // key: BAR_ID
	q         map[uint16]chan []byte // key: PDR_ID
	qlen      int
	mutex     sync.Mutex
}

func newPfcpSession(lSeid, rSeid uint64, smfCli sbi.ConsumerClient) (s *PfcpSession) {
	s = &PfcpSession{
		localID:   lSeid,
		remoteID:  rSeid,
		smfCli:    smfCli,
		pdrList:   make(map[uint16]*PDRInfo),
		farIdList: make(map[uint32]struct{}),
		qerIdList: make(map[uint32]struct{}),
		urrIdList: make(map[uint32]*URRInfo),
		barIdList: make(map[uint8]struct{}),
		q:         make(map[uint16]chan []byte),
		Entry: log.WithFields(logrus.Fields{
			"lSeid": fmt.Sprintf("%d", lSeid),
		}),
	}

	return
}

func (s *PfcpSession) RemoteSeid() uint64 {
	return s.remoteID
}

func (s *PfcpSession) fSEID() *ie.FSEID {
	var v4 net.IP
	var v4_str string
	//Note: since nodeId is valid, there is no error
	_ = json.Unmarshal(_sessMan.nodeId.Bytes(), &v4_str) //IP address of N4 (SBI)
	addrv4, _ := net.ResolveIPAddr("ip4", v4_str)
	v4 = addrv4.IP

	return ie.NewFSEID(&v4, nil, s.localID)
}

func (s *PfcpSession) close() []report.USAReport {

	var usars []report.USAReport

	//remove PDRs
	for id := range s.pdrList {
		if rs, err := s.removePDR(&ie.RemovePDR{
			PDRID: id,
		}); err != nil {
			s.Errorf("remove PDR err: %+v", err)
		} else if rs != nil {
			usars = append(usars, rs...)
		}
	}
	//remove URRs
	for id := range s.urrIdList {
		if rs, err := s.removeURR(&ie.RemoveURR{
			URRID: id,
		}); err != nil {
			s.Errorf("Remove URR err: %+v", err)
			continue
		} else if rs != nil {
			usars = append(usars, rs...)
		}
	}

	//remove QERs
	for id := range s.qerIdList {
		if err := s.removeQER(&ie.RemoveQER{
			QERID: id,
		}); err != nil {
			s.Errorf("Remove QER err: %+v", err)
		}
	}

	//remove FARs
	for id := range s.farIdList {
		if err := s.removeFAR(&ie.RemoveFAR{
			FARID: id,
		}); err != nil {
			s.Errorf("Remove FAR err: %+v", err)
		}
	}

	//remove BARs
	for id := range s.barIdList {
		if err := s.removeBAR(&ie.RemoveBAR{
			BARID: id,
		}); err != nil {
			s.Errorf("Remove BAR err: %+v", err)
		}
	}

	for _, q := range s.q {
		close(q)
	}
	return usars
}

func (s *PfcpSession) createPDR(req *ie.CreatePDR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	pdrId := uint16(req.PDRID)

	urrids := make(map[uint32]struct{})
	if len(req.URRID) > 0 {
		for _, urr := range req.URRID {
			urrids[uint32(urr)] = struct{}{}
			if urrInfo, ok := s.urrIdList[uint32(urr)]; ok {
				urrInfo.refPdrNum++
			}
		}
	}

	s.pdrList[pdrId] = &PDRInfo{
		RelatedurrIdList: urrids,
	}

	if err := _sessMan.driver.CreatePDR(s.localID, req); err != nil {
		return utils.WrapError("Create PDR", err)
	}
	s.Debugf("PDR %d created", pdrId)

	return nil
}

func (s *PfcpSession) diassociateURR(urrid uint32) []report.USAReport {

	urrInfo, ok := s.urrIdList[urrid]
	if !ok {
		return nil
	}

	if urrInfo.refPdrNum > 0 {
		urrInfo.refPdrNum--
		if urrInfo.refPdrNum == 0 {
			// indicates usage report being reported for a URR due to dissociated from the last PDR
			usars, err := _sessMan.driver.QueryURR(s.localID, urrid)
			if err != nil {
				return nil
			}
			for i := range usars {
				usars[i].USARTrigger.Flags |= report.USAR_TRIG_TERMR
			}
			return usars
		}
	} else {
		s.Warnf("diassociateURR: wrong refPdrNum(%d)", urrInfo.refPdrNum)
	}
	return nil
}

func (s *PfcpSession) updatePDR(req *ie.UpdatePDR) ([]report.USAReport, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	pdrId := uint16(req.PDRID)
	newUrrids := make(map[uint32]struct{})
	newUrrids[uint32(*req.URRID)] = struct{}{}

	pdrInfo, ok := s.pdrList[pdrId]
	if !ok {
		return nil, errors.Errorf("UpdatePDR: PDR(%#x) not found", pdrId)
	}

	if err := _sessMan.driver.UpdatePDR(s.localID, req); err != nil {
		return nil, utils.WrapError("Update PDR", err)
	}

	var usars []report.USAReport
	for urrid := range pdrInfo.RelatedurrIdList {
		_, ok = newUrrids[urrid]
		if !ok {
			usar := s.diassociateURR(urrid)
			if len(usar) > 0 {
				usars = append(usars, usar...)
			}
		}
	}
	pdrInfo.RelatedurrIdList = newUrrids

	s.Debugf("PDR %d updated", pdrId)
	return usars, nil
}

func (s *PfcpSession) removePDR(req *ie.RemovePDR) ([]report.USAReport, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	pdrId := uint16(req.PDRID)
	pdrInfo, ok := s.pdrList[pdrId]
	if !ok {
		return nil, errors.Errorf("RemovePDR: PDR(%d) not found", pdrId)
	}

	if err := _sessMan.driver.RemovePDR(s.localID, req); err != nil {
		return nil, utils.WrapError("PDR removed", err)
	}

	var usars []report.USAReport
	for urrid := range pdrInfo.RelatedurrIdList {
		usar := s.diassociateURR(urrid)
		if len(usar) > 0 {
			usars = append(usars, usar...)
		}
	}
	delete(s.pdrList, pdrId)
	s.Debugf("PDR %d removed", pdrId)
	return usars, nil
}

func (s *PfcpSession) createFAR(req *ie.CreateFAR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.FARID)
	s.farIdList[id] = struct{}{}

	if err := _sessMan.driver.CreateFAR(s.localID, req); err != nil {
		return utils.WrapError("Create FAR", err)
	}
	s.Debugf("FAR %d created", id)
	return nil
}

func (s *PfcpSession) updateFAR(req *ie.UpdateFAR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.FARID)

	_, ok := s.farIdList[id]
	if !ok {
		return errors.Errorf("UpdateFAR: FAR(%d) not found", id)
	}
	if err := _sessMan.driver.UpdateFAR(s.localID, req); err != nil {
		return utils.WrapError("Update FAR", err)
	}

	s.Debugf("FAR %d updated", id)
	return nil
}

func (s *PfcpSession) removeFAR(req *ie.RemoveFAR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.FARID)

	_, ok := s.farIdList[id]
	if !ok {
		return errors.Errorf("RemoveFAR: FAR(%#x) not found", id)
	}

	err := _sessMan.driver.RemoveFAR(s.localID, req)
	if err != nil {
		return utils.WrapError("Remove FAR", err)
	}

	delete(s.farIdList, id)
	s.Debugf("FAR %d removed", id)
	return nil
}

func (s *PfcpSession) createQER(req *ie.CreateQER) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.QERID)
	s.qerIdList[id] = struct{}{}

	if err := _sessMan.driver.CreateQER(s.localID, req); err != nil {
		return utils.WrapError("Create QER", err)
	}
	s.Debugf("QER %d created", id)
	return nil
}

func (s *PfcpSession) updateQER(req *ie.UpdateQER) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.QERID)

	_, ok := s.qerIdList[id]
	if !ok {
		return errors.Errorf("UpdateQER: QER(%#x) not found", id)
	}
	if err := _sessMan.driver.UpdateQER(s.localID, req); err != nil {
		return utils.WrapError("Update QER", err)
	}

	s.Debugf("QER %d updated", id)
	return nil
}

func (s *PfcpSession) removeQER(req *ie.RemoveQER) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.QERID)

	_, ok := s.qerIdList[id]
	if !ok {
		return errors.Errorf("RemoveQER: QER(%d) not found", id)
	}

	if err := _sessMan.driver.RemoveQER(s.localID, req); err != nil {
		return utils.WrapError("Remove QER", err)
	}

	delete(s.qerIdList, id)
	s.Debugf("QER %d removed", id)
	return nil
}

func (s *PfcpSession) createURR(req *ie.CreateURR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.URRID)

	s.urrIdList[id] = &URRInfo{
		MeasureMethod: ie.MeasureMethod{
			DURAT: req.MeasurementMethod.HasDURAT(),
			VOLUM: req.MeasurementMethod.HasVOLUM(),
			EVENT: req.MeasurementMethod.HasEVENT(),
		},
	}
	if req.MeasurementInformation != nil {
		s.urrIdList[id].MeasurementInformation = *req.MeasurementInformation
	}

	if err := _sessMan.driver.CreateURR(s.localID, req); err != nil {
		return utils.WrapError("Create URR", err)
	}
	s.Debugf("URR %d created", id)
	return nil
}

func (s *PfcpSession) updateURR(req *ie.UpdateURR) ([]report.USAReport, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.URRID)

	urrInfo, ok := s.urrIdList[id]
	if !ok {
		return nil, errors.Errorf("UpdateURR: URR[%d] not found", id)
	}
	urrInfo.DURAT = req.MeasurementMethod.HasDURAT()
	urrInfo.VOLUM = req.MeasurementMethod.HasVOLUM()
	urrInfo.EVENT = req.MeasurementMethod.HasEVENT()

	urrInfo.Mbqe = req.MeasurementInformation.HasMBQE()
	urrInfo.Inam = req.MeasurementInformation.HasINAM()
	urrInfo.Radi = req.MeasurementInformation.HasRADI()
	urrInfo.Istm = req.MeasurementInformation.HasISTM()
	urrInfo.Mnop = req.MeasurementInformation.HasMNOP()
	usars, err := _sessMan.driver.UpdateURR(s.localID, req)
	if err != nil {
		return nil, utils.WrapError("Update URR", err)
	}
	s.Debugf("URR %d updated", id)
	return usars, nil
}

func (s *PfcpSession) removeURR(req *ie.RemoveURR) ([]report.USAReport, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.URRID)

	info, ok := s.urrIdList[id]
	if !ok {
		return nil, errors.Errorf("RemoveURR: URR[%d] not found", id)
	}
	info.removed = true // remove URRInfo later

	usars, err := _sessMan.driver.RemoveURR(s.localID, req)
	if err != nil {
		return nil, utils.WrapError("Remove URR", err)
	}

	// indicates usage report being reported for a URR due to the removal of the URR
	for i := range usars {
		usars[i].USARTrigger.Flags |= report.USAR_TRIG_TERMR
	}

	s.Debugf("URR %d removed", id)
	return usars, nil
}

func (s *PfcpSession) queryURR(req *ie.QueryURR) ([]report.USAReport, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint32(req.URRID)

	_, ok := s.urrIdList[id]
	if !ok {
		return nil, errors.Errorf("QueryURR: URR[%d] not found", id)
	}

	usars, err := _sessMan.driver.QueryURR(s.localID, id)
	if err != nil {
		return nil, err
	}

	// indicates an immediate report reported on CP function demand
	for i := range usars {
		usars[i].USARTrigger.Flags |= report.USAR_TRIG_IMMER
	}
	return usars, nil
}

func (s *PfcpSession) createBAR(req *ie.CreateBAR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint8(req.BARID)
	s.barIdList[id] = struct{}{}

	if err := _sessMan.driver.CreateBAR(s.localID, req); err != nil {
		return utils.WrapError("Create BAR", err)
	}
	s.Debugf("BAR %d created", id)
	return nil
}

func (s *PfcpSession) updateBAR(req *ie.UpdateBARSessionModificationRequest) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint8(req.BARID)

	if _, ok := s.barIdList[id]; !ok {
		return errors.Errorf("UpdateBAR: BAR(%#x) not found", id)
	}
	if err := _sessMan.driver.UpdateBAR(s.localID, req); err != nil {
		return utils.WrapError("Update BAR", err)
	}

	s.Debugf("BAR %d updated", id)
	return nil
}

func (s *PfcpSession) removeBAR(req *ie.RemoveBAR) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id := uint8(req.BARID)

	if _, ok := s.barIdList[id]; !ok {
		return errors.Errorf("RemoveBAR: BAR(%d) not found", id)
	}

	err := _sessMan.driver.RemoveBAR(s.localID, req)
	if err != nil {
		return err
	}

	delete(s.barIdList, id)
	s.Debugf("BAR %d removed", id)
	return nil
}

/*
	func (s *PfcpSession) Push(pdrId uint16, p []byte) {
		pkt := make([]byte, len(p))
		copy(pkt, p)
		q, ok := s.q[pdrId]
		if !ok {
			s.q[pdrId] = make(chan []byte, s.qlen)
			q = s.q[pdrId]
		}

		select {
		case q <- pkt:
			s.Debugf("Push bufPkt to q[%d](len:%d)", pdrId, len(q))
		default:
			s.Debugf("q[%d](len:%d) is full, drop it", pdrId, len(q))
		}
	}

	func (s *PfcpSession) Len(pdrId uint16) int {
		q, ok := s.q[pdrId]
		if !ok {
			return 0
		}
		return len(q)
	}

	func (s *PfcpSession) Pop(pdrId uint16) ([]byte, bool) {
		q, ok := s.q[pdrId]
		if !ok {
			return nil, ok
		}
		select {
		case pkt := <-q:
			s.Debugf("Pop bufPkt from q[%d](len:%d)", pdrId, len(q))
			return pkt, true
		default:
			return nil, false
		}
	}
*/
func (s *PfcpSession) URRSeq(urrid uint32) uint32 {
	info, ok := s.urrIdList[urrid]
	if !ok {
		return 0
	}
	seq := info.SEQN
	info.SEQN++
	return seq
}
