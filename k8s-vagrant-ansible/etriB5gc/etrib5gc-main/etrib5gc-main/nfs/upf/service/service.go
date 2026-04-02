package service

import (
	"encoding/json"
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"github.com/reogac/utils/httpw"
	"net"
	"sync"

	"etrib5gc/nfs/upf/config"
	"etrib5gc/nfs/upf/context"
	"etrib5gc/nfs/upf/forwarder"
	"etrib5gc/nfs/upf/sessions"
	"etrib5gc/nfs/upf/smfman"
	"github.com/reogac/utils"

	"etrib5gc/nfs/app"
	"etrib5gc/nfs/upf/sbi/producer"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/conf"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry

type UPF struct {
	producer *producer.Producer //handling Sbi requests received at the server
	driver   forwarder.Driver
	wg       sync.WaitGroup
}

func (nf *UPF) Init(ctx *app.Context) error {
	log = logctx.Entry(logrus.Fields{
		"mod": "service",
	})

	var cfg config.Config
	var err error
	if err := json.Unmarshal(ctx.ConfigData, &cfg); err != nil {
		return utils.WrapError("Unmarshal config", err)
	}

	//create context
	context.Init(&cfg)

	// get SBI registered IP to be used as Pfcp Node Id
	var n4Ip net.IP
	if n4Ip, err = common.GetPrimaryIP(); err != nil {
		return utils.WrapError("Get Primary IP", err)
	}

	// create driver to connect to kernel-GTP module
	if nf.driver, err = forwarder.NewDriver(&nf.wg, &cfg); err != nil {
		return utils.WrapError("Create GTP driver", err)
	}

	//initialize Pfcp session manager
	sessions.Init(n4Ip, nf.driver)

	//initialize Smf Manager
	smfman.Init()

	ctx.SetMeshOptions(&mesh.Options{
		Config: &cfg.Mesh,
		App:    nf,
	})
	return nil
}

func (nf *UPF) Run() (err error) {
	var cidrList []string
	if cidrList, err = nf.getCidrList(); err != nil {
		return utils.WrapError("Get Data Network Information from NSM", err)
	}
	nf.driver.AddRoutes(cidrList)
	return nil
}

func (nf *UPF) Stop() {
	nf.driver.Close()
	nf.wg.Wait()
}

func (nf *UPF) GetRouteGroups() []httpw.RouteGroup {
	return producer.RouteGroups()
}

func (nf *UPF) getCidrList() (cidrList []string, err error) {
	//create client to NSM
	nsmId := common.NsmServiceName(context.PlmnId())
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		err = utils.WrapError("Create NSM client", err)
		return
	}

	var cfg *models.UserPlaneConfigurationResponse
	log.Infof("Get User Plane Configuration")
	if cfg, err = conf.GetUserPlaneConfiguration(nsmCli, &models.UserPlaneConfigurationRequest{
		Slices: context.Slices(),
	}); err != nil {
		return
	} else {
		for _, dn := range cfg.DataNetworks {
			cidrList = append(cidrList, dn.Cidr)
		}
		log.Infof("Cidr list: %v", cidrList)
	}

	return
}

func (nf *UPF) GetSharedConfig() []byte {
	info := context.UpfInfo()
	buf, _ := json.Marshal(&info)
	return buf
}
