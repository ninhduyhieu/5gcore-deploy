package controller

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh/controller/monitoring"
	"etrib5gc/mesh/controller/oambackend"
	"etrib5gc/mesh/lbs"
	"etrib5gc/mesh/models"
	"etrib5gc/nfs/app"
	"fmt"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	EP_LEFT uint8 = iota
	EP_JOIN
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

var log *logrus.Entry

type Controller struct {
	server      *httpw.Server
	evch        chan epEvent
	gwman       *GatewayManager  //gateway manager
	epman       *EndpointManager //endpoint manager
	sman        *ServiceManager  //service manager
	groupList   GroupList        //service group definitions
	routingList RoutingList      //service routing definitions

	wg   sync.WaitGroup
	quit chan bool

	cert   *tls.Certificate
	caPool *x509.CertPool

	k8sCli     *kubernetes.Clientset
	svcWatcher watch.Interface

	k8sNameSpace string
}

type epEvent struct {
	evtype uint8
	dat    interface{}
}

func (ctrl *Controller) Init(ctx *app.Context) error {
	//init log
	log = logctx.Entry(nil)

	var cfg Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	ctrl.evch = make(chan epEvent, 256)
	ctrl.quit = make(chan bool)

	ctrl.epman = newEndpointManager(ctrl)
	ctrl.gwman = newGatewayManager(ctrl)

	if ctrl.k8sNameSpace = os.Getenv("POD_NAMESPACE"); len(ctrl.k8sNameSpace) > 0 { //in K8s
		if k8sConfig, err := rest.InClusterConfig(); err != nil {
			return utils.WrapError("Get K8s client configuration", err)
		} else {
			if ctrl.k8sCli, err = kubernetes.NewForConfig(k8sConfig); err != nil {
				return utils.WrapError("Create client set", err)
			}
		}
		if err := ctrl.loadK8sServices(); err != nil {
			return utils.WrapError("load K8s services", err)
		}
	}

	//add services
	ctrl.parseServices(cfg.Services)
	//add service groups
	ctrl.parseGroupings(cfg.Groupings)

	//add service routes
	ctrl.parseRoutings(cfg.Routings)

	//keep cert information
	if ctx.CaPool != nil {
		ctrl.cert, ctrl.caPool = &ctx.Cert, ctx.CaPool
	}

	bindIp, bindPort, err := common.ParseHostPort(cfg.Bind, "0.0.0.0", CONTROLLER_PORT)
	if err != nil {
		return utils.WrapError("Parse binding address", err)
	}

	//create http server
	ctrl.server = httpw.NewServer(httpw.Options{
		Addr:   fmt.Sprintf("%s:%d", bindIp, bindPort),
		Routes: ctrl.makeHttpRoutes(),
		Cert:   ctx.Cert,
		CaPool: ctx.CaPool,
	})
	if len(ctx.CertName) > 0 {
		log.Infof("Controller certificate's subject CN: %s", ctx.CertName)
	}

	//monitoring routes
	monServ := monitoring.New(&oamApi{
		ctrl: ctrl,
	})

	ctrl.server.AddRoutes("mon", monServ.MakeHttpRoutes())
	oamHandler := oambackend.NewOamHandler(&oamApi{
		ctrl: ctrl,
	})
	ctrl.server.AddRoutes("oam", oamHandler.Routes())
	return nil
}

