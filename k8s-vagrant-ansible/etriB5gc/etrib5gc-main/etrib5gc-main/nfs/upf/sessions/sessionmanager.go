package sessions

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/nfs/upf/forwarder"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/sbi"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
	"math"
	"net"
)

type SessionManager struct {
	nodeId   ie.NodeID //Local pfcp NodeID
	driver   forwarder.Driver
	sessions common.SimpleMap[uint64, PfcpSession]
	seidGen  idgen.Generator[uint64]
}

var log *logrus.Entry
var _sessMan *SessionManager

func Init(nodeId net.IP, d forwarder.Driver) {
	if _sessMan == nil {
		log = logctx.Entry(logrus.Fields{
			"mod": "sessions",
		})

		_sessMan = createSessionManager(nodeId, d)
	}
}

func createSessionManager(nodeIp net.IP, d forwarder.Driver) (sm *SessionManager) {
	getSessionId := func(s *PfcpSession) uint64 {
		return s.localID
	}

	return &SessionManager{
		driver:   d,
		nodeId:   ie.NewNodeIDv4(nodeIp),
		sessions: common.NewSimpleMap[uint64, PfcpSession](getSessionId),
		seidGen:  idgen.NewGenerator[uint64](1, math.MaxUint64),
	}
}

func CreateSession(smfCli sbi.ConsumerClient, rSeid uint64) (s *PfcpSession) {
	lSeid := _sessMan.seidGen.Allocate()
	s = newPfcpSession(lSeid, rSeid, smfCli)
	_sessMan.sessions.Add(s)
	return
}

func FindSession(lSeid uint64) *PfcpSession {
	return _sessMan.sessions.Find(lSeid)
}

func DeleteSession(lSeid uint64) {
	_sessMan.sessions.Remove(lSeid)
	_sessMan.seidGen.Free(lSeid)
}
