package dataplane

import (
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/utils/idgen"
	"github.com/sirupsen/logrus"
	"math"
)

type UpfNode interface {
	GetId() string
	IsReachable([]string) bool
	WithFields(logrus.Fields) *logrus.Entry
}

type Upf struct {
	*logrus.Entry
	node     UpfNode
	cli      sbi.ConsumerClient //connection to UPF
	qerIdGen idgen.Generator[uint32]
	teidGen  idgen.Generator[uint32]
	farIdGen idgen.Generator[uint32]
	pdrIdGen idgen.Generator[uint16]
	sessions map[uint64]*PfcpSession //index by localSeid
}

func NewUpf(cli sbi.ConsumerClient, node UpfNode, teidL, teidU uint32) *Upf {
	upf := &Upf{
		node: node,
		Entry: node.WithFields(logrus.Fields{
			"mod": "dataplane",
		}),
		cli:      cli,
		teidGen:  idgen.NewGenerator[uint32](teidL, teidU),
		qerIdGen: idgen.NewGenerator[uint32](1, math.MaxUint32),
		farIdGen: idgen.NewGenerator[uint32](1, math.MaxUint32),
		pdrIdGen: idgen.NewGenerator[uint16](1, math.MaxUint16),
		sessions: make(map[uint64]*PfcpSession),
	}
	return upf
}

//check if the UPF connects to any of the transport networks
func (upf *Upf) IsReachable(nets []string) bool {
	return upf.node.IsReachable(nets)
}

func (upf *Upf) Id() string {
	return upf.node.GetId()
}

func (upf *Upf) CreateSession(ctx SmContext) (session *PfcpSession, err error) {
	upf.Infof("Create new Pfcp session at UPF %s", upf.Id())
	localSeid := _pool.seidGen.Allocate()
	session = &PfcpSession{
		Entry: upf.WithFields(logrus.Fields{
			"seid": fmt.Sprintf("%d", localSeid),
		}),
		ctx:       ctx,
		upf:       upf,
		localSeid: localSeid,
		links:     make(map[string]*Link),
		qosFlows:  make(map[uint8]*QosFlow),
	}
	_pool.add(session)
	return
}

func (upf *Upf) addSession(s *PfcpSession) {
	upf.sessions[s.localSeid] = s
}

func (upf *Upf) deleteSession(s *PfcpSession) {
	delete(upf.sessions, s.localSeid)
}
