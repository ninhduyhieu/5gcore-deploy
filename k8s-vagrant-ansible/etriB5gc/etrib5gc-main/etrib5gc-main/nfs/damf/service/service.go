package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/utils/httpw"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/damf/config"
	"etrib5gc/nfs/damf/context"
	"etrib5gc/nfs/damf/nssf"
	"etrib5gc/nfs/damf/ue"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/damf/sbi/producer"
	"github.com/reogac/sbi/apis/nsm/conf"
)

type DAMF struct {
	*logrus.Entry
}

func (nf *DAMF) Init(ctx *app.Context) error {
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
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
	})
	return nil
}
func (nf *DAMF) Run() error {
	if err := nf.initNssf(); err != nil {
		return utils.WrapError("Init NSSF configuration", err)
	}

	ue.InitUePool()
	return nil
}

func (nf *DAMF) Stop() {
	ue.CleanUePool()
}

func (nf *DAMF) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *DAMF) subscribedServices() []string {
	plmnId := context.PlmnId()
	return []string{
		common.AmfServiceName(plmnId, ""),
		common.UdmServiceName(plmnId),
		common.NsmServiceName(plmnId),
		common.AusfServiceName(plmnId),
	}
}

func (nf *DAMF) initNssf() (err error) {
	//create client to NSM
	nsmId := common.NsmServiceName(context.PlmnId())
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		err = utils.WrapError("Create NSM client", err)
		return
	}

	var cfg *models.NssfConfiguration
	if cfg, err = conf.GetNssfConfiguration(nsmCli); err != nil {
		err = utils.WrapError("Get Nssf configuration", err)
		return
	} else {
		nf.Infof("Receive Nssf configuration from NSM")
		nssf.Init(cfg)
	}

	return
}
