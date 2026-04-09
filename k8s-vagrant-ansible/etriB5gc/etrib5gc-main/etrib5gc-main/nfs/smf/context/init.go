package context

import (
	"etrib5gc/common"
	"etrib5gc/mesh"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/nsm/conf"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func GetSessionManagementConfiguration() (err error) {
	//create client to NSM
	nsmId := common.NsmServiceName(&_ctx.plmnId)
	var nsmCli sbi.ConsumerClient
	if nsmCli, err = mesh.Consumer(nsmId, nil); err != nil {
		err = utils.WrapError("Create NSM client", err)
		return
	}

	var cfg *models.SessionManagementConfiguration
	uuid := mesh.EndpointInfo().Uuid
	_ctx.Infof("Get session management config for %s[%v]", uuid, _ctx.snssai)
	if cfg, err = conf.GetSessionManagementConfiguration(nsmCli, conf.GetSessionManagementConfigurationParams{
		Slice: &_ctx.snssai,
		Uuid:  uuid,
	}); err != nil {
		err = utils.WrapError("Request session management configuration from NSM", err)
		return
	} else {
		//set session manangent info for SMF
		_ctx.Infof("Received session management configuration: %+v", *cfg)
		_ctx.transportNetworks = common.UniqueArray[string](cfg.TransportNetworks)
		for _, dn := range cfg.DataNetworks {
			_ctx.addDataNetwork(dn)
		}
	}
	return
}
