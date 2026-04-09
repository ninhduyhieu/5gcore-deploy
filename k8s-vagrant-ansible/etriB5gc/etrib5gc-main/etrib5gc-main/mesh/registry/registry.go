package registry

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh/models"
	"fmt"
	sbimodels "github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	REGISTER_INTERVAL    int = 5 //second
	MAX_REGISTRATION_TRY int = 3
)

type Options struct {
	Cli              *httpw.Client
	Registrar        string //registrar address (addr:port)
	SbiUrl, AgentUrl string
	Labels           map[string]string //endpoint labels if not running in K8s
	OnEpLeft         func(string)
	OnEpJoin         func(sbimodels.EndpointInfo, []string, []byte)
	Cert             tls.Certificate
	CaPool           *x509.CertPool
	CertName         string
	GatewayName      string //Gateway name for certificate verification
}

type Registry struct {
	sbiUrl, agentUrl string
	labels           map[string]string

	cli    *httpw.Client //client to the service controller
	cert   *tls.Certificate
	caPool *x509.CertPool
	name   string //agent certificate's name

	id     string // genereted by service controller
	insId  string //customized service instance id (set by application)
	gwAddr string //gateway URL return from the gateway (may be different from the registra Url)
	gwId   string

	epman EpManager      //endpoint manager
	sman  ServiceManager //service manager

	registered int32
}

var _agent *Registry

var log *logrus.Entry

func CreateRegistry(opts Options) *Registry {
	if _agent != nil {
		return _agent
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "agent",
	})

	reg := &Registry{
		sbiUrl:   opts.SbiUrl,
		agentUrl: opts.AgentUrl,
		gwAddr:   opts.Registrar,
		labels:   opts.Labels,

		name:       opts.CertName,
		registered: 0,
	}
	if opts.CaPool != nil {
		reg.cert = &opts.Cert
		reg.caPool = opts.CaPool
	}

	reg.cli = httpw.NewClient(&opts.Cert, opts.CaPool, opts.GatewayName)

	reg.epman = newEpManager(reg)
	reg.epman.onEpLeft = opts.OnEpLeft
	reg.epman.onEpJoin = opts.OnEpJoin
	reg.sman = newServiceManager(&reg.epman)
	_agent = reg

	return reg
}

func (reg *Registry) HttpRoutes() []httpw.Route {
	return []httpw.Route{
		httpw.Route{
			Method:  http.MethodPost,
			Pattern: "/update",
			Handler: reg.onEndpointUpdate,
		},
		httpw.Route{
			Method:  http.MethodGet,
			Pattern: "/ping",
			Handler: reg.onPing,
		},
		httpw.Route{
			Method:  http.MethodGet,
			Pattern: "/mon/services",
			Handler: reg.onGetServices,
		},
		httpw.Route{
			Method:  http.MethodGet,
			Pattern: "/mon/endpoints",
			Handler: reg.onGetEndpoints,
		},
	}
}

// check if another endpoint of the same gateway is reachable by its sbi URI
func (reg *Registry) isReachable(addr string) bool {
	//get endpoint IP
	parts := strings.Split(addr, ":")
	ip := net.ParseIP(parts[0])
	if len(ip) == 0 { //not a valid IP (may be an FQDN)
		return false
	}

	parts = strings.Split(reg.sbiUrl, ":")
	myIp := net.ParseIP(parts[0])
	if len(myIp) == 0 { //not a valid IP (may be an FQDN)
		return false
	}
	if myIp.IsLoopback() {
		if ip.IsLoopback() { //both are loop back IPs
			return true
		} else {
			return false
		}
	} else {
		if ip.IsLoopback() {
			return false
		}
	}
	//both are not loopback IPs
	return common.IsSameNetwork(ip)
}

//make an http request to send to the registrar, get http response and read
//respond body
func (reg *Registry) sendHttpRequest(msg any, method string, suffix string) (rsp *http.Response, rspBody []byte, err error) {
	var reqBody io.Reader
	if msg != nil {
		var buf []byte
		if buf, err = json.Marshal(msg); err != nil {
			return nil, nil, utils.WrapError("Endcode request body", err)
		}
		reqBody = bytes.NewBuffer(buf)
	}
	url := fmt.Sprintf("%s/%s", reg.gwAddr, suffix)
	//log.Tracef("Send request %s", url)
	rsp, rspBody, err = reg.cli.Send(method, url, reqBody)
	return
}

// register to the service mesh control plane
func (reg *Registry) Register(sbiPrefix string, configDat []byte, pending bool, quit chan struct{}) error {
	podName := os.Getenv("POD_NAME")
	podNameSpace := os.Getenv("POD_NAMESPACE")
	if len(podName) > 0 {
		log.Debugf("Running in K8s")
	} else {
		log.Debugf("Running in K8s")
	}

	msg := &models.RegistrationRequest{
		PodName:      podName,
		PodNameSpace: podNameSpace,
		SbiUrl:       reg.sbiUrl,
		SbiPrefix:    sbiPrefix,
		AgentUrl:     reg.agentUrl,
		CertName:     reg.name,
		Labels:       reg.labels,
		Active:       !pending,
		Config:       configDat,
	}

	// register to the mesh, try several times
	cnt := 0
	for {
		if rsp, rspBody, err := reg.sendHttpRequest(msg, http.MethodPost, "register"); err != nil {
			log.Warnf("Fail to send registration request: %+v", err)
		} else {
			if rsp.StatusCode != http.StatusCreated {
				log.Warnf("Registration failed [%d]: %s (%s)", rsp.StatusCode, rsp.Status, string(rspBody))
			} else {
				var rspMsg models.RegistrationResponse
				if err = json.Unmarshal(rspBody, &rspMsg); err != nil {
					log.Warnf("Fail to decode registration response: %+v", err)
				} else {
					atomic.StoreInt32(&reg.registered, 1)
					//update agent uuid generated by the controller
					reg.id = rspMsg.Id
					//update gateway information
					reg.gwId = rspMsg.GwId
					log.Infof("Agent registered with id: %s[gwId=%s]", rspMsg.Id, reg.gwId)
					return nil
				}
			}
		}

		cnt++
		if cnt >= MAX_REGISTRATION_TRY {
			return fmt.Errorf("Fail registration")
		}
		t := time.NewTimer(time.Duration(REGISTER_INTERVAL) * time.Second)
		select {
		case <-quit:
			return fmt.Errorf("Interupted")
		case <-t.C:
		}
	}

	return nil
}

