package context

import (
	"etrib5gc/common"
	"etrib5gc/mesh"
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi"
)

// Searching for Amf information from the paging manager give UE's 5gsTmsi
func FindAmf(fiveGsTmsi *ies.FiveGSTMSI, locInfo *ies.UserLocationInformation) (amfCli sbi.ConsumerClient, err error) {
	// look for AMF from 5GSTmsi
	if fiveGsTmsi != nil {
		tmp := fiveGsTmsi.AMFSetID.Bytes
		amfSet := tmp[0]*4 + (tmp[1] >> 6)
		amfSetStr := fmt.Sprintf("%d-%d", _cu.amfRegion, amfSet)
		insId := common.AmfPointerString(fiveGsTmsi.AMFPointer.Bytes[0])

		_cu.Infof("Search amf %s-%s", amfSetStr, insId)

		amfService := common.AmfServiceName(&_cu.plmnId, amfSetStr)
		if amfCli, err = mesh.ConsumerWithInstanceId(amfService, insId); err == nil {
			return
		} else {
			_cu.Warnf("Can't find AMF:%s-%s, use DAMF instead", amfSetStr, insId)
			err = nil
		}
	}

	//just create a client to DAMF
	damf := common.DamfServiceName(&_cu.plmnId)
	options := GetNfSelection()
	//TODO: indlude user location information into the options
	if amfCli, err = mesh.Consumer(damf, options); err != nil {
		_cu.Errorf("Fail to create Damf client: %+v", err)
		return
	}
	return
}
