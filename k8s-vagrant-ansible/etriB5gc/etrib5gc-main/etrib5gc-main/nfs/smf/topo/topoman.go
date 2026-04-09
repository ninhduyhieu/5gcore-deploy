package topo

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/sbi/apis/upf/n4sbi"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
	"sync"
)

var _man *upfTopoMan

func GetPath(query UpfQuery) (datapath UpfPath, err error) {
	return _man.getPath(query)
}

func HandleUpfJoin(ep *models.EndpointInfo, dat []byte) {
	_man.handleUpfJoin(ep, dat)
}

func HandleUpfLeft(uuid string) {
	_man.handleUpfLeft(uuid)
}

var log *logrus.Entry

type upfTopoMan struct {
	nodes map[string]*UpfNode //indexed by uuid
	faces map[string]NetInf
	mutex sync.RWMutex //for read/write UpfNode lists
}

func Init() {
	if _man != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "topo",
	})
	_man = &upfTopoMan{
		nodes: make(map[string]*UpfNode),
		faces: make(map[string]NetInf),
	}
}

func (t *upfTopoMan) removeUpf(uuid string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if upf, ok := t.nodes[uuid]; ok {
		delete(t.nodes, uuid)
		for _, f := range upf.getInfs() {
			delete(t.faces, f.id)
		}
		log.Warnf("Upf %s removed", uuid)
	}
}

func (t *upfTopoMan) addUpf(info *common.UpfInfo, sbiEp *models.EndpointInfo) (err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// parse information to create a new node
	var upf *UpfNode
	if upf, err = createUpfNode(t, info, sbiEp); err != nil {
		log.Errorf(err.Error())
		return
	}
	log.Infof("Add UPF: %s", upf.String())
	// add node
	t.nodes[upf.id] = upf
	for _, f := range upf.getInfs() {
		t.faces[f.id] = f
	}
	return
}

func (t *upfTopoMan) close() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	smfId := mesh.EndpointInfo().Uuid
	for _, node := range t.nodes {
		//disassociate
		if err := n4sbi.DisassociationRequest(node.upfCli, smfId); err != nil {
			log.Errorf("Disassociate %s return error: %+v", smfId, err)
		}
	}
	t.nodes = make(map[string]*UpfNode)
}

func (t *upfTopoMan) handleUpfJoin(ep *models.EndpointInfo, dat []byte) {
	if len(dat) > 0 {
		var upfinfo common.UpfInfo
		if err := json.Unmarshal(dat, &upfinfo); err != nil {
			log.Errorf("Fail to parse config from %s: %+v", ep.Uuid, err)
			return
		}
		//add UPF to the topo if it is in the expected slice
		if err := t.addUpf(&upfinfo, ep); err != nil {
			log.Warnf("Fail to add UPF: %+v", err)
		}
	}
}

func (t *upfTopoMan) handleUpfLeft(uuid string) {
	//remove UPF from the topo
	t.removeUpf(uuid)
}
