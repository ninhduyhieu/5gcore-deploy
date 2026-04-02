package ue

import (
	"etrib5gc/logctx"
	"fmt"
	//"github.com/alitto/pond/v2"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
	"sync/atomic"
)

type UePool struct {
	ueList         map[int64]*UeContext          //indexed by AmfUeId
	ranUeIdIndexes map[models.RanUeId]*UeContext //indexed by RanUeId
	mutex          sync.RWMutex
	ueIdGen        idgen.Generator[int64] // AmfUeId generator
	closed         int32
}

var _pool *UePool
var log *logrus.Entry

func InitUePool() {
	if _pool != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "ue",
	})

	_pool = &UePool{
		ueList:         make(map[int64]*UeContext),
		ranUeIdIndexes: make(map[models.RanUeId]*UeContext),
		ueIdGen:        idgen.NewGenerator[int64](0, math.MaxInt64),
		closed:         0,
	}
	initFsm()
}

// clear and close all UeContext in the pool
func CleanUePool() {
	if _pool == nil {
		return
	}
	//mark closed
	atomic.StoreInt32(&_pool.closed, 1)

	//stop all UEs
	log.Tracef("Close all UEs")

	for _, ueCtx := range _pool.listAll() {
		ueCtx.kill()
	}

	//kill worker pool
	log.Tracef("Stop worker pool")
	_sm.Stop()

	_pool = nil
}
func (p *UePool) listAll() []*UeContext {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return lo.Values(p.ueList)
}

// check if UeContext exists for a given RanUeId
func HasUe(id models.RanUeId) bool {
	_pool.mutex.RLock()
	defer _pool.mutex.RUnlock()
	_, ok := _pool.ranUeIdIndexes[id]
	return ok
}

// find UeContext by AmfUeId
func FindUe(id int64) *UeContext {
	_pool.mutex.RLock()
	defer _pool.mutex.RUnlock()
	ueCtx, _ := _pool.ueList[id]
	return ueCtx
}

//allocate AmfUeId then add new UeContext
func (p *UePool) add(ueCtx *UeContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ueCtx.amfUeId = _pool.ueIdGen.Allocate()

	ueCtx.Entry = log.WithFields(logrus.Fields{
		"ranUeId": ueCtx.ranUeId.String(),
		"amfUeId": fmt.Sprintf("%d", ueCtx.amfUeId),
	})

	p.ueList[ueCtx.amfUeId] = ueCtx
	p.ranUeIdIndexes[ueCtx.ranUeId] = ueCtx
}

func (p *UePool) removeUe(ueCtx *UeContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	id := ueCtx.amfUeId
	delete(p.ueList, id)
	delete(p.ranUeIdIndexes, ueCtx.ranUeId)
	p.ueIdGen.Free(id)
}

// Used for checking before creating a new UeContext
func (pool *UePool) isClosed() bool {
	return atomic.LoadInt32(&pool.closed) == 1
}
