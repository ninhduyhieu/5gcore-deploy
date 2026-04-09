package context

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"etrib5gc/nfs/pcf/config"
	"etrib5gc/nfs/udr"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/conf"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

type PcfContext struct {
	udr    *udr.BackendDb
	plmnId models.PlmnId
}

var _ctx *PcfContext
var log *logrus.Entry

func Init(conf *config.Config) {
	if _ctx != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "context",
	})
	_ctx = &PcfContext{
		plmnId: models.PlmnId(conf.PlmnId),
	}
}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnId
}

func GetUdrConfiguration() (err error) {
	//create client to NSM
	nsmId := common.NsmServiceName(&_ctx.plmnId)
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		err = utils.WrapError("Create NSM client", err)
		return
	}

	var cfg *models.UdrConfiguration
	if cfg, err = conf.GetUdrConfiguration(nsmCli); err != nil {
		err = utils.WrapError("Request UDR configuration from NSM", err)
		return
	} else {
		//set session manangent info for SMF
		log.Infof("receive UDR configuration: %+v", *cfg)
		_ctx.udr = udr.New(cfg)
		if err = _ctx.udr.Connect(false); err == nil {
		}
	}
	return
}
func Stop() {
	if _ctx != nil && _ctx.udr != nil {
		_ctx.udr.Close()
	}
}
func Udr() *udr.BackendDb {
	return _ctx.udr
}
