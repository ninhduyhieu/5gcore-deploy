package mesh

import (
	"crypto/tls"
	"crypto/x509"
	"etrib5gc/mesh/httpdp"
	"etrib5gc/mesh/registry"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"
	"math/rand"
	"time"
)

const (
	EV_EP_LEFT EventType = iota
	EV_EP_JOIN
)

type HasSharedConfig interface{ GetSharedConfig() []byte }

type EventType uint8

type EventInfo struct {
	EvType  EventType
	Content interface{}
}

type EpJoinEventInfo struct {
	Services []string            //serving services
	Endpoint models.EndpointInfo //endpoint address
	Config   []byte              //configuration shared by endpoint (can be nil)
}

type App interface {
	GetRouteGroups() []httpw.RouteGroup
}

// mesh initialization options
type Options struct {
	Config             *MeshConfig
	SbiPending         bool           //sbi server is pending until NF asks agent to activation
	App                App            //interface to application (NF) layer
	SubscribedServices []string       //services to subscribe for receiving enpoind update events
	EvCh               chan EventInfo //to receive endpoint events
	SbiPrefix          string         //prefix for SBI routes (can be empty)
	Cert               tls.Certificate
	CaPool             *x509.CertPool
	CertName           string
	GatewayName        string //gateway name for certificate verification
}

// initialize service mesh agent
func Init(opts Options) error {
	if _mesh != nil {
		return nil
	}
	rand.Seed(time.Now().UnixNano())
	return createMesh(&opts)
}

// Terminate agent
func Terminate() error {
	if _mesh != nil {
		m := _mesh
		_mesh = nil
		return m.stop()
	}
	return nil
}

// get mesh registered address of the agent
func EndpointInfo() *models.EndpointInfo {
	if _mesh != nil {
		ep := _mesh.agent.GetEndpointInfo()
		return &ep
	}
	return nil
}

// create an Sbi client toward a service given its name and optional paramaters
// for using in endpoint selection
func Consumer(serviceId string, options map[string]string) (cli *httpdp.Client, err error) {
	var matched *registry.MatchedGroup
	if matched, err = _mesh.agent.Search(serviceId, options); err == nil {
		if matched.IsStateful() {
			ep := matched.Select()
			cli = httpdp.NewStatefulClient(serviceId, ep)
		} else {
			cli = httpdp.NewStatelessClient(serviceId, matched)
		}
	} else {
		err = utils.WrapError("Create Sbi Consumer", err)
	}
	return
}

// create an Sbi client using service name and instance identity (AMFPointer)
func ConsumerWithInstanceId(serviceId, insId string) (cli *httpdp.Client, err error) {
	var ep *registry.Endpoint
	if ep, err = _mesh.agent.GetServiceInstance(serviceId, insId); err == nil {
		cli = httpdp.NewClientWithEndpoint(ep, "")
	}
	return
}

// create a http client with an endpoint information
func ConsumerFromEndpoint(epInfo *models.EndpointInfo) (cli *httpdp.Client, err error) {
	var ep *registry.Endpoint
	if ep, err = _mesh.agent.GetEndpoint(epInfo); err == nil {
		cli = httpdp.NewClientWithEndpoint(ep, epInfo.Stickiness)
	}
	return
}

// set sbi server state to active
// send an registration update to controller
// set an customized service instance id (AMF pointer)
func ActivateSbi(insId string) error {
	return _mesh.agent.ActivateSbi(insId)
}
