package ue

import (
	"encoding/hex"
	"etrib5gc/util/ngapconv"
	"fmt"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"time"
)

func ngapUeSecurityCapability(secCap *models.UeSecurityCapability) (ngapSecCap *ies.UESecurityCapabilities, err error) {
	if secCap == nil {
		err = fmt.Errorf("Empty UeSecurityCapability")
		return
	}

	tmp := &ies.UESecurityCapabilities{}
	if secCap.Nr != nil {
		if len(secCap.Nr.Enc) != 2 {
			err = fmt.Errorf("Invalid NR encryption algorithm capabilities")
			return
		}
		if len(secCap.Nr.Int) != 2 {
			err = fmt.Errorf("Invalid NR integrity algorithm capabilities")
			return
		}
		//tmp.NRencryptionAlgorithms = secCap.Nr.Enc
		//tmp.NRintegrityProtectionAlgorithms = secCap.Nr.Int
		tmp.NRencryptionAlgorithms = aper.BitString{
			NumBits: 16,
			Bytes:   secCap.Nr.Enc,
		}
		tmp.NRintegrityProtectionAlgorithms = aper.BitString{
			NumBits: 16,
			Bytes:   secCap.Nr.Int,
		}
	}
	if secCap.Eutra != nil {
		if len(secCap.Eutra.Enc) != 2 {
			err = fmt.Errorf("Invalid NR encryption algorithm capabilities")
			return
		}
		if len(secCap.Eutra.Int) != 2 {
			err = fmt.Errorf("Invalid NR integrity algorithm capabilities")
			return
		}
		//tmp.EUTRAencryptionAlgorithms = secCap.Eutra.Enc
		//tmp.EUTRAintegrityProtectionAlgorithms = secCap.Eutra.Int
		tmp.EUTRAencryptionAlgorithms = aper.BitString{
			NumBits: 16,
			Bytes:   secCap.Eutra.Enc,
		}
		tmp.EUTRAintegrityProtectionAlgorithms = aper.BitString{
			NumBits: 16,
			Bytes:   secCap.Eutra.Int,
		}
	}
	ngapSecCap = tmp
	return
}

func ngapSecurityContext(secCtx *models.SecurityContext) (ngapSecCtx *ies.SecurityContext, err error) {
	if secCtx == nil {
		err = fmt.Errorf("Empty SecurityContext")
		return
	}
	if len(secCtx.Nh) != 32 {
		err = fmt.Errorf("Invalid key size (%s) for security context; expected value is 32", len(secCtx.Nh))
		return
	}
	/*
		ngapSecCtx = &ies.SecurityContext{
			NextHopNH:            secCtx.Nh,
			NextHopChainingCount: int64(secCtx.Ncc),
		}
	*/
	ngapSecCtx = &ies.SecurityContext{
		NextHopNH: aper.BitString{
			NumBits: 256,
			Bytes:   secCtx.Nh,
		},
		NextHopChainingCount: int64(secCtx.Ncc),
	}
	return
}

func ngapGuami(guami *models.Guami) (ngapGuami *ies.GUAMI, err error) {
	var plmnId []byte
	if plmnId, err = ngapconv.PlmnIdToNgap(models.PlmnId{
		Mnc: guami.PlmnId.Mnc,
		Mcc: guami.PlmnId.Mcc,
	}); err != nil {
		err = fmt.Errorf("Fail to convert plmnid to Ngap: %+v", err)
		return
	}
	var amfRegion, amfSet, amfPointer aper.BitString
	if amfRegion, amfSet, amfPointer, err = ngapconv.AmfIdToNgap(guami.AmfId); err != nil {
		err = fmt.Errorf("Fail to convert amfid to Ngap: %+v", err)
		return
	}
	/*
		ngapGuami = &ies.GUAMI{
			PLMNIdentity: plmnId,
			AMFRegionID:  amfRegion.Bytes,
			AMFSetID:     amfSet.Bytes,
			AMFPointer:   amfPointer.Bytes,
		}
	*/
	ngapGuami = &ies.GUAMI{
		PLMNIdentity: plmnId,
		AMFRegionID:  amfRegion,
		AMFSetID:     amfSet,
		AMFPointer:   amfPointer,
	}
	return
}

