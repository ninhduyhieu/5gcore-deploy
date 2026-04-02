package context

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	"etrib5gc/nfs/udm/config"
	"etrib5gc/util/suci"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/conf"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

type UdmContext struct {
	plmnId       models.PlmnId
	udr          models.UdrConfiguration
	suciProfiles []suci.Profile
}

var _ctx *UdmContext
var log *logrus.Entry

func Init(cfg *config.Config) {
	if _ctx != nil {
		return
	}

	log = logctx.Entry(logrus.Fields{
		"mod": "context",
	})
	_ctx = &UdmContext{
		plmnId: models.PlmnId(cfg.PlmnId),
	}
}

func UdrConfiguration() *models.UdrConfiguration {
	return &_ctx.udr
}

func SuciProfiles() []suci.Profile {
	return _ctx.suciProfiles
}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnId
}

func GetUdmConfiguration() (err error) {
	//create client to NSM
	nsmId := common.NsmServiceName(&_ctx.plmnId)
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		err = utils.WrapError("Create NSM client", err)
		return
	}

	var cfg *models.UdmConfiguration
	if cfg, err = conf.GetUdmConfiguration(nsmCli); err != nil {
		err = utils.WrapError("Request UDM configuration from NSM", err)
		return
	} else {
		//set session manangent info for SMF
		log.Infof("Receive UDM configuration from NSM")
		_ctx.udr = cfg.Udr
		for _, profile := range cfg.SuciProfiles {
			_ctx.suciProfiles = append(_ctx.suciProfiles, suci.Profile{
				ProtectionScheme: int(profile.ProtectionScheme),
				PrivateKey:       profile.PrivateKey,
				PublicKey:        profile.PublicKey,
			})
		}
	}
	return

}
