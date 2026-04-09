package service

import (
	"encoding/json"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/udm/config"
	"etrib5gc/nfs/udm/context"
	"etrib5gc/nfs/udm/ue"
	"etrib5gc/nfs/udr"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"

	"etrib5gc/nfs/udm/sbi/producer"
)

type UDM struct {
	*logrus.Entry
}

func (nf *UDM) Init(ctx *app.Context) error {
	var cfg config.Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.Entry = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	//init context
	context.Init(&cfg)

	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		SbiPending:         true,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
	})
	return nil
}

func (nf *UDM) Run() error {
	if err := context.GetUdmConfiguration(); err != nil {
		return utils.WrapError("Get Udm Configuration", err)
	}
	db := udr.New(context.UdrConfiguration())
	if err := db.Connect(false); err == nil {
		ue.CreateDataStore(db)
	} else {
		return utils.WrapError("Connect UDR", err)
	}

	return mesh.ActivateSbi("")
}

func (nf *UDM) Stop() {
	ue.CloseDataStore()
}

func (nf *UDM) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *UDM) subscribedServices() []string {
	return []string{}
}
