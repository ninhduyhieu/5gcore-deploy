package context

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/nfs/nsm/config"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/idgen"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

type NsmContext struct {
	plmnid            models.PlmnId
	dataNetworks      map[string]*DataNetwork
	transportNetworks []string
	slices            map[string]Slice
	amfSets           map[string]*AmfSet
	endpoints         map[string]Endpoint
	udr               models.UdrConfiguration
	suciProfiles      []models.SuciProfile
	plmnPeers         map[models.PlmnId]PlmnPeer
}

type Endpoint struct {
	left func(string)
}

var _ctx *NsmContext //singleton context
var log *logrus.Entry

func Init(cfg *config.Config) {
	if _ctx != nil {
		return
	}
	log = logctx.Entry(logrus.Fields{
		"mod": "context",
	})

	_ctx = &NsmContext{
		plmnid:       cfg.PlmnId,
		slices:       make(map[string]Slice),
		plmnPeers:    make(map[models.PlmnId]PlmnPeer),
		endpoints:    make(map[string]Endpoint),
		udr:          cfg.Udr,
		suciProfiles: cfg.SuciProfiles,
	}
	//get transport networks
	_ctx.transportNetworks = lo.Uniq(cfg.TransportNetworks)

	//parse deployed data networks
	_ctx.parseDataNetworks(cfg.DataNetworks)

	//parse deployed slices
	for _, slice := range cfg.Slices {
		sliceId := models.SnssaiToString(slice.Id)
		if _, ok := _ctx.slices[sliceId]; ok {
			continue
		}
		netList := []string{}
		for _, dnn := range slice.DataNetworks {
			if dn, ok := _ctx.dataNetworks[dnn]; ok {
				netList = append(netList, dnn)
				dn.slices = append(dn.slices, slice.Id)
			}
		}
		_ctx.slices[sliceId] = Slice{
			id:           slice.Id,
			dataNetworks: common.UniqueArray[string](netList),
		}
	}
	//parse HPlmn Peers
	for _, plmn := range cfg.PlmnPeers {
		_ctx.plmnPeers[plmn.PlmnId] = _ctx.createPlmn(&plmn)
	}

	//parse amf set definitions
	_ctx.parseAmfSets(cfg.AmfSets)
}

func (ctx *NsmContext) parseDataNetworks(items []config.DataNetworkConfig) {
	ctx.dataNetworks = make(map[string]*DataNetwork)
	for _, item := range items {
		if len(item.Name) == 0 {
			continue
		}
		if _, ok := ctx.dataNetworks[item.Name]; ok {
			continue
		}
		if dn := ctx.createDataNetwork(item); dn != nil {
			ctx.dataNetworks[item.Name] = dn
		}
	}
}

func (ctx *NsmContext) parseAmfSets(items []config.AmfSetConfig) {
	ctx.amfSets = make(map[string]*AmfSet)
	for _, item := range items {
		if _, err := common.AmfSetFromString(item.SetId); err != nil { //check format (AmfRegion + AmfSet)
			log.Warnf("AmfSet %s is invalid", item.SetId)
			continue
		}
		amfSet := &AmfSet{
			id:         item.SetId,
			slices:     make(map[string]models.Snssai),
			pointerGen: idgen.NewGenerator[uint8](0, MAX_6_BITS),
			amfList:    make(map[string]*Amf),
		}
		for _, slice := range item.Slices {
			amfSet.slices[slice.String()] = slice
		}
		ctx.amfSets[amfSet.id] = amfSet
	}

}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnid
}

func AmfRegister(req *models.AmfRegistrationRequest) (rsp *models.AmfRegistrationResponse, err error) {
	if amfSet, ok := _ctx.amfSets[req.AmfSet]; !ok {
		err = fmt.Errorf("AmfSet not found: %s", req.AmfSet)
	} else if len(req.Uuid) == 0 {
		err = fmt.Errorf("Uuid empty")
	} else {
		var amf *Amf
		var ok bool
		if amf, ok = amfSet.amfList[req.Uuid]; !ok {
			amf = &Amf{
				uuid:    req.Uuid,
				setId:   req.AmfSet,
				pointer: amfSet.pointerGen.Allocate(),
			}
			amfSet.amfList[req.Uuid] = amf
			_ctx.endpoints[req.Uuid] = Endpoint{
				left: amfSet.removeAmf,
			}
			log.Infof("AMF %s-%d registered", amf.setId, amf.pointer)
		}
		rsp = &models.AmfRegistrationResponse{
			AmfPointer: int16(amf.pointer),
			Slices:     amfSet.getSlices(),
		}
		//add home plmn
		for _, item := range _ctx.plmnPeers {
			rsp.PlmnPeers = append(rsp.PlmnPeers, item.toConfig())
		}

	}
	return
}

