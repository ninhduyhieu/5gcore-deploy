package ran

import (
	"etrib5gc/util/ngapconv"
	//	"github.com/lvdund/ngap/aper"
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
)

func ranId2String(id *ies.GlobalRANNodeID) (plmnId models.PlmnId, idStr string, access models.AccessType) {
	access = models.ACCESSTYPE_3GPP_ACCESS
	switch id.Choice {
	case ies.GlobalRANNodeIDPresentGlobalgnbId:
		plmnId = ngapconv.PlmnIdToModels(id.GlobalGNBID.PLMNIdentity)
		idStr = fmt.Sprintf("plnmnId=%s;gnbId=%x", models.PlmnIdToString(plmnId), id.GlobalGNBID.GNBID.GNBID.Bytes)
	case ies.GlobalRANNodeIDPresentGlobalngenbId:
		plmnId = ngapconv.PlmnIdToModels(id.GlobalNgENBID.PLMNIdentity)
		idStr = fmt.Sprintf("plnmnId=%s;gnbId=%x", models.PlmnIdToString(plmnId), id.GlobalNgENBID.NgENBID.MacroNgENBID.Bytes)
	case ies.GlobalRANNodeIDPresentGlobaln3IwfId:
		plmnId = ngapconv.PlmnIdToModels(id.GlobalN3IWFID.PLMNIdentity)
		idStr = fmt.Sprintf("plnmnId=%s;gnbId=%x", models.PlmnIdToString(plmnId), id.GlobalN3IWFID.N3IWFID.N3IWFID.Bytes)
		access = models.ACCESSTYPE_NON_3GPP_ACCESS
	default:
		idStr = "unknown-ran-id"
	}
	return
}

func causeConvert(cause *ies.Cause) (present uint8, value uint8) {
	if cause == nil {
		return
	}
	present = uint8(cause.Choice)
	switch cause.Choice {
	case ies.CausePresentRadionetwork:
		value = uint8(cause.RadioNetwork.Value)
	case ies.CausePresentTransport:
		value = uint8(cause.Transport.Value)
	case ies.CausePresentNas:
		value = uint8(cause.Nas.Value)
	case ies.CausePresentProtocol:
		value = uint8(cause.Protocol.Value)
	case ies.CausePresentMisc:
		value = uint8(cause.Misc.Value)
	}
	return
}

/*
func convertSupportedTai(tailist *ies.SupportedTAList) (results common.SupportedTAList) {
	results = make(common.SupportedTAList, len(tailist.Value))
	for i, item := range tailist.Value {
		newitem := &results[i]
		newitem.Tac = hex.EncodeToString(item.TAC.Value)
		newitem.Plmns = make(map[string]common.SupportedPlmnItem)
		for _, plmn := range item.BroadcastPLMNList.Value {
			plmnId := ngapconv.PlmnIdToModels(*plmn.PLMNIdentity)
			slices := []models.Snssai{}
			for _, snssai := range plmn.TAISliceSupportList.Value {
				slices = append(slices, ngapconv.SNssaiToModels(*snssai.SNSSAI))
			}
			newitem.Plmns[j] = common.NewSupportedPlmnItem(plmnId, slices)
		}
	}
	return
}
*/
func handleCriticalityDiagnostics(v *ies.CriticalityDiagnostics) {
	//DO NOTHING NOW

}

/*
func (r *Ran) printAndGetCause(cause *ies.Cause) (present int, value aper.Enumerated) {
	present = int(cause.Choice)
	switch cause.Choice {
	case ies.CausePresentRadioNetwork:
		r.Warnf("Cause RadioNetwork[%d]", cause.RadioNetwork.Value)
		value = cause.RadioNetwork.Value
	case ies.CausePresentTransport:
		r.Warnf("Cause Transport[%d]", cause.Transport.Value)
		value = cause.Transport.Value
	case ies.CausePresentProtocol:
		r.Warnf("Cause Protocol[%d]", cause.Protocol.Value)
		value = cause.Protocol.Value
	case ies.CausePresentNas:
		r.Warnf("Cause Nas[%d]", cause.Nas.Value)
		value = cause.Nas.Value
	case ies.CausePresentMisc:
		r.Warnf("Cause Misc[%d]", cause.Misc.Value)
		value = cause.Misc.Value
	default:
		r.Errorf("Invalid Cause group[%d]", cause.Choice)
	}
	return
}
*/