func ngapUeAmbr(ueAmbr *models.UeAmbr) *ies.UEAggregateMaximumBitRate {
	if ueAmbr == nil {
		return nil
	}
	return &ies.UEAggregateMaximumBitRate{
		UEAggregateMaximumBitRateUL: ueAmbr.Ul,
		UEAggregateMaximumBitRateDL: ueAmbr.Dl,
	}
}
func locConvert(ngapLoc *ies.UserLocationInformation) (sbiLoc *models.UserLocation, err error) {
	if ngapLoc == nil {
		err = fmt.Errorf("Empty UserLocationInformation")
		return
	}
	switch ngapLoc.Choice {
	case ies.UserLocationInformationPresentUserlocationinformationeutra:
		err = fmt.Errorf("EUTRA location not support")
	case ies.UserLocationInformationPresentUserlocationinformationnr:
		tai := ngapLoc.UserLocationInformationNR.TAI
		plmnId := ngapconv.PlmnIdToModels(tai.PLMNIdentity)
		tac := hex.EncodeToString(tai.TAC)

		nrcgi := ngapLoc.UserLocationInformationNR.NRCGI
		nrPlmnId := ngapconv.PlmnIdToModels(nrcgi.PLMNIdentity)
		nrCellId := ngapconv.BitStringToHex(&nrcgi.NRCellIdentity)

		sbiLoc = &models.UserLocation{
			NrLocation: &models.NrLocation{
				Tai: models.Tai{
					PlmnId: plmnId,
					Tac:    tac,
				},
				Ncgi: models.Ncgi{
					PlmnId:   nrPlmnId,
					NrCellId: nrCellId,
				},
				UeLocationTimestamp: time.Now().Format(time.UnixDate),
			},
		}

		if ts := ngapLoc.UserLocationInformationNR.TimeStamp; len(ts) > 0 {
			age, _ := ngapconv.TimeStampToInt32(ts)
			tmp := int(age)
			sbiLoc.NrLocation.AgeOfLocationInformation = &tmp
		}
	case ies.UserLocationInformationPresentUserlocationinformationn3Iwf:
		err = fmt.Errorf("N3IWF location not support")

	case ies.UserLocationInformationPresentChoiceExtensions:
		err = fmt.Errorf("Extention location type not support")
	default:
		err = fmt.Errorf("unknown location type")
	}
	return
}

func ngapCause(cause models.N2Cause) ies.Cause {
	ngapCause := ies.Cause{
		Choice: uint64(cause.CausePresent),
	}
	switch ngapCause.Choice {
	case ies.CausePresentRadionetwork:
		ngapCause.RadioNetwork = &ies.CauseRadioNetwork{
			Value: aper.Enumerated(cause.CauseValue),
		}
	case ies.CausePresentTransport:
		ngapCause.Transport = &ies.CauseTransport{
			Value: aper.Enumerated(cause.CauseValue),
		}
	case ies.CausePresentNas:
		ngapCause.Nas = &ies.CauseNas{
			Value: aper.Enumerated(cause.CauseValue),
		}
	case ies.CausePresentProtocol:
		ngapCause.Protocol = &ies.CauseProtocol{
			Value: aper.Enumerated(cause.CauseValue),
		}
	default:
		ngapCause.Choice = ies.CausePresentMisc
		ngapCause.Misc = &ies.CauseMisc{
			Value: aper.Enumerated(cause.CauseValue),
		}
	}
	return ngapCause
}

func n2Cause(cause *ies.Cause) models.N2Cause {
	present, value := causeConvert(cause)
	return models.N2Cause{
		CausePresent: int16(present),
		CauseValue:   int16(value),
	}
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

func convertTargetId(ngapId *ies.TargetID) (id *models.GlobalRanNodeId, err error) {
	if ngapId.Choice != ies.TargetIDPresentTargetrannodeid {
		err = fmt.Errorf("targetId type[%d] is not supported", ngapId.Choice)
		return
	}
	if tmp := ngapId.TargetRANNodeID.GlobalRANNodeID; tmp.Choice != ies.GlobalRANNodeIDPresentGlobalgnbId {
		err = fmt.Errorf("targetId is not not a global ran node id")
	} else {
		plmnId := ngapconv.PlmnIdToModels(tmp.GlobalGNBID.PLMNIdentity)
		if gnbId := tmp.GlobalGNBID.GNBID.GNBID; gnbId == nil || len(gnbId.Bytes) == 0 {
			err = fmt.Errorf("Empty Global GnB identity")
		} else {
			id = &models.GlobalRanNodeId{
				PlmnId: plmnId,
				GNbId: &models.GNbId{
					BitLength: 8 * len(gnbId.Bytes), //TODO: need BitString type gnbId
					GNBValue:  string(gnbId.Bytes),
				},
			}
		}
	}
	return
}

func newAperOctetString(buf []byte) *aper.OctetString {
	tmp := aper.OctetString(buf)
	return &tmp
}

func newBool(v bool) *bool {
	return &v
}

func newInt64(v int64) *int64 {
	return &v
}
