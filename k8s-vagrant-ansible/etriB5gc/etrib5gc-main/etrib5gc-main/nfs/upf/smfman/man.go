package smfman

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"etrib5gc/nfs/upf/context"
	"fmt"
	"github.com/reogac/pfcp/message"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
	"math"
)

const (
	NUM_TEID_RANGES int = 50
)

var log *logrus.Entry

type TeidRange struct {
	index    int
	l, u     uint32 //lower bound and upper bound
	assigned bool   //assigned to SMF?
}

type SmfManager struct {
	smfList    common.SimpleMap[string, Smf]
	teidRanges []TeidRange
}

var _smfMan *SmfManager

func Init() {
	if _smfMan == nil {
		log = logctx.Entry(logrus.Fields{
			"mod": "smfman",
		})

		_smfMan = createSmfManager()
	}
}

func createSmfManager() *SmfManager {
	getSmfId := func(smf *Smf) string {
		return smf.id
	}
	m := &SmfManager{
		smfList: common.NewSimpleMap[string, Smf](getSmfId),
	}

	//create TEID ranges
	rangeSize := math.MaxUint32 / NUM_TEID_RANGES
	ranges := make([]TeidRange, NUM_TEID_RANGES)
	for i := 0; i < NUM_TEID_RANGES; i++ {
		l := i * rangeSize
		u := l + rangeSize
		ranges[i] = TeidRange{
			index:    i,
			l:        uint32(l),
			u:        uint32(u),
			assigned: false,
		}
	}
	m.teidRanges = ranges
	return m
}

func RegisterSmf(callback models.EndpointInfo, req *message.PFCPAssociationSetupRequest) (rsp *message.PFCPAssociationSetupResponse, err error) {
	m := _smfMan

	if smf := m.smfList.Find(callback.Uuid); smf != nil {
		err = fmt.Errorf("Smf already associated")
		return
	}
	if req.PFCPASReqFlags == nil {
		err = fmt.Errorf("Bad request")
		return
	}
	reqContent := &message.AssociationRequest{}
	if json.Unmarshal(req.PFCPASReqFlags, reqContent); err != nil {
		return
	}
	if !context.HasSlice(&reqContent.Slice) {
		err = fmt.Errorf("Slice not supported")
		return
	}
	var cli sbi.ConsumerClient
	if cli, err = mesh.ConsumerFromEndpoint(&callback); err != nil {
		return
	}
	smf := &Smf{
		smfCli: cli,
		id:     callback.Uuid,
		slice:  reqContent.Slice,
	}
	//allocate teid range
	if smf.teidRangeIndex, err = m.allocateTeidRange(); err != nil {
		return
	}
	defer func() {
		//undo allocations
		if err != nil {
			m.freeSmf(smf)
		}
	}()

	//add SMF to registered list
	m.smfList.Add(smf)
	log.Infof("Register Smf: %s", smf.id)
	//compose response message
	teidRange := m.teidRanges[smf.teidRangeIndex]
	rspContent := &message.AssociationResponse{
		TeidRange: message.TeidRange{
			LowerBound: teidRange.l,
			UpperBound: teidRange.u,
		},
	}
	rspContentBytes, _ := json.Marshal(rspContent)
	rsp = &message.PFCPAssociationSetupResponse{
		PFCPASRspFlags: rspContentBytes,
	}
	return
}

func UnregisterSmf(smfId string) (err error) {
	m := _smfMan

	var smf *Smf
	if smf = m.smfList.Find(smfId); smf == nil {
		err = fmt.Errorf("Smf not found")
		return
	}
	log.Infof("Unregister Smf: %s", smfId)
	//free allocated resources for SMF
	m.freeSmf(smf)
	//remove SMF
	m.smfList.Remove(smfId)
	return
}

func FindSmf(smfId string) *Smf {
	return _smfMan.smfList.Find(smfId)
}

func (m *SmfManager) freeSmf(smf *Smf) {
	//release allocated resources
	m.freeTeidRange(smf.teidRangeIndex)
}

func (m *SmfManager) allocateTeidRange() (index int, err error) {
	for i := 0; i < NUM_TEID_RANGES; i++ {
		if m.teidRanges[i].assigned == false {
			index = i
			m.teidRanges[i].assigned = true
			return
		}
	}
	err = fmt.Errorf("No more Teid range")

	return
}

func (m *SmfManager) freeTeidRange(index int) {
	if index >= 0 && index < NUM_TEID_RANGES {
		m.teidRanges[index].assigned = false
	}
}
