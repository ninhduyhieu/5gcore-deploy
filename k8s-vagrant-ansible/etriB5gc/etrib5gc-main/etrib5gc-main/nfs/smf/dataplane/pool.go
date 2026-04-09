package dataplane

import (
	"etrib5gc/nfs/smf/context"
	"github.com/reogac/utils/idgen"
	"math"
	"sync"
	"sync/atomic"

	"github.com/alitto/pond/v2"
)

type SessionPool struct {
	sessions map[uint64]*PfcpSession
	seidGen  idgen.Generator[uint64]
	closed   int32
	workers  pond.Pool
	mutex    sync.RWMutex
}

var _pool *SessionPool

func InitPfcpSessionPool() {
	if _pool == nil {
		maxNumPfcpSessions := context.MaxNumPfcpSessions()
		_pool = &SessionPool{
			sessions: make(map[uint64]*PfcpSession),
			seidGen:  idgen.NewGenerator[uint64](1, math.MaxUint64),
			workers:  pond.NewPool(maxNumPfcpSessions),
		}
	}
}

func CleanPfcpSessionPool() {
	if _pool != nil {
		atomic.StoreInt32(&_pool.closed, 1) //do not accept new SmContext
		//TODO: clean all sessions
		_pool.workers.StopAndWait()
	}

}

func (p *SessionPool) isClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *SessionPool) add(s *PfcpSession) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.sessions[s.localSeid] = s
}

func (p *SessionPool) remove(s *PfcpSession) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.sessions, s.localSeid)
	p.seidGen.Free(s.localSeid)
}

func Worker() pond.Pool {
	return _pool.workers
}
