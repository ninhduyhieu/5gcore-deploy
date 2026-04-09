package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/utils/httpw"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/pcf/config"
	"etrib5gc/nfs/pcf/context"
	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/pcf/sbi/producer"
	"github.com/reogac/utils"
)

type PCF struct {
	*logrus.Entry
}

func (nf *PCF) Init(ctx *app.Context) error {
	var cfg config.Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.Entry = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	//create context
	context.Init(&cfg)

	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		SbiPending:         true,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
	})
	return nil
}

func (nf *PCF) Run() error {
	if err := context.GetUdrConfiguration(); err != nil {
		return utils.WrapError("Get UDR configuration", err)
	}
	mesh.ActivateSbi("")
	return nil
}

func (nf *PCF) Stop() {
	context.Stop()
}

func (nf *PCF) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *PCF) subscribedServices() []string {
	return []string{
		common.UdmServiceName(context.PlmnId()),
	}
}
