package sepp

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"etrib5gc/mesh/httpdp"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/conf"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/reogac/utils/httpw"

	"github.com/sirupsen/logrus"

	"etrib5gc/nfs/app"
	"net/http"
)

const (
	SEPP_PORT              int    = 7788
	SEPP_SBI_PREFIX        string = "sepp"
	SEPP_SBI_PREFIX_OFFSET int    = len(SEPP_SBI_PREFIX) + 2
)

var TARGET_ID_HEADER string = httpdp.STICKINESS_HEADER
var CALLBACK_HEADER string = http.CanonicalHeaderKey("Callback")
var SERVICE_ID_HEADER string = httpdp.HOST_HEADER

type SEPP struct {
	*logrus.Entry
	plmnId          models.PlmnId
	srv             *httpw.Server
	url             string //SEPP url
	peers           *PeerList
	remoteEndpoints *RemoteEndpoints
	localEndpoints  *LocalEndpoints
}

func (nf *SEPP) Init(ctx *app.Context) error {
	var cfg Config

	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	nf.Entry = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	// initialize SEPP context
	nf.plmnId = cfg.PlmnId
	nf.url = cfg.Url
	nf.remoteEndpoints = newRemoteEndpoints()
	nf.localEndpoints = newLocalEndpoints()

	ctx.SetMeshOptions(&mesh.Options{
		Config:             &cfg.Mesh,
		App:                nf,
		SubscribedServices: nf.subscribedServices(),
		SbiPrefix:          SEPP_SBI_PREFIX,
	})
	bind := fmt.Sprintf(":%d", SEPP_PORT)
	if len(cfg.Bind) > 0 {
		bind = cfg.Bind
	}
	nf.srv = httpw.NewServer(httpw.Options{
		Addr: bind,
		Routes: []httpw.Route{
			{
				Method:  "ANY",
				Pattern: "/*any",
				Handler: nf.handleIngress,
			},
		},
	})

	return nil
}

func (nf *SEPP) Run() (err error) {
	//create client to NSM
	nsmId := common.NsmServiceName(&nf.plmnId)
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		err = utils.WrapError("Create NSM client", err)
		return
	}

	var cfg *models.SeppConfiguration
	if cfg, err = conf.GetSeppConfiguration(nsmCli); err != nil {
		err = utils.WrapError("Get Sepp configuration", err)
		return
	} else {
		nf.Infof("Receive Sepp configuration from NSM")
		nf.peers = newPeerList(cfg.PlmnList)
	}

	if err = nf.srv.Start(); err != nil {
		utils.WrapError("Start Sepp ingress server", err)
	}
	return
}

func (nf *SEPP) Stop() {
	nf.Infof("Sepp stops")
	nf.srv.Stop()
	//nothing to do for now
}

func (nf *SEPP) GetRouteGroups() []httpw.RouteGroup {
	return []httpw.RouteGroup{
		{
			Routes: []httpw.Route{ //agent reverse proxy
				{
					Method:  "ANY",
					Pattern: "/*any",
					Handler: nf.handleEgress,
				},
			},
		},
	}
}

func (nf *SEPP) subscribedServices() []string {
	return []string{
		common.UdmServiceName(&nf.plmnId),
		common.PcfServiceName(&nf.plmnId),
		common.AusfServiceName(&nf.plmnId),
	}
}
