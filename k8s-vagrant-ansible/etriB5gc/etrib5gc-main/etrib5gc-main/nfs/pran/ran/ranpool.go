package ran

import (
	"etrib5gc/logctx"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
	"math"
	"net"
	"sync"
)

type RanPool struct {
	ranIdGen idgen.Generator[uint16]
	gnbList  map[net.Conn]*Ran
	indexes  map[string]*Ran
	mutex    sync.Mutex
}

var _pool *RanPool
var log *logrus.Entry

func CreateRanPool() {
	if _pool != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "gnb",
	})

	_pool = &RanPool{
		ranIdGen: idgen.NewGenerator[uint16](1, math.MaxUint16),
		gnbList:  make(map[net.Conn]*Ran),
		indexes:  make(map[string]*Ran),
	}
}

func FindRanWithConn(conn net.Conn) *Ran {
	ran, _ := _pool.gnbList[conn]
	return ran
}

func FindRanWithNgapId(ranId *ies.GlobalRANNodeID) *Ran {
	_, id, _ := ranId2String(ranId)
	ran, _ := _pool.indexes[id]
	return ran
}

func FindRanWithId(ranId *models.GlobalRanNodeId) *Ran {
	//TODO
	return nil
}

func RemoveRan(gnB *Ran) {
	_pool.remove(gnB)
	//remove all associated UeContexts
	gnB.removeUes()
}

// add or update ran
func (pool *RanPool) add(gnB *Ran) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.gnbList[gnB.conn] = gnB
	pool.indexes[gnB.id] = gnB
}

func (pool *RanPool) remove(gnB *Ran) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	delete(pool.gnbList, gnB.conn)
	delete(pool.indexes, gnB.id)
}
func (pool *RanPool) updateRanId(gnB *Ran) {
	//	pool.ranList.UpdateId(RAN_INDEX_GNB, gnB)
}
