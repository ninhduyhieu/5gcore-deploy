package sm

import (
	"etrib5gc/logctx"
	"etrib5gc/nfs/smf/context"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"math"
	"sync"
	"sync/atomic"

	"github.com/alitto/pond/v2"
	"github.com/sirupsen/logrus"
)

type SmContextPool struct {
	mutex         sync.Mutex
	list          map[string]*SmContext
	listBysupiRef map[string]*SmContext
	smIdGen       idgen.Generator[uint32]
	closed        int32
	workers       pond.Pool
}

var log *logrus.Entry
var _pool *SmContextPool

func InitSmContextPool() {
	if _pool != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "sm",
	})

	_pool = &SmContextPool{
		smIdGen:       idgen.NewGenerator[uint32](1, math.MaxUint32),
		list:          make(map[string]*SmContext),
		listBysupiRef: make(map[string]*SmContext),
		closed:        0,
		workers:       pond.NewPool(context.MaxNumSmContexts()),
	}
	initFsm()
}

func (p *SmContextPool) isClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *SmContextPool) size() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.list)
}

func (p *SmContextPool) getSmContexts() (smList []*SmContext) {
	p.mutex.Lock()
	for _, smCtx := range p.list {
		smList = append(smList, smCtx)
	}
	p.mutex.Unlock()
	return
}

func (p *SmContextPool) clean() {
	atomic.StoreInt32(&p.closed, 1) //do not accept new SmContext
	smList := p.getSmContexts()
	log.Infof("There are %d SmContexts to be removed", len(smList))
	var wg sync.WaitGroup
	for _, smCtx := range smList {
		wg.Add(1)
		go func(iWg *sync.WaitGroup) {
			smCtx.Close()
			iWg.Done()
		}(&wg)
	}
	wg.Wait()
	p.workers.StopAndWait()
}

func CleanSmContextPool() {
	if _pool != nil {
		_pool.clean()
		_sm.Stop()
	}
}

// find by identity created by SMF
func FindSmContext(ref string) (smCtx *SmContext) {
	_pool.mutex.Lock()
	defer _pool.mutex.Unlock()
	smCtx, _ = _pool.list[ref]
	return
}

// find by identity made of Supi and Pdu session ID
func FindSmContextBySupi(supi string, sid uint32) (smCtx *SmContext) {
	_pool.mutex.Lock()
	defer _pool.mutex.Unlock()
	ref := smContextRef(supi, sid)
	smCtx, _ = _pool.listBysupiRef[ref]
	return
}

func CreateSmContext(callback models.EndpointInfo, msg *models.SmContextCreateData) (smCtx *SmContext, err error) {
	if _pool.isClosed() {
		err = fmt.Errorf("SMF is terminated")
		return
	}

	smCtx, err = createSmContext(callback, msg)
	return
}

//NOTE: idempotent, can call multiple times
func (p *SmContextPool) remove(smCtx *SmContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ok := p.list[smCtx.idString()]; !ok {
		return
	}

	delete(p.list, smCtx.idString())
	p.smIdGen.Free(smCtx.id)

	//check the case where there has been a new session with the same Supi and
	//Pdu session Id
	ref := smCtx.supiRef()
	if s, ok := p.listBysupiRef[ref]; ok {
		if s == smCtx { //delete if it has not been overwritten by a new session context
			delete(p.listBysupiRef, ref)
		}
	}

	smCtx.Warnf("SmContext is removed from session pool")
}

func (p *SmContextPool) add(smCtx *SmContext) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	smCtx.id = _pool.smIdGen.Allocate()
	p.list[smCtx.idString()] = smCtx
	p.listBysupiRef[smCtx.supiRef()] = smCtx
	smCtx.Entry = smCtx.WithFields(logrus.Fields{
		"ref": smCtx.idString(),
	})
	smCtx.Infof("SmContext is added to session pool")
}