func (c *Controller) makeHttpRoutes() []httpw.Route {
	//backend for monitoring API
	return []httpw.Route{
		httpw.Route{ //gateway registers
			Method:  http.MethodPost,
			Pattern: "/gw",
			Handler: c.onGwRegister,
		},
		httpw.Route{ //gateway deregisters
			Method:  http.MethodPost,
			Pattern: "/gw/del/:gwId",
			Handler: c.onGwDeregister,
		},
		httpw.Route{ //gateway get peer info
			Method:  http.MethodGet,
			Pattern: "/gw-info/:peerId",
			Handler: c.onGwGetPeer,
		},

		httpw.Route{ //endpoint subscribes
			Method:  http.MethodPost,
			Pattern: "/subscribe/:epId",
			Handler: c.onSubscribe,
		},
		httpw.Route{ //endpoint subscribes
			Method:  http.MethodGet,
			Pattern: "/get/:epId",
			Handler: c.onGetEndpoint,
		},

		httpw.Route{ //endpoint activates
			Method:  http.MethodPost,
			Pattern: "/activate/:epId",
			Handler: c.onActivate,
		},
		httpw.Route{ //endpoint deregister
			Method:  http.MethodDelete,
			Pattern: "/delete/:epId",
			Handler: c.onDelete,
		},
		httpw.Route{ //endpoint register
			Method:  http.MethodPost,
			Pattern: "/register/:gwId",
			Handler: c.onEpRegister,
		},
		httpw.Route{ //add service
			Method:  http.MethodPost,
			Pattern: "/service/add",
			Handler: c.onAddService,
		},
	}
}

func (c *Controller) parseServices(services []ServiceConfig) {
	c.sman = newServiceManager(c.epman)
	for _, item := range services {
		s := newService(item)
		c.sman.addService(s)
		log.Infof("Add service: %s[%s]", s.name, s.selectors.String())
	}
}

func (c *Controller) parseGroupings(groupings []GroupingConfig) {
	c.groupList = newGroupList()

	for _, grouping := range groupings {
		service := grouping.Service
		for _, group := range grouping.Groups {
			g := &ServiceGroup{
				service:   service,
				name:      group.Id,
				selectors: group.Selectors,
			}
			c.groupList.addGroup(g)
			log.Infof("Add group: %s-%s-%s", service, group.Id, group.Selectors.String())
		}
	}

}

func (c *Controller) parseRoutings(routings []RoutingConfig) {
	c.routingList = newRoutingList()

	for _, routing := range routings {
		for _, r := range routing.Routes {
			//build a route
			route := &Route{
				service: routing.Service,
				match:   make(map[string]string),
			}

			for k, v := range r.Match {
				route.match[k] = v
			}

			//add route destinations
			for _, d := range r.Destinations {
				if c.checkValidGroupId(routing.Service, d.Group) {
					dst := &Destination{
						groupId: d.Group,
						weight:  d.Weight,
						lb:      lbs.ParseLoadBalancerType(d.Lb),
					}
					if dst.lb == lbs.LB_TYPE_UNKNOWN {
						log.Warnf("No supported load balancer: %s, set to random load balancer", d.Lb)
						dst.lb = DEFAULT_LB
					}
					route.targets = append(route.targets, *dst)
				}
			}
			c.routingList.addRoute(route)
			log.Infof("Add route: %s-%s", route.service, route.idStr)
		}
	}
}

func (c *Controller) checkValidGroupId(service, groupNam string) bool {
	//TODO: do the check
	return true
}

func (c *Controller) Run() (err error) {
	//start http server
	if err = c.server.Start(); err != nil {
		return
	}
	log.Infof("Controller listen to %s", c.server.Srv.Addr)
	//start event loop
	c.wg.Add(1)
	go c.epEventLoop()
	return
}

func (c *Controller) Stop() {
	log.Info("Controller is terminating")
	if c.svcWatcher != nil {
		c.svcWatcher.Stop()
	}
	c.server.Stop()
	c.gwman.close()
	close(c.quit)
	close(c.evch)

	//	c.db.clean()
	c.wg.Wait()
}
func (c *Controller) sendEpEvent(evtype uint8, ep *Endpoint) {
	c.evch <- epEvent{
		evtype: evtype,
		dat:    ep,
	}
}

