package ranue

import (
	"math"
	"sync"

	"etrib5gc/logctx"
	amfctx "etrib5gc/nfs/amf/context"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
)

type RanUePool struct {
	ueIdGen idgen.Generator[int64]
	mutex   sync.RWMutex
	ranUes  map[int64]*RanUe
	indexes map[string]*RanUe
}

var _ranUePool *RanUePool
var log *logrus.Entry

func InitRanUePool() {
	if _ranUePool != nil {
		return
	}

	log = logctx.Entry(logrus.Fields{
		"mod": "ranue",
	})

	_ranUePool = &RanUePool{
		ueIdGen: idgen.NewGenerator[int64](1, math.MaxInt64),
		ranUes:  make(map[int64]*RanUe),
		indexes: make(map[string]*RanUe),
	}
}

func CloseRanUePool() {
	//TODO: nothing
}

func CreateRanUe(ueCtx UeContext, msg *models.InitialUeMessage, callback *models.EndpointInfo, gmmMsg *nas.DecodedGmmMessage) (*RanUe, error) {
	ran, err := amfctx.GetRan(callback)
	if err != nil {
		return nil, utils.WrapError("Fail to connect to RAN", err)
	}

	//remove orphan RanUe if it exists
	id := msg.RanUeId.String()
	_ranUePool.mutex.RLock()
	ranUe, ok := _ranUePool.indexes[id]
	_ranUePool.mutex.RUnlock()

	if ok {
		log.Warnf("Found an existing RanUe with RanUeId=%s, remove it", id)
		//detach from Ue context
		ranUe.ReleaseContext(models.N2Cause{})
	}

	//create a new RanUe
	return newRanUe(ueCtx, ran.Client(), msg, callback), nil
}

func (ranUe *RanUe) AddToPool() {
	_ranUePool.add(ranUe)
	ranUe.Entry = ranUe.WithFields(logrus.Fields{
		"amfUeId": ranUe.localId,
	})

}

func (p *RanUePool) add(ranUe *RanUe) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ranUe.localId = p.ueIdGen.Allocate()
	log.Infof("RanUe[AmfUeId=%d] is added to pool", ranUe.localId)
	p.ranUes[ranUe.localId] = ranUe
	p.indexes[ranUe.ranUeId.String()] = ranUe
}

func (p *RanUePool) remove(ranUe *RanUe) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.ranUes, ranUe.localId)
	delete(p.indexes, ranUe.ranUeId.String())
	p.ueIdGen.Free(ranUe.localId)
	log.Warnf("RanUe[AmfUeId=%d] removed", ranUe.localId)
}

func FindRanUe(ueId int64) *RanUe {
	_ranUePool.mutex.RLock()
	defer _ranUePool.mutex.RUnlock()
	ranUe, _ := _ranUePool.ranUes[ueId]
	return ranUe
}
