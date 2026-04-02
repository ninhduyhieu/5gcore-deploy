package context

import (
	"etrib5gc/util/ngapconv"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
)

func GetServedGuamiList() (list []ies.ServedGUAMIItem) {
	var regionId, setId, prtId aper.BitString
	var err error
	item := &ies.ServedGUAMIItem{}
	if item.GUAMI.PLMNIdentity, err = ngapconv.PlmnIdToNgap(_cu.plmnId); err != nil {
		return
	}
	if regionId, setId, prtId, err = ngapconv.AmfIdToNgap(_cu.amfId); err != nil {
		return
	}
	item.GUAMI.AMFRegionID = aper.BitString{
		Bytes:   regionId.Bytes,
		NumBits: 8,
	}
	item.GUAMI.AMFSetID = aper.BitString{
		Bytes:   setId.Bytes,
		NumBits: 10,
	}
	item.GUAMI.AMFPointer = aper.BitString{
		Bytes:   prtId.Bytes,
		NumBits: 6,
	}
	list = []ies.ServedGUAMIItem{*item}

	return
}

func GetPlmnSupportList() (list []ies.PLMNSupportItem) {
	var err error
	var snssai *ies.SNSSAI
	item := ies.PLMNSupportItem{}
	if item.PLMNIdentity, err = ngapconv.PlmnIdToNgap(_cu.plmnId); err != nil {
		_cu.Warning("Fail to convert to NGAP PlmnId: %+v", err)
		return nil
	}
	for _, slice := range _cu.slices {
		ssitem := &ies.SliceSupportItem{}
		if snssai, err = ngapconv.SNssaiToNgap(slice); err != nil {
			_cu.Warning("Fail to convert to NGAP snssai: %+v", err)
			continue
		}
		ssitem.SNSSAI = *snssai
		item.SliceSupportList = append(item.SliceSupportList, *ssitem)
	}
	return []ies.PLMNSupportItem{item}
}