func (c *Controller) onEpRegister(ctx *gin.Context) {
	log.Debugf("Receivce an endpoint registration request")
	uuid := ctx.Param("gwId")
	var gw *Gateway
	if gw = c.gwman.findByUuid(uuid); gw == nil {
		ctx.String(http.StatusNotFound, "Unknown gateway")
		return
	}
	var err error
	var dat []byte
	var msg models.RegistrationFwRequest
	//parse message
	if dat, err = ioutil.ReadAll(ctx.Request.Body); err == nil {
		if err = json.Unmarshal(dat, &msg); err == nil {
			//register the  endpoint
			var rsp *models.RegistrationResponse
			if rsp, err = c.epman.registerEndpoint(&msg, gw); err == nil {
				ctx.JSON(http.StatusCreated, *rsp)
				return
			}
		}
	}
	ctx.String(http.StatusInternalServerError, err.Error())
}

func (c *Controller) onAddService(ctx *gin.Context) {
	log.Debugf("Receive a service adding request")
	var err error
	var dat []byte
	var msg models.AddServiceRequest
	//parse message
	if dat, err = ioutil.ReadAll(ctx.Request.Body); err == nil {
		if err = json.Unmarshal(dat, &msg); err == nil {
			//register the  endpoint
			if addedServices := c.sman.handleUpdateService(msg); len(addedServices) > 0 {
				ctx.JSON(http.StatusCreated, "OK")
				for _, s := range addedServices {
					log.Infof("Service %s is added", s)
				}
				return
			}
			ctx.JSON(http.StatusInternalServerError, "Fail to add services")
			return
		}
	}
	ctx.String(http.StatusInternalServerError, fmt.Sprintf("Fail to read request body: %+v", err))
}

func (c *Controller) onGwRegister(ctx *gin.Context) {
	log.Debugf("Receive a GW registration request")
	var err error
	var dat []byte
	var msg models.GwRegistrationRequest
	//parse message
	if dat, err = ioutil.ReadAll(ctx.Request.Body); err == nil {
		if err = json.Unmarshal(dat, &msg); err == nil {
			//register the gateway
			var rsp *models.GwRegistrationResponse
			if rsp, err = c.gwman.registerGw(&msg); err == nil {
				ctx.JSON(http.StatusCreated, *rsp)
				return
			}
		}
	}
	ctx.String(http.StatusInternalServerError, err.Error())
}

func (c *Controller) onGwDeregister(ctx *gin.Context) {
	log.Debugf("Receive a GW deregistration request")
	gwId := ctx.Param("gwId")
	if gw := c.gwman.findByUuid(gwId); gw == nil {
		log.Errorf("gateway not found")
		ctx.String(http.StatusNotFound, fmt.Sprintf("gateway %s not found", gwId))
	} else {
		gw.close()
		ctx.String(http.StatusCreated, "OK")
	}
}

func (c *Controller) onGwGetPeer(ctx *gin.Context) {
	log.Debugf("Receive a GW get peer information request")
	gwId := ctx.Param("peerId")
	if gw := c.gwman.findByUuid(gwId); gw == nil {
		ctx.String(http.StatusNotFound, fmt.Sprintf("Gateway %s not found", gwId))
	} else {
		info := models.GatewayInfo{
			Id:       gwId,
			Url:      gw.url,
			CertName: gw.name,
		}
		ctx.JSON(http.StatusOK, &info)
	}
}

func (c *Controller) onSubscribe(ctx *gin.Context) {
	log.Debugf("Receive an endpoint service subscription request")
	var err error
	var dat []byte
	epId := ctx.Param("epId")
	var msg models.SubscribeRequest
	//parse message
	if dat, err = ioutil.ReadAll(ctx.Request.Body); err == nil {
		if err = json.Unmarshal(dat, &msg); err == nil {
			//add subscibed services for endpoint
			var rsp *models.SubscribeResponse
			if rsp, err = c.endpointSubscribe(epId, &msg); err == nil {
				ctx.JSON(http.StatusCreated, rsp)
				return
			}
		}
	}
	ctx.String(http.StatusInternalServerError, err.Error())
}

