package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"etrib5gc/nfs/app"
	"etrib5gc/nfs/smf/config"
	"etrib5gc/nfs/smf/context"
	"etrib5gc/nfs/smf/dataplane"
	backend "etrib5gc/nfs/smf/oam-backend"
	"etrib5gc/nfs/smf/sbi/producer"
	"etrib5gc/nfs/smf/sm"
	"etrib5gc/nfs/smf/topo"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"
	"github.com/reogac/utils/oam"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
)

var log *logrus.Entry

type SMF struct {
	evCh    chan mesh.EventInfo //receiv joining endpoint from service mesh
	quit    chan struct{}
	running bool //flag indicated if the service is initiated
	wg      sync.WaitGroup
	expFile string
}

func (nf *SMF) Init(ctx *app.Context) error {

	log = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	var cfg config.Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.expFile = ctx.String("exp")

	nf.evCh = make(chan mesh.EventInfo, 128)
	nf.quit = make(chan struct{})

	if err := context.InitContext(&cfg); err != nil {
		return utils.WrapError("Init SMF context", err)
	}

	//create topology manager
	topo.Init()

	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		SbiPending:         true,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
		EvCh:               nf.evCh,
	})
	return nil
}

func (nf *SMF) Run() (err error) {

	if err = context.GetSessionManagementConfiguration(); err != nil {
		return utils.WrapError("Get Session Management Configuration", err)
	}

	//mark the service as initiated
	nf.running = true
	//create SmContext pool
	sm.InitSmContextPool()

	//create Pfcp session pool
	dataplane.InitPfcpSessionPool()

	//create a go routing to receive joining endpoint event from service mesh
	//(to manage UPF topology)
	nf.wg.Add(1)
	go func() {
		defer nf.wg.Done()
	LOOP:
		for {
			select {
			case <-nf.quit:
				break LOOP
			case ev := <-nf.evCh:
				switch ev.EvType {
				case mesh.EV_EP_LEFT:
					id, _ := ev.Content.(string)
					nf.onEpLeft(id)
				case mesh.EV_EP_JOIN:
					info, _ := ev.Content.(*mesh.EpJoinEventInfo)
					nf.onEpJoin(info)
				}
			}
		}
	}()

	//activate sbi server to start handing service requests
	return mesh.ActivateSbi("")
}

func (nf *SMF) Stop() {
	//releasing resources if the service was initiated
	if nf.running {
		//log experimental results
		if len(nf.expFile) > 0 {
			if expFile, err := os.OpenFile(nf.expFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
				log.Errorf("Failed to create logfile " + nf.expFile)
			} else {
				sm.LogExpResults(expFile)
				expFile.Close()
			}

		}

		close(nf.quit)

		log.Infof("Clean SmContext pool")
		sm.CleanSmContextPool()

		log.Infof("Clean PfcpSession pool")
		dataplane.CleanPfcpSessionPool()
		nf.wg.Wait()
	}
}

func (nf *SMF) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *SMF) subscribedServices() []string {
	plmnId := context.PlmnId()
	return []string{
		common.UpfServiceName(plmnId),
		common.UdmServiceName(plmnId),
		common.PcfServiceName(plmnId),
	}
}

func (nf *SMF) CreateOamHandler(ext *oam.HandlerContext) *oam.OamHandler {
	return backend.CreateOamHandler(ext)
}

func (nf *SMF) onEpLeft(uuid string) {
	topo.HandleUpfLeft(uuid)
}

func (nf *SMF) onEpJoin(info *mesh.EpJoinEventInfo) {
	//ep models.EndpointInfo, services []string, dat []byte
	upfServiceId := common.UpfServiceName(context.PlmnId())
	for _, s := range info.Services {
		if upfServiceId == s {
			topo.HandleUpfJoin(&info.Endpoint, info.Config)
			break
		}
	}
}