func HandleEpLeft(uuid string) {
	log.Debugf("Endpoint %s left", uuid)
	if ep, ok := _ctx.endpoints[uuid]; ok {
		ep.left(uuid)
	}
}

func GetSupportedSlices(amfRegion string) (slices []models.Snssai) {
	uniqList := make(map[string]models.Snssai)
	for _, item := range _ctx.amfSets {
		if item.inRegion(amfRegion) {
			for id, s := range item.slices {
				uniqList[id] = s
			}
		}
	}
	for _, s := range uniqList {
		slices = append(slices, s)
	}
	return
}

func GetNssfConfiguration() *models.NssfConfiguration {
	config := &models.NssfConfiguration{}

	// add slices
	for _, item := range _ctx.slices {
		config.Slices = append(config.Slices, models.SliceConfiguration{
			Id:      item.id,
			DnnList: item.dataNetworks,
		})
	}

	//add home plmn
	for _, item := range _ctx.plmnPeers {
		config.PlmnPeers = append(config.PlmnPeers, item.toConfig())
	}

	for _, item := range _ctx.amfSets {
		config.AmfSets = append(config.AmfSets, item.createConfigModel())
	}
	if len(config.AmfSets) > 0 {
		return config
	}
	return nil
}

func GetSessionManagementConfiguration(smfId string, slice *models.Snssai) *models.SessionManagementConfiguration {
	cfg := &models.SessionManagementConfiguration{}
	//fill transport networks
	for _, tn := range _ctx.transportNetworks {
		cfg.TransportNetworks = append(cfg.TransportNetworks, tn)
	}

	//fill data networks
	for _, dn := range _ctx.dataNetworks {
		if dn.hasSlice(slice) {
			cfg.DataNetworks = append(cfg.DataNetworks, dn.buildConfigurationModel(smfId))
		}
	}

	if len(cfg.DataNetworks) > 0 {
		_ctx.endpoints[smfId] = Endpoint{
			left: _ctx.smfLeft,
		}
		log.Infof("Build session management information for SMF %s", smfId)
		return cfg
	}
	return nil
}

func (ctx *NsmContext) smfLeft(uuid string) {
	log.Warnf("SMF removed:%s", uuid)
	for _, dn := range ctx.dataNetworks {
		dn.removePoolOwner(uuid)
	}
}
func GetSeppConfiguration() *models.SeppConfiguration {
	info := new(models.SeppConfiguration)
	for _, item := range _ctx.plmnPeers {
		info.PlmnList = append(info.PlmnList, item.toConfig())
	}
	return info
	return nil
}

func GetUdrConfiguration() *models.UdrConfiguration {
	return &_ctx.udr
}

func GetUdmConfiguration() *models.UdmConfiguration {
	return &models.UdmConfiguration{
		Udr:          _ctx.udr,
		SuciProfiles: _ctx.suciProfiles,
	}
}

func GetUserPlaneConfiguration(slices []models.Snssai) *models.UserPlaneConfigurationResponse {
	cfg := &models.UserPlaneConfigurationResponse{}
	//fill transport networks
	for _, tn := range _ctx.transportNetworks {
		cfg.TransportNetworks = append(cfg.TransportNetworks, tn)
	}

	//fill data networks
	for _, dn := range _ctx.dataNetworks {
		for _, s := range slices {
			if dn.hasSlice(&s) {
				cfg.DataNetworks = append(cfg.DataNetworks, models.DataNetworkInfo{
					Cidr: dn.cidr.String(),
					Name: dn.name,
				})
				break
			}
		}

	}

	if len(cfg.DataNetworks) > 0 {
		return cfg
	}
	return nil
}

func (ctx *NsmContext) hasSlice(s *models.Snssai) bool {
	_, ok := ctx.slices[s.String()]
	return ok
}