func (c *Controller) onActivate(ctx *gin.Context) {
	log.Debugf("Receive an endpoint activation update")
	epId := ctx.Param("epId")
	var err error
	var dat []byte
	var msg models.UpdateNotification
	//parse message
	if dat, err = ioutil.ReadAll(ctx.Request.Body); err == nil {
		if err = json.Unmarshal(dat, &msg); err == nil {
			if err = c.epman.activateEndpoint(epId, &msg); err == nil {
				ctx.JSON(http.StatusCreated, nil)
				return
			} else {
				err = fmt.Errorf("Can't add subscriber")
			}
		}
	}
	ctx.String(http.StatusInternalServerError, err.Error())
}

func (c *Controller) onGetEndpoint(ctx *gin.Context) {
	log.Debugf("Receive a get endpoint request")
	epId := ctx.Param("epId")
	if ep := c.epman.findByUuid(epId); ep == nil {
		ctx.JSON(http.StatusNotFound, fmt.Sprintf("Endpoint not found for %s", epId))
	} else {
		msg := ep.toModel()
		ctx.JSON(http.StatusOK, &msg)
	}
}

func (c *Controller) onDelete(ctx *gin.Context) {
	log.Debugf("Receive an endpoint deletion request")
	epId := ctx.Param("epId")
	if err := c.deleteEndpoint(epId); err == nil {
		ctx.Status(http.StatusCreated)
	} else {
		ctx.String(http.StatusInternalServerError, err.Error())
	}
}

func (c *Controller) epEventLoop() {
	defer c.wg.Done()
	for ev := range c.evch {
		switch ev.evtype {
		case EP_LEFT: // an Endpoint was removed
			if ep, ok := ev.dat.(*Endpoint); ok {
				c.onEpLeft(ep)
			}
		case EP_JOIN:
			//an endpoint joined/activated
			if ep, ok := ev.dat.(*Endpoint); ok {
				c.onEpJoin(ep)
			}
		}
	}
}

func (c *Controller) onEpLeft(ep *Endpoint) {
	ep.Debugf("Receive an endpoint left event")
	//notify of endpoint leaving
	//TODO: use a worker pool
	subs := c.sman.getSubscribers(ep.servings)
	for _, sub := range subs {
		sub.notifyEpLeft(ep)
	}
}

func (c *Controller) onEpJoin(ep *Endpoint) {
	ep.Debugf("Receive an endpoint joined event")
	//notify of endpoint joining
	subs := c.sman.getSubscribers(ep.servings)
	//TODO: use a worker pool
	for _, sub := range subs {
		sub.notifyEpJoin(ep)
	}
}

// endpoint subscribes to services
func (c *Controller) endpointSubscribe(uuid string, msg *models.SubscribeRequest) (dat *models.SubscribeResponse, err error) {
	//find the endpoint with its identity
	var ep *Endpoint
	if ep = c.epman.findByUuid(uuid); ep == nil {
		log.Errorf("Endpoint %s not found", uuid)
		err = fmt.Errorf("Endpoint %s not found", uuid)
		return
	}
	//subscribe services for the endpoint
	services := c.sman.addSubscriber(ep, msg.Services)

	// compose subscription data for the subsribed endpoint
	dat = &models.SubscribeResponse{}
	dat.Endpoints = make(map[string]models.Endpoint)
	dat.Services = make(map[string]models.Service)
	//get relevant endpoints
	endpoints := c.epman.getEndpoints(services)
	//add relevant endpoints
	for _, item := range endpoints {
		dat.Endpoints[item.id] = item.toModel()
	}

	//add service definitions
	for _, s := range services {
		dat.Services[s.name] = models.Service{
			Id:        s.name,
			Stateful:  s.stateful,
			Selectors: models.Selectors(s.selectors),
			Groups:    c.groupList.buildGroupInfo(s.name),
			Routes:    c.routingList.buildRouteInfo(s.name),
		}
	}
	return
}

func (c *Controller) deleteEndpoint(uuid string) (err error) {
	if ep := c.epman.findByUuid(uuid); ep == nil {
		log.Errorf("Endpoint %s not found", uuid)
		err = fmt.Errorf("Endpoint %s not found", uuid)
	} else {
		c.epman.removeEp(ep)
	}

	return
}
