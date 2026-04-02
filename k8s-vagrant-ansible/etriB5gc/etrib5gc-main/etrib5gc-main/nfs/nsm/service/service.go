package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/utils/httpw"
	"sync"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/nsm/config"
	"etrib5gc/nfs/nsm/context"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/nsm/sbi/producer"
)

type NSM struct {
	*logrus.Entry
	cfg  *config.Config      // loaded NSM config
	evCh chan mesh.EventInfo //receive endpoint update events
	quit chan struct{}
	wg   sync.WaitGroup
}

func (nf *NSM) Init(ctx *app.Context) error {
	var cfg config.Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.Entry = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	nf.cfg = &cfg
	nf.evCh = make(chan mesh.EventInfo, 128)
	nf.quit = make(chan struct{})

	// initialize NSM context
	context.Init(&cfg)

	//create mesh agent
	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		App:                nf,
		EvCh:               nf.evCh,
		SubscribedServices: nf.subscribedServices(),
	})

	return nil
}

func (nf *NSM) Run() (err error) {
	nf.wg.Add(1)
	go func() {
		defer nf.wg.Done()
	LOOP:
		for {
			select {
			case <-nf.quit:
				break LOOP

			case info := <-nf.evCh:
				if info.EvType == mesh.EV_EP_LEFT {
					id, _ := info.Content.(string)
					context.HandleEpLeft(id)
				}
			}
		}

	}()
	return
}

func (nf *NSM) Stop() {
	close(nf.quit)
	nf.wg.Wait()
}

func (nf *NSM) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *NSM) subscribedServices() []string {
	return []string{
		common.AmfServiceName(context.PlmnId(), ""),
		common.SmfServiceName(context.PlmnId(), nil),
	}
}
