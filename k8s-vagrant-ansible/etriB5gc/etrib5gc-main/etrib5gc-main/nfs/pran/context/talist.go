package context

import (
	"etrib5gc/mesh"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/amf/callback"
	"github.com/reogac/sbi/models"
)

func UpdateTAList(tacs map[string][]models.Snssai) {
	for tac, slices := range tacs {
		_cu.tacs[tac] = slices
	}
	//notify AMF
	_cu.Infof("Notify AMFs of updated tracking area list")
	for _, amf := range _cu.subscribedAmfs {
		amf.notifySupportedTAList()
	}
}

func GetSupportedTAList() (list []models.SupportedTAItem) {
	/*
		for _, item := range _cu.supportedTAList {
			list = append(list, item.ToModel())
		}
	*/
	return
}

func SubscribeAmf(callback *models.EndpointInfo, id string) (err error) {

	amfId := models.EndpointInfoToString(*callback)

	if _, ok := _cu.subscribedAmfs[amfId]; ok {
		return
	}
	amf := Amf{
		amfId: amfId,
		ranId: id,
	}
	if amf.cli, err = mesh.ConsumerFromEndpoint(callback); err != nil {
		_cu.Errorf("Fail to create callback: %+v", err)
		return
	}
	_cu.Infof("AMF %s subscribed", models.EndpointInfoToString(*callback))
	_cu.subscribedAmfs[amfId] = amf
	return
}

type Amf struct {
	amfId string
	ranId string //ran id generated at AMF
	cli   sbi.ConsumerClient
}

func (amf *Amf) notifySupportedTAList() {
	_cu.Infof("Notify AMF %s", amf.amfId)
	callback.RanInfoUpdate(amf.cli, amf.ranId, &models.RanInfoUpdateData{
		SupportedTAList: GetSupportedTAList(),
	})
}
