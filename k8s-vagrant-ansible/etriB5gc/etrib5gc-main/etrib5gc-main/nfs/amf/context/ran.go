package context

import (
	"etrib5gc/common"
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/pran/subs"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/idgen"
	"math"
	"sync"
)

type RanList struct {
	remoteIdIndex map[string]*Ran
	list          map[string]*Ran //local id index
	idGen         idgen.Generator[uint16]
	mutex         sync.Mutex //for ranList
}

func newRanList() (rl RanList) {
	rl.remoteIdIndex = make(map[string]*Ran)
	rl.list = make(map[string]*Ran)
	rl.idGen = idgen.NewGenerator[uint16](0, math.MaxUint16)
	return
}

type Ran struct {
	plmnId  models.PlmnId
	id      string
	info    models.EndpointInfo
	localId uint16
	cli     sbi.ConsumerClient
	tacs    map[string][]models.Snssai
}

func (r *Ran) Client() sbi.ConsumerClient {
	return r.cli
}

func (r *Ran) RanInfo() *models.EndpointInfo {
	return &r.info
}

// check if RAN can serve one of the tracking area in the list
func (r *Ran) inTAList(list []models.Tai) bool {
	for _, item := range list {
		if common.IsPlmnIdEqual(&_ctx.plmnId, &item.PlmnId) {
			if _, ok := r.tacs[item.Tac]; ok {
				return true
			}
		}
	}
	return false
}

func (r *Ran) sendPaging(tmsi uint32, taList []models.Tai) error {
	//TODO: send Paging message to RAN
	return nil
}
func FindRan(targetId *models.GlobalRanNodeId) *Ran {
	//TODO
	return nil
}

func GetRan(ranInfo *models.EndpointInfo) (ran *Ran, err error) {
	_ctx.ranList.mutex.Lock()
	defer _ctx.ranList.mutex.Unlock()

	id := models.EndpointInfoToString(*ranInfo)
	var ok bool
	if ran, ok = _ctx.ranList.remoteIdIndex[id]; !ok {
		var cli sbi.ConsumerClient
		if cli, err = mesh.ConsumerFromEndpoint(ranInfo); err != nil {
			err = utils.WrapError("Create Ran consumer", err)
			return
		}

		//subscribe to RAN
		//guami := fmt.Sprintf("%s-%s", AmfIdString(), models.PlmnIdToString(_ctx.plmnId))
		localId := _ctx.ranList.idGen.Allocate()
		localIdStr := fmt.Sprintf("ran-%d", localId)

		req := &models.AmfSubscribeRequest{
			Id: localIdStr,
		}

		callback := mesh.EndpointInfo()

		var rsp *models.AmfSubscribeResponse
		if rsp, err = subs.AmfSubscribe(cli, callback, req); err == nil {
			ran = &Ran{
				plmnId:  rsp.PlmnId,
				info:    *ranInfo,
				id:      id,
				localId: localId,
				cli:     cli,
				tacs:    newSupportedTAList(rsp.SupportedTAList),
			}
			_ctx.ranList.remoteIdIndex[id] = ran
			_ctx.ranList.list[localIdStr] = ran
			log.Infof("Add Ran with supported TA list: %v", rsp.SupportedTAList)
		}
	}
	return
}

func UpdateRanInfo(ranId string, data *models.RanInfoUpdateData) {
	if ran, ok := _ctx.ranList.list[ranId]; ok {
		log.Infof("Update SupportedTAList for ran: %s", ranId)
		ran.tacs = newSupportedTAList(data.SupportedTAList)
	}
}

func newSupportedTAList(items []models.SupportedTAItem) (tacs map[string][]models.Snssai) {
	if len(items) > 0 {
		tacs = make(map[string][]models.Snssai)
		for _, item := range items {
			tac := fmt.Sprintf("%s", item.Tac)
			tacs[tac] = item.Slices
		}
	}
	return
}

func SendPaging(tmsi uint32, taList []models.Tai) error {
	for _, ran := range _ctx.ranList.list {
		if ran.inTAList(taList) {
			return ran.sendPaging(tmsi, taList)
		}
	}
	return fmt.Errorf("No RAN found for paging")
}
