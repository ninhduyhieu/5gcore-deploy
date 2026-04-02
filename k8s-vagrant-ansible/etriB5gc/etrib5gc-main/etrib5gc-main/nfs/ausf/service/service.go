package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"

	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/ausf/auth"
	"etrib5gc/nfs/ausf/config"
	"etrib5gc/nfs/ausf/context"

	"etrib5gc/nfs/ausf/sbi/producer"
)

type AUSF struct {
	*logrus.Entry
}

func (nf *AUSF) Init(ctx *app.Context) error {
	var cfg config.Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.Entry = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	// initialize AUSF context
	context.Init(&cfg)
	auth.CreateDataStore()

	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
	})
	return nil
}

func (nf *AUSF) Run() error {
	//Nothing to do for now
	return nil
}

func (nf *AUSF) Stop() {
	auth.CloseDataStore()
}

func (nf *AUSF) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *AUSF) subscribedServices() []string {
	return []string{
		common.UdmServiceName(context.PlmnId()),
	}
}
