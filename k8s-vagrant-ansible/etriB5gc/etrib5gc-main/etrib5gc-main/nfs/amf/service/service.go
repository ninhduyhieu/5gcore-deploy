package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/utils/httpw"
	"time"

	"github.com/reogac/sbi"
	"github.com/reogac/utils"
	"github.com/reogac/utils/oam"

	"etrib5gc/nfs/amf/config"
	"etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/oam-backend"
	"etrib5gc/nfs/amf/ranue"
	"etrib5gc/nfs/amf/uecontext"
	"etrib5gc/nfs/app"
	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/amf/sbi/producer"

	"github.com/alitto/pond/v2"
)

var log *logrus.Entry

type AMF struct {
	workers pond.Pool //for executing parallel jobs
	closeCh chan bool
	running bool
}

func (nf *AMF) Init(ctx *app.Context) error {
	log = logctx.Entry(logrus.Fields{
		"mod": "service",
	})
	var cfg config.Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.closeCh = make(chan bool)

	// initialize AMF context
	if err := context.Init(&cfg); err != nil {
		return utils.WrapError("Create AmfContext", err)
	}
	ctx.SetMeshOptions(&mesh.Options{
		SbiPending:         true, //not starting sbi server; wait for retrieving configuration from NSM
		Config:             &cfg.Mesh,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
	})
	return nil
}

func (nf *AMF) Run() error {
	if nsmCli, err := nf.connectNsm(); err != nil {
		return utils.WrapError("Create NSM client", err)
	} else if err := context.RegisterAmf(nsmCli); err != nil {
		return utils.WrapError("Register to NSM", err)
	}

	//mark the service is running
	nf.running = true
	//initialize public worker pool
	nf.workers = pond.NewPool(context.MaxNumUes())
	//initialize UeContext pool
	uecontext.InitUePool(nf.workers)
	//initialize RanUe pool
	ranue.InitRanUePool()
	//start Sbi server
	return mesh.ActivateSbi(common.AmfPointerString(context.AmfPointer()))
}

func (nf *AMF) Stop() {
	close(nf.closeCh) //make sure the connecting NSM go routine terminated

	if nf.running { //clean resources if service is already running
		uecontext.CloseUePool()
		ranue.CloseRanUePool()
		log.Infof("Pool metrics: %d:%d:%d:%d", nf.workers.RunningWorkers(), nf.workers.SubmittedTasks(), nf.workers.DroppedTasks(), nf.workers.SuccessfulTasks())
		nf.workers.StopAndWait()
	}
}

func (nf *AMF) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *AMF) subscribedServices() []string {
	plmnId := context.PlmnId()
	return []string{
		common.NsmServiceName(plmnId),
		common.UdmServiceName(plmnId),
		common.PcfServiceName(plmnId),
		common.AusfServiceName(plmnId),
	}
}

func (nf *AMF) CreateOamHandler(ext *oam.HandlerContext) *oam.OamHandler {
	return backend.CreateOamHandler(ext)
}

// create a connection to the NSM (for later issuing of AMF identity)
func (nf *AMF) connectNsm() (sbi.ConsumerClient, error) {
	cliCh := make(chan sbi.ConsumerClient, 1)
	go func() {
		t := time.NewTicker(time.Second)
		cnt := 0
		loop := true
		var nsmCli sbi.ConsumerClient
		for loop {
			//create client to NSM
			nsmId := common.NsmServiceName(context.PlmnId())
			if tmp, err := mesh.Consumer(nsmId, nil); err != nil {
				log.Warnf("Create NSM client failed: %v", err)
				cnt++
				if cnt >= 3 {
					break
				}
			} else {
				nsmCli = tmp
				break
			}
			select {
			case <-nf.closeCh:
				loop = false
			case <-t.C:
			}
		}
		cliCh <- nsmCli
	}()

	cli := <-cliCh
	if cli == nil {
		return nil, fmt.Errorf("Fail to connect to NSM")
	}
	return cli, nil
}
