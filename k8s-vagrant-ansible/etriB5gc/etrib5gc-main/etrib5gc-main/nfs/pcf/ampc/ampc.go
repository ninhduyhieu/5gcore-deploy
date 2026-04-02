package ampc

import (
	"etrib5gc/logctx"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
)

var _ampc *AmPolicies
var log *logrus.Entry

func GetAmPolicies() *AmPolicies {
	if _ampc == nil {
		log = logctx.Entry(logrus.Fields{
			"mod": "ampc",
		})
		_ampc = newAmPolicies()
	}
	return _ampc
}

type AmPolicies struct {
	policies   map[string]*AmPolicy
	amPolIdGen idgen.Generator[uint32]
	mutex      sync.RWMutex
}

func newAmPolicies() *AmPolicies {
	return &AmPolicies{
		policies:   make(map[string]*AmPolicy),
		amPolIdGen: idgen.NewGenerator[uint32](1, math.MaxUint32),
	}
}

func (l *AmPolicies) CreateAmPolicy(req *models.PolicyAssociationRequest) (amPol *AmPolicy, err error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	amPol = &AmPolicy{
		info: *req, //should we make a deep-copy?
		pras: make(map[string]models.PresenceInfo),
	}

	idNum := uint32(l.amPolIdGen.Allocate())
	amPol.setId(idNum)
	amPol.Entry = log.WithFields(logrus.Fields{
		"supi":    req.Supi,
		"amPolId": fmt.Sprintf("%d", idNum),
	})
	//TODO: apply policy desicion to make change to the request

	l.policies[amPol.idStr] = amPol
	return
}

func (l *AmPolicies) DeleteAmPolicy(id string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if amPol, ok := l.policies[id]; ok {
		l.amPolIdGen.Free(amPol.id)
		delete(l.policies, id)
	}
}

func (l *AmPolicies) GetAmPolicy(id string) (amPol *AmPolicy) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	amPol, _ = l.policies[id]
	return
}

type AmPolicy struct {
	*logrus.Entry
	id       uint32
	idStr    string
	info     models.PolicyAssociationRequest
	triggers []models.RequestTrigger
	pras     map[string]models.PresenceInfo
}

func (amPol *AmPolicy) Id() string {
	return amPol.idStr
}
func (amPol *AmPolicy) setId(id uint32) {
	amPol.id = id
	amPol.idStr = fmt.Sprintf("%s-%d", amPol.info.Supi, id)
}

func (amPol *AmPolicy) Update(req *models.PolicyAssociationUpdateRequest) (rsp *models.PolicyUpdate, err error) {
	//TODO: update policy association and fill a report
	rsp = &models.PolicyUpdate{
		Pras: make(map[string]models.PresenceInfo),
	}
	for _, t := range amPol.triggers {
		rsp.Triggers = append(rsp.Triggers, string(t))
	}
	for k, v := range amPol.pras {
		rsp.Pras[k] = v
	}
	return
}

func (amPol *AmPolicy) PolicyAssociation() (rsp *models.PolicyAssociation) {
	rsp = &models.PolicyAssociation{
		Request: &amPol.info,
		Pras:    make(map[string]models.PresenceInfo),
	}
	for _, t := range amPol.triggers {
		rsp.Triggers = append(rsp.Triggers, string(t))
	}
	for k, v := range amPol.pras {
		rsp.Pras[k] = v
	}
	return
}
