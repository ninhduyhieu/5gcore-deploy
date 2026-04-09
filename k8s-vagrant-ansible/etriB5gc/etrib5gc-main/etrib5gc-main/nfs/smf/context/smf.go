package context

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/nfs/smf/config"

	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"net"
)

const (
	MAX_NUM_SMCONTEXTS    uint = 16000
	MAX_NUM_PFCP_SESSIONS uint = 16000
)

type SmfContext struct {
	logctx.LogWriter

	plmnId models.PlmnId
	snssai models.Snssai

	transportNetworks []string
	dataNetworks      map[string]*DataNetwork

	nodeId ie.NodeID //SMF identity in PFCP

	sessionType string //defaul pdu session type (in case UE does not specify)

	doneCh chan bool
	// closed int32
	maxNumSmContexts   uint
	maxNumPfcpSessions uint
}

var _ctx *SmfContext

func InitContext(cfg *config.Config) (err error) {
	if _ctx == nil {
		_ctx = &SmfContext{
			LogWriter: logctx.WithFields(logctx.Fields{
				"mod": "context",
			}),
			plmnId:             models.PlmnId(cfg.PlmnId),
			doneCh:             make(chan bool),
			snssai:             cfg.Slice,
			sessionType:        "IPv4",
			dataNetworks:       make(map[string]*DataNetwork),
			maxNumSmContexts:   MAX_NUM_SMCONTEXTS,
			maxNumPfcpSessions: MAX_NUM_PFCP_SESSIONS,
		}
		var ip net.IP
		if ip, err = common.GetPrimaryIP(); err != nil {
			err = utils.WrapError("Get Primary IP", err)
		} else {
			_ctx.nodeId = ie.NewNodeIDv4(ip)
		}
	}
	return
}

func HasTransportNetwork(netName string) bool {
	for _, tn := range _ctx.transportNetworks {
		if tn == netName {
			return true
		}
	}
	return false
}

func GetPfcpNodeId() *ie.NodeID {
	return &_ctx.nodeId
}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnId
}

func Snssai() *models.Snssai {
	return &_ctx.snssai
}

func DefaultPduSessionType() (t uint8) {
	switch _ctx.sessionType {
	case "IPv4":
		t = nas.PduSessionTypeIpv4
	case "IPv6":
		t = nas.PduSessionTypeIpv6
	case "IPv4v6":
		t = nas.PduSessionTypeIpv4Ipv6
	case "Ethernet":
		t = nas.PduSessionTypeEthernet
	default:
		t = nas.PduSessionTypeIpv4
	}
	return
}

func IsIpv4SessionSupported() bool {
	if _ctx.sessionType == "IPv4" || _ctx.sessionType == "IPv4v6" {
		return true
	}
	return false
}

func IsIpv6SessionSupported() bool {
	if _ctx.sessionType == "IPv6" || _ctx.sessionType == "IPv4v6" {
		return true
	}
	return false
}

func IsEthernetSessionSupported() bool {
	return _ctx.sessionType == "Ethernet"
}

func MaxNumSmContexts() int {
	return int(_ctx.maxNumSmContexts)
}

func MaxNumPfcpSessions() int {
	return int(_ctx.maxNumPfcpSessions)
}

func AllocateUeIp(dnn string) (ip *LeasedIp, err error) {
	if allocator, ok := _ctx.dataNetworks[dnn]; !ok {
		err = fmt.Errorf("Can't allocate UeIP for the data network[%s]", dnn)
	} else {
		ip, err = allocator.allocateUeIp()
	}
	return
}

func GetDns(dnn string) (v4, v6 net.IP) {
	if dn, ok := _ctx.dataNetworks[dnn]; ok {
		if dn.dns != nil {
			v4, v6 = dn.dns.v4, dn.dns.v6
		}
	}
	return
}
func GetPcscf(dnn string) (v4, v6 net.IP) {
	if dn, ok := _ctx.dataNetworks[dnn]; ok {
		if dn.pcscf != nil {
			v4, v6 = dn.pcscf.v4, dn.pcscf.v6
		}
	}
	return
}

func Ipv4LinkMtu() uint16 {
	return 1400
}