// update to controller that Sbi server is ready to handle requests
func (reg *Registry) ActivateSbi(insId string) error {
	reg.insId = insId
	msg := &models.UpdateNotification{
		InsId:  reg.insId,
		Active: true,
	}

	rsp, _, err := reg.sendHttpRequest(msg, http.MethodPost, fmt.Sprintf("activate/%s", reg.id))
	if err != nil {
		return utils.WrapError("Send SBI activate notification", err)
	}

	if rsp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Update failed")
	} else {
		log.Infof("Sbi server activated")
	}
	return nil
}

// deregister to the service controller
func (reg *Registry) Deregister() error {
	if atomic.LoadInt32(&reg.registered) != 1 {
		return ErrNotRegistered
	}
	log.Debugf("Send deregistration request")
	if _, _, err := reg.sendHttpRequest(nil, http.MethodPost, fmt.Sprintf("deregister/%s", reg.id)); err != nil {
		return utils.WrapError("Send deregistration request", err)
	}
	return nil
}

// send subscription to services and get endpoin list + routing information
func (reg *Registry) Subscribe(services []string) error {
	log.Debugf("Send service subscription for services: %s", strings.Join(services, ", "))

	msg := &models.SubscribeRequest{
		Services: services,
	}
	rsp, rspBody, err := reg.sendHttpRequest(msg, http.MethodPost, fmt.Sprintf("subscribe/%s", reg.id))
	if err != nil {
		return utils.WrapError("Send subscription request", err)
	}

	var rspMsg models.SubscribeResponse

	if rsp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Subscribe failed [%d]: %s (%s)", rsp.StatusCode, rsp.Status, string(rspBody))
	}

	if err = json.Unmarshal(rspBody, &rspMsg); err != nil {
		return utils.WrapError("Decode subscription response message", err)
	}
	//add services and add existing endpoints
	reg.sman.addServices(rspMsg.Services)
	//add new endpoints
	reg.epman.addEndpoints(rspMsg.Endpoints)
	return nil
}

func (reg *Registry) onEndpointUpdate(ctx *gin.Context) {
	log.Debugf("Receive Enpoint Update")
	var err error
	var dat []byte
	var msg models.EndpointUpdates
	//parse message
	if dat, err = ioutil.ReadAll(ctx.Request.Body); err != nil {
		log.Warnf("Fail to read request body: %+v", err)
	} else if err = json.Unmarshal(dat, &msg); err != nil {
		log.Warnf("Fail to decode request body: %+v", err)
	} else {
		reg.epman.updateEndpoints(&msg)
		ctx.Status(http.StatusCreated)
		return
	}
	ctx.String(http.StatusInternalServerError, "Invalid request")
}

func (reg *Registry) onPing(ctx *gin.Context) {
	log.Trace("Receive a ping")
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (reg *Registry) Search(serviceId string, options map[string]string) (*MatchedGroup, error) {
	if atomic.LoadInt32(&reg.registered) != 1 {
		return nil, ErrNotRegistered
	}
	if s, err := reg.getService(serviceId); err != nil {
		return nil, utils.WrapError("Get service", err)
	} else {
		//service found, now pick an endpoint
		log.Debugf("Search endpoint for service %s with options = {%v}", serviceId, options)
		return s.match(options)
	}
}

func (reg *Registry) discoverEndpoint(uuid string) *Endpoint {
	log.Debugf("Discover the endpoint %s", uuid)
	if rsp, rspBody, err := reg.sendHttpRequest(nil, http.MethodGet, fmt.Sprintf("get/%s", uuid)); err != nil {
		log.Errorf("Fail to discover endpoint: %+v", err)
	} else if rsp.StatusCode == http.StatusOK {
		var info models.Endpoint
		if err := json.Unmarshal(rspBody, &info); err != nil {
			log.Errorf("Fail to decode endpoint infor: %+v", err)
		} else {
			return reg.createEndpoint(info)
		}
	}
	return nil
}

func (reg *Registry) getService(serviceId string) (*Service, error) {
	var s *Service
	if s = reg.sman.findService(serviceId); s == nil {
		log.Debugf("%s is not a subscribed service, send subscription", serviceId)
		if err := reg.Subscribe([]string{serviceId}); err != nil {
			//failed to subscribe service
			return nil, utils.WrapError("Send service subscription", err)
		} else {
			if s = reg.sman.findService(serviceId); s == nil {
				return nil, ErrNoService
			}
		}
	}
	return s, nil

}

func (reg *Registry) GetServiceInstance(serviceId, insId string) (*Endpoint, error) {
	if atomic.LoadInt32(&reg.registered) != 1 {
		return nil, ErrNotRegistered
	}

	if s, err := reg.getService(serviceId); err != nil {
		return nil, utils.WrapError("Get service", err)
	} else {
		if ep := s.searchInstance(insId); ep == nil {
			return nil, fmt.Errorf("instance id %s not found in service %s", insId, serviceId)
		} else {
			return ep, nil
		}
	}
}
