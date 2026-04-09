package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/pran/config"
	"etrib5gc/nfs/pran/context"
	"etrib5gc/nfs/pran/ran"
	"etrib5gc/nfs/pran/ue"

	"etrib5gc/nfs/pran/ngap"
	"github.com/reogac/sbi"

	"etrib5gc/nfs/pran/sbi/producer"
	"github.com/reogac/utils"
	"strings"
)

const (
	NGAP_PORT int = 38412
)

type PRAN struct {
	*logrus.Entry
	ngap *ngap.Server //ngap server handling Ran connections and ngap messages
}

func (nf *PRAN) Init(ctx *app.Context) error {
	var cfg config.Config
	var err error

	if err = json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.Entry = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	// initialize PRAN context
	context.Init(&cfg)

	//create NGAP server

	var binds []string
	var port int
	ipSet := make(map[string]bool)
	for _, ip := range cfg.NgapBinds {
		if len(ip) > 0 {
			ipSet[ip] = true
		}
	}
	if _, ok := ipSet["0.0.0.0"]; !ok {
		for ip, _ := range ipSet {
			binds = append(binds, ip)
		}
	}

	if len(binds) == 0 {
		binds = []string{"0.0.0.0"}
	}

	port = NGAP_PORT
	if cfg.NgapPort != nil {
		port = *cfg.NgapPort
	}

	nf.Infof("NGAP listening IP:%s", strings.Join(binds, ", "))
	nf.ngap = ngap.NewServer(binds, port)

	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
	})
	return nil
}

func (nf *PRAN) Run() error {
	var err error
	// find NSM
	nsmId := common.NsmServiceName(context.PlmnId())
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		return utils.WrapError("Create NSM client", err)
	}

	//get supported plmn list from NSM
	if err = context.GetSupportedSlices(nsmCli); err != nil {
		return utils.WrapError("Get Supported Plmn List from NSM", err)
	}
	//create UePool
	ue.CreateUePool()
	//create RanPool
	ran.CreateRanPool()

	// start ngap server
	if err = nf.ngap.Run(); err != nil {
		ue.CloseUePool()
		return utils.WrapError("Run NGAP server", err)
	}
	return nil
}

func (nf *PRAN) Stop() {
	//stop ngap server
	ue.CloseUePool()
	nf.ngap.Stop()
}

func (nf *PRAN) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *PRAN) subscribedServices() []string {
	plmnId := context.PlmnId()
	return []string{
		common.UdmServiceName(plmnId),
		common.DamfServiceName(plmnId),
		common.AusfServiceName(plmnId),
	}
}
