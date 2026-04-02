package ngapconv

/*
import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func TraceDataToModels(traceActivation ies.TraceActivation) (traceData models.TraceData) {
	// TODO: finish this function when need
	return
}

func TraceDataToNgap(traceData models.TraceData, trsr string) (traceAct ies.TraceActivation, err error) {

	if len(trsr) != 4 {
		err = fmt.Errorf("Trace Recording Session Reference should be 2 octets")
		return
	}

	// NG-RAN Trace ID (left most 6 octet Trace Reference + last 2 octet Trace Recoding Session Reference)
	subStringSlice := strings.Split(traceData.TraceRef, "-")

	if len(subStringSlice) != 2 {
		err = fmt.Errorf("TraceRef format is not correct")
		return
	}

	plmnID := models.PlmnId{}
	plmnID.Mcc = subStringSlice[0][:3]
	plmnID.Mnc = subStringSlice[0][3:]
	var traceID []byte

	if traceID, err = hex.DecodeString(subStringSlice[1]); err != nil {
		return
	}
	var tmp *ies.PLMNIdentity
	if tmp, err = PlmnIdToNgap(plmnID); err != nil {
		return
	}
	traceReference := append(tmp.Value, traceID...)
	var trsrNgap []byte
	if trsrNgap, err = hex.DecodeString(trsr); err != nil {
		err = utils.WrapError("Decode trsr failed", err)
		return
	}

	nGRANTraceID := append(traceReference, trsrNgap...)

	traceAct.NGRANTraceID.Value = nGRANTraceID

	// Interfaces To Trace
	var interfacesToTrace []byte
	if interfacesToTrace, err = hex.DecodeString(traceData.InterfaceList); err != nil {
		err = utils.WrapError("Decode Interface failed", err)
		return
	}

	traceAct.InterfacesToTrace.Value = aper.BitString{
		Bytes:   interfacesToTrace,
		NumBits: 8,
	}

	// Trace Collection Entity IP Address
	var ngapIP ies.TransportLayerAddress
	if ngapIP, err = IPAddressToNgap(traceData.CollectionEntityIpv4Addr, string(traceData.CollectionEntityIpv6Addr)); err != nil {
		return
	}
	traceAct.TraceCollectionEntityIPAddress = &ngapIP

	// Trace Depth
	switch traceData.TraceDepth {
	case models.TRACEDEPTH_MINIMUM:
		traceAct.TraceDepth.Value = ies.TraceDepthMinimum
	case models.TRACEDEPTH_MEDIUM:
		traceAct.TraceDepth.Value = ies.TraceDepthMedium
	case models.TRACEDEPTH_MAXIMUM:
		traceAct.TraceDepth.Value = ies.TraceDepthMaximum
	case models.TRACEDEPTH_MINIMUM_WO_VENDOR_EXTENSION:
		traceAct.TraceDepth.Value = ies.TraceDepthMinimumwithoutvendorspecificextension
	case models.TRACEDEPTH_MEDIUM_WO_VENDOR_EXTENSION:
		traceAct.TraceDepth.Value = ies.TraceDepthMediumwithoutvendorspecificextension
	case models.TRACEDEPTH_MAXIMUM_WO_VENDOR_EXTENSION:
		traceAct.TraceDepth.Value = ies.TraceDepthMaximumwithoutvendorspecificextension
	}

	return
}
*/
