package mesh

import (
	"etrib5gc/common"
	"etrib5gc/mesh/registry"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"
	"github.com/reogac/utils/oam"
)

const (
	REGISTER_INTERVAL    int = 5 //second
	MAX_REGISTRATION_TRY int = 3
)

type HasOamBackend interface {
	CreateOamHandler(ext *oam.HandlerContext) *oam.OamHandler
}

type Running interface {
	Start() error
	Stop()
}

type b5gcMesh struct {
	cfg         MeshConfig
	app         App                //interface to application layer
	agent       *registry.Registry //local registry
	sbiServer   *httpw.Server      //sbi http server
	agentServer *httpw.Server      //agent http server
	runnings    []Running          //all running services
	quit        chan struct{}
	evCh        chan EventInfo
}

var _mesh *b5gcMesh

//start agent http server and register to the service mesh control plane
func createMesh(opts *Options) error {
	config := opts.Config
	m := &b5gcMesh{
		quit: make(chan struct{}, 1),
		evCh: opts.EvCh,
	}

	//get binding address
	bindIp, sbiBindPort, err := common.ParseHostPort(config.Bind, "0.0.0.0", SBI_PORT)
	if err != nil {
		return utils.WrapError("Parse binding address", err)
	}

	//set agent port
	agentBindPort := AGENT_PORT
	if config.AgentPort != nil {
		agentBindPort = fmt.Sprintf("%d", *config.AgentPort)
	}

	// get registrar address
	host, port, err := common.ParseHostPort(config.Registrar, "127.0.0.1", REGISTRAR_PORT)
	if err != nil {
		return utils.WrapError("Parse registrar address", err)
	}
	registrar := fmt.Sprintf("%s:%d", host, port)

	//create SBI http server
	m.sbiServer = httpw.NewServer(httpw.Options{
		Addr:   fmt.Sprintf("%s:%d", bindIp, sbiBindPort),
		Cert:   opts.Cert,
		CaPool: opts.CaPool,
	})

	//set up routes for SBI services
	for _, g := range opts.App.GetRouteGroups() {
		if len(opts.SbiPrefix) > 0 {
			m.sbiServer.AddRoutes(opts.SbiPrefix+"/"+g.Name, g.Routes)
		} else {
			m.sbiServer.AddRoutes(g.Name, g.Routes)
		}
	}

	//parse registered SBI URL
	host, port, err = common.ParseHostPort(config.RegisteredSbiUrl, bindIp, fmt.Sprintf("%d", sbiBindPort))
	if err != nil {
		return utils.WrapError("Parse registered Sbi URL", err)
	}

	if host == "0.0.0.0" {
		if localIp, err := common.GetPrimaryIP(); err != nil {
			return utils.WrapError("Get Primary IP", err)
		} else {
			host = localIp.String()
		}

	}

	sbiUrl := fmt.Sprintf("%s:%d", host, port)
	agentUrl := fmt.Sprintf("%s:%s", host, agentBindPort)

	//create registry
	m.agent = registry.CreateRegistry(registry.Options{
		SbiUrl:      sbiUrl,
		AgentUrl:    agentUrl,
		Registrar:   registrar,
		Cert:        opts.Cert,
		CaPool:      opts.CaPool,
		CertName:    opts.CertName,
		GatewayName: opts.GatewayName,
		Labels:      config.Labels,
		OnEpLeft:    m.onEpLeft,
		OnEpJoin:    m.onEpJoin,
	})

	//add OAM routes to sbi server if the application supports OAM APIs
	if g, ok := opts.App.(HasOamBackend); ok {
		agentOamHandlerContext := m.agent.CreateOamHandlerContext()
		oamHandler := g.CreateOamHandler(agentOamHandlerContext)
		m.sbiServer.AddRoutes("oam", oamHandler.Routes())
	} else {
		oamHandler := m.agent.CreateOamHandler()
		m.sbiServer.AddRoutes("oam", oamHandler.Routes())
	}

	//create agent http server
	m.agentServer = httpw.NewServer(httpw.Options{
		Addr:   fmt.Sprintf("%s:%s", bindIp, agentBindPort),
		Cert:   opts.Cert,
		CaPool: opts.CaPool,
		Routes: m.agent.HttpRoutes(),
	})

	// set the singleton mesh
	_mesh = m

	//start all services
	return m.start(opts)
}

//run all services; do registration and service subscribe
func (m *b5gcMesh) start(opts *Options) error {
	//register services to execute
	services := []Running{}
	services = append(services, m.sbiServer)
	services = append(services, m.agentServer)

	//execute service sequentially
	for _, service := range services {
		if err := service.Start(); err != nil {
			m.stop()
			return utils.WrapError("Start a service", err)
		}
		m.runnings = append(m.runnings, service)
	}

	// register to the mesh controller
	var configBytes []byte
	if f, ok := opts.App.(HasSharedConfig); ok {
		configBytes = f.GetSharedConfig()
	}

	if err := m.agent.Register(opts.SbiPrefix, configBytes, opts.SbiPending, m.quit); err != nil {
		return utils.WrapError("Register to the mesh controller", err)
	}

	//subscribe to services
	if len(opts.SubscribedServices) > 0 {
		if err := m.agent.Subscribe(opts.SubscribedServices); err != nil {
			return utils.WrapError("Subscribe services", err)
		}
	}
	return nil
}

func (m *b5gcMesh) stop() error {
	m.quit <- struct{}{} //stop registration if it is running
	//stop all running services
	for _, service := range m.runnings {
		service.Stop()
	}
	if err := m.agent.Deregister(); err != nil {
		return utils.WrapError("Deregister", err)
	}

	return nil
}

func (m *b5gcMesh) onEpLeft(uuid string) {
	if m.evCh != nil {
		m.evCh <- EventInfo{
			EvType:  EV_EP_LEFT,
			Content: uuid,
		}
	}
}

func (m *b5gcMesh) onEpJoin(ep models.EndpointInfo, services []string, dat []byte) {
	if m.evCh != nil {
		m.evCh <- EventInfo{
			EvType: EV_EP_JOIN,
			Content: &EpJoinEventInfo{
				Endpoint: ep,
				Services: services,
				Config:   dat,
			},
		}
	}
}
