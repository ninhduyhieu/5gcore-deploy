package ran

import (
	"etrib5gc/nfs/pran/context"
	"fmt"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/utils"
)

func (r *Ran) sendNgapMsg(msg ngap.NgapMessageEncoder) (err error) {
	var packet []byte
	if packet, err = ngap.NgapEncode(msg); err == nil {
		if err = r.Send(packet); err != nil {
			err = utils.WrapError("Send Ngap packet", err)
		}
	} else {
		err = utils.WrapError("NgapEncode", err)
	}
	return
}

func (r *Ran) sendUeNotFoundError(amfUeNgapId *int64, ranUeNgapId *int64) {
	cause := &ies.Cause{
		Choice: ies.CausePresentRadionetwork,
		RadioNetwork: &ies.CauseRadioNetwork{
			Value: ies.CauseRadioNetworkUnknownlocaluengapid,
		},
	}
	if amfUeNgapId != nil {
		r.Warnf("UE Context not found for AmfUeNgapId %d]", *amfUeNgapId)
	} else if ranUeNgapId != nil {
		r.Warnf("UE Context not found for RanUeNgapId %d]", *ranUeNgapId)
	}
	r.sendErrorIndication(amfUeNgapId, ranUeNgapId, cause, nil)
}

func (r *Ran) sendErrorIndication(amfUeNgapId *int64, ranUeNgapId *int64,
	cause *ies.Cause, criticalityDiagnostics *ies.CriticalityDiagnostics) {
	r.Debug("Send ErrorIndication to gnB")

	if cause == nil && criticalityDiagnostics == nil {
		r.Error("[Build Error Indication] shall contain at least either the Cause or the Criticality Diagnostics")
	}

	msg := &ies.ErrorIndication{
		AMFUENGAPID:            amfUeNgapId,
		RANUENGAPID:            ranUeNgapId,
		Cause:                  cause,
		CriticalityDiagnostics: criticalityDiagnostics,
	}
	if err := r.sendNgapMsg(msg); err != nil {
		r.Errorf("Fail to send ErrorIndication: %+v", err)
	}
}

func (r *Ran) sendNGSetupResponse() error {
	r.Debug("Send NGSetupResponse to gnB")
	msg := &ies.NGSetupResponse{
		AMFName:             []byte(context.AmfName()),
		RelativeAMFCapacity: int64(context.RelativeCapacity()),
	}
	//served Guami list
	guamiList := context.GetServedGuamiList()
	if len(guamiList) == 0 {
		return fmt.Errorf("Served GUAMI list is empty")
	}
	msg.ServedGUAMIList = guamiList

	//plmn support list
	plmnList := context.GetPlmnSupportList()
	if len(plmnList) == 0 {
		return fmt.Errorf("Plmn support list is empty")
	}
	msg.PLMNSupportList = plmnList

	return r.sendNgapMsg(msg)
}

func (r *Ran) sendNGSetupFailure(cause ies.Cause) {
	r.Debug("Send NGSetupFailure to gnB")
	if cause.Choice == ies.CausePresentNothing {
		r.Errorf("Cause present is nil")
		return
	}
	msg := &ies.NGSetupFailure{
		Cause: cause,
	}
	if err := r.sendNgapMsg(msg); err != nil {
		r.Errorf("Fail to send NGSetupFailure: %+v", err)
	}
	return
}

// deprecated
func (r *Ran) sendRanConfigurationUpdateAcknowledge(criticalityDiagnostics *ies.CriticalityDiagnostics) error {
	r.Debug("Send RanConfigurationUpdateAcknowledge to gnB")
	msg := &ies.RANConfigurationUpdateAcknowledge{
		CriticalityDiagnostics: criticalityDiagnostics,
	}
	return r.sendNgapMsg(msg)
}

// deprecated
func (r *Ran) sendRanConfigurationUpdateFailure(cause ies.Cause,
	criticalityDiagnostics *ies.CriticalityDiagnostics) error {
	r.Debug("Send RanConfigurationUpdateFailure to gnB")
	msg := &ies.RANConfigurationUpdateFailure{
		CriticalityDiagnostics: criticalityDiagnostics,
		Cause:                  cause,
		TimeToWait: &ies.TimeToWait{
			Value: ies.TimeToWaitV1S,
		},
	}
	return r.sendNgapMsg(msg)
}

// partOfNGInterface: if reset type is "reset all", set it to nil TS 38.413 9.2.6.11
func (r *Ran) sendNGReset(cause ies.Cause,
	partOfNGInterface []ies.UEassociatedLogicalNGconnectionItem) error {
	r.Debug("Send NGReset to gnB")

	msg := &ies.NGReset{
		Cause: cause,
	}
	if partOfNGInterface == nil {
		msg.ResetType.Choice = ies.ResetTypePresentNgInterface
		msg.ResetType.NGInterface = &ies.ResetAll{
			Value: ies.ResetAllResetall,
		}
	} else {
		msg.ResetType.Choice = ies.ResetTypePresentPartofngInterface
		msg.ResetType.PartOfNGInterface = partOfNGInterface
	}

	return r.sendNgapMsg(msg)
}

func (r *Ran) sendNGResetAcknowledge(partOfNGInterface []ies.UEassociatedLogicalNGconnectionItem,
	criticalityDiagnostics *ies.CriticalityDiagnostics) error {
	r.Debug("Send NGResetAcknowledge to gnB")

	msg := &ies.NGResetAcknowledge{}
	// UE-associated Logical NG-connection List (optional)
	if len(partOfNGInterface) > 0 {
		items := []ies.UEassociatedLogicalNGconnectionItem{}
		for _, item := range partOfNGInterface {
			if item.AMFUENGAPID == nil && item.RANUENGAPID == nil {
				r.Warn("[Build NG Reset Ack] No AmfUeNgapID & UeContextNgapID")
				continue
			}

			items = append(items, item)
		}

		msg.UEassociatedLogicalNGconnectionList = items
	} else if partOfNGInterface != nil {
		return fmt.Errorf("Length of partOfNGInterface is 0")
	}
	return r.sendNgapMsg(msg)
}

func (r *Ran) sendPathSwitchRequestFailure(amfUeNgapId int64, ranUeNgapId int64, sessions []ies.PDUSessionResourceReleasedItemPSFail, criticalityDiagnostics *ies.CriticalityDiagnostics) error {
	r.Debug("Send PathSwitchRequestFailure to gnB")
	msg := &ies.PathSwitchRequestFailure{
		AMFUENGAPID:                          amfUeNgapId,
		RANUENGAPID:                          ranUeNgapId,
		PDUSessionResourceReleasedListPSFail: sessions,
		CriticalityDiagnostics:               criticalityDiagnostics,
	}
	return r.sendNgapMsg(msg)
}
