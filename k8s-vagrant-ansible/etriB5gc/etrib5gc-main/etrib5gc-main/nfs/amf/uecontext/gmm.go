package uecontext

import (
	amfctx "etrib5gc/nfs/amf/context"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/utils"
	"time"
)

func (ueCtx *UeContext) sendNasError(isGpp bool) {
	//TODO:
	ueCtx.Warnf("Send NAS error indication to UE not implement")
}

func (ueCtx *UeContext) buildDlNasTransport(isGpp bool, containerType uint8, content []byte, pduId *uint8, cause *uint8, timeRu *uint8, timeRv uint8) ([]byte, error) {
	msg := &nas.DlNasTransport{
		PayloadContainerType: containerType,
		PayloadContainer:     content,
		PduSessionId:         pduId,
		GmmCause:             cause,
	}
	if timeRu != nil {
		msg.BackOffTimerValue = nas.NewGprsTimer3(*timeRu, timeRv)
	}
	nasCtx := ueCtx.getNasContext()
	msg.SetSecurityHeader(nas.NasSecBoth)
	return nas.EncodeMm(nasCtx, msg, isGpp)
}

// notify UE of N1Sm sending error
func (ueCtx *UeContext) sendN1SmError(isGpp bool, n1Sm []byte, sId uint8) {
	cause := nas.Cause5GMMPayloadWasNotForwarded
	containerType := nas.PayloadContainerTypeN1SMInfo
	if pdu, err := ueCtx.buildDlNasTransport(isGpp, containerType, n1Sm, &sId, &cause, nil, 0); err != nil {
		ueCtx.Errorf("Fail to build Nas Dowlink for N1Sm error indication")
	} else if err = ueCtx.sendNas(pdu, isGpp); err != nil {
		ueCtx.Errorf("Fail to send N1Sm not-delivery indication for sesion %d: %+v", sId, err)
	}

	ueCtx.Infof("N1Sm not delivery indication for session %d is sent to UE", sId)
}

func (ueCtx *UeContext) sendNas(pdu []byte, isGpp bool) error {
	if isGpp && ueCtx.gpp.ranUe != nil {
		return ueCtx.gpp.ranUe.SendNas(pdu)
	} else if ueCtx.nonGpp.ranUe != nil {
		return ueCtx.nonGpp.ranUe.SendNas(pdu)
	}
	return fmt.Errorf("No access to send NAS")
}

/*
func (ueCtx *UeContext) sendGmm(nasCtx *nas.NasContext, msg nas.GmmMessage, isGpp bool) error {
	if pdu, err := nas.EncodeMm(nasCtx, msg, isGpp); err != nil {
		return utils.WrapError("Encode N1MM", err)
	} else {
		if err = ueCtx.sendNas(pdu, isGpp); err != nil {
			return utils.WrapError("Send N1Mm", err)
		}
	}
	return nil
}
*/

func (ueCtx *UeContext) sendRegistrationReject(isGpp bool, n1Cause uint8) {
	msg := &nas.RegistrationReject{
		GmmCause: n1Cause,
		T3502Value: &nas.GprsTimer2{
			Value: amfctx.T3502(),
		},
	}
	if attCtx := ueCtx.attCtx; attCtx != nil && len(attCtx.eap) > 0 {
		msg.EapMessage = attCtx.eap
	}

	nasCtx := ueCtx.getNasContext()
	if nasCtx != nil {
		msg.SetSecurityHeader(nas.NasSecBoth)
	} else {
		msg.SetSecurityHeader(nas.NasSecNone)
	}

	if pdu, err := nas.EncodeMm(nasCtx, msg, isGpp); err != nil {
		ueCtx.Errorf("Fail to encode RegistrationReject: %+v", err)
	} else {
		if err = ueCtx.sendNas(pdu, isGpp); err != nil {
			ueCtx.Errorf("Fail to send RegistrationReject: %+v", err)
		} else {
			ueCtx.Infof("RegistrationReject was sent to UE")
		}
	}
}

func (ueCtx *UeContext) sendServiceReject(isGpp bool, cause uint8, pduSessionStatus *[16]bool) {
	msg := &nas.ServiceReject{
		GmmCause: cause,
	}
	if attCtx := ueCtx.attCtx; attCtx != nil && len(attCtx.eap) > 0 {
		msg.EapMessage = attCtx.eap
	}
	if pduSessionStatus != nil {
		msg.PduSessionStatus = new(nas.PduSessionStatus)
		msg.PduSessionStatus.Set(*pduSessionStatus)
	}

	nasCtx := ueCtx.getNasContext()
	if nasCtx != nil {
		msg.SetSecurityHeader(nas.NasSecBoth)
	} else {
		msg.SetSecurityHeader(nas.NasSecNone)
	}

	if pdu, err := nas.EncodeMm(nasCtx, msg, isGpp); err == nil {
		if err := ueCtx.sendNas(pdu, isGpp); err != nil {
			ueCtx.Errorf("Fail to encode ServiceReject: %+v", err)
		} else {
			ueCtx.Infof("ServiceReject was sent to UE")
		}
	} else {
		ueCtx.Errorf("Fail to send ServiceReject: %+v", err)
	}
}

func (ueCtx *UeContext) buildRegistrationAccept(isGpp bool) (pdu []byte, err error) {
	msg := &nas.RegistrationAccept{}
	//fill in generic information
	if err = amfctx.FillRegistrationAccept(msg, isGpp); err != nil {
		return
	}

	//set registration result
	msg.RegistrationResult.SetResult(ueCtx.registrationStatus(isGpp))
	//set guti
	guti := ueCtx.getNasGuti()
	ueCtx.Tracef("Write GUTI: %s", guti.String())
	msg.Guti = &nas.MobileIdentity{
		Id: guti,
	}
	if ladnInfo := ueCtx.ladnInfo; len(ladnInfo) > 0 {
		/*
			//TODO:
				msg.LADNInformation = nas.NewLADNInformation(nas.RegistrationAcceptLADNInformationType)
				buf := make([]uint8, 0)
				var nasLadn []byte
				for _, ladn := range ladnInfo {
					if nasLadn, err = nas.LadnToNas(ladn.Dnn, ladn.TaiList); err != nil {
						return
					}
					buf = append(buf, nasLadn...)
				}
				msg.LADNInformation.SetLen(uint16(len(buf)))
				msg.LADNInformation.SetLADND(buf)
		*/
	}

	//set ue specific drx parameter
	if ueCtx.drx != nas.DRXValueNotSpecified {
		ueCtx.Tracef("Write DRX value")
		msg.NegotiatedDrxParameters = newUint8(ueCtx.drx)
	}

	if amPol := ueCtx.amPolicy; isGpp && amPol != nil {
		/* &&	amPol.ServAreaRes != nil { */
		/*

			msg.ServiceAreaList = nas.NewServiceAreaList(nas.RegistrationAcceptServiceAreaListType)
			partialServiceAreaList, convErr := nas.PartialServiceAreaListToNas(ueCtx.plmnId, *amPol.ServAreaRes)
			if convErr != nil {
				err = convErr
				return
			}
			msg.ServiceAreaList.SetLen(uint8(len(partialServiceAreaList)))
			msg.ServiceAreaList.SetPartialServiceAreaList(partialServiceAreaList)
		*/
	}

	attCtx := ueCtx.attCtx
	report := attCtx.report
	if len(attCtx.eap) > 0 {
		msg.EapMessage = attCtx.eap
	}

	//fill in UeContext specific information
	if len(ueCtx.taList) > 0 {
		ueCtx.Tracef("Write registration area list")
		msg.TaiList = buildServiceAreaList(ueCtx.taList)
	}

	if allowedSlices := ueCtx.getAllowedSlices(isGpp); len(allowedSlices) > 0 {
		ueCtx.Tracef("Write allowed slices")
		msg.AllowedNssai = buildNasAllowedSnssai(allowedSlices)
	}

	/*
		//TODO: add reject slices and configured slices

			if ue.NetworkSliceInfo != nil {
				if len(ue.NetworkSliceInfo.RejectedNssaiInPlmn) != 0 || len(ue.NetworkSliceInfo.RejectedNssaiInTa) != 0 {
					rejectedNssaiNas := nasconv.RejectedNssaiToNas(
						ue.NetworkSliceInfo.RejectedNssaiInPlmn, ue.NetworkSliceInfo.RejectedNssaiInTa)
					msg.RejectedNSSAI = &rejectedNssaiNas
					msg.RejectedNSSAI.SetIei(nas.RegistrationAcceptRejectedNSSAIType)
				}
			}

			if includeConfiguredNssaiCheck(ue) {
				msg.ConfiguredNSSAI = nas.NewConfiguredNSSAI(nas.RegistrationAcceptConfiguredNSSAIType)
				var buf []uint8
				for _, snssai := range ue.ConfiguredNssai {
					buf = append(buf, nasconv.SnssaiToNas(*snssai.ConfiguredSnssai)...)
				}
				msg.ConfiguredNSSAI.SetLen(uint8(len(buf)))
				msg.ConfiguredNSSAI.SetSNSSAIValue(buf)
			}
	*/

	/*
		//tungtq: not supported
			if ueCtx.SliceSubChanged() {
				msg.NetworkSlicingIndication =
					nas.NewNetworkSlicingIndication(nas.RegistrationAcceptNetworkSlicingIndicationType)
				msg.NetworkSlicingIndication.SetNSSCI(1)
				msg.NetworkSlicingIndication.SetDCNI(0)
				ueCtx.ue.SetSliceSubChanged(false) // reset the value
			}
	*/

	if report != nil {
		if report.StatusList != nil {
			msg.PduSessionStatus = new(nas.PduSessionStatus)
			msg.PduSessionStatus.Set(*report.StatusList)
		}

		if report.ReactList != nil {
			msg.PduSessionReactivationResult = new(nas.PduSessionReactivationResult)
			msg.PduSessionReactivationResult.Set(*report.ReactList)
		}

		if len(report.ErrPduList) > 0 {
			msg.PduSessionReactivationResultErrorCause = new(nas.PduSessionReactivationResultErrorCause)
			msg.PduSessionReactivationResultErrorCause.Set(report.ErrPduList, report.ErrCauses)
		}
	}
	msg.SetSecurityHeader(nas.NasSecBoth)
	nasCtx := ueCtx.getNasContext()

	return nas.EncodeMm(nasCtx, msg, isGpp)
}

func (ueCtx *UeContext) buildServiceAccept(isGpp bool) (pdu []byte, err error) {
	//statusList *[16]bool, reactList *[16]bool, errPduList []uint8, errCauses []uint8
	msg := new(nas.ServiceAccept)
	attCtx := ueCtx.attCtx  //non-nil
	report := attCtx.report //non-nil

	if len(attCtx.eap) > 0 {
		msg.EapMessage = attCtx.eap
	}

	if report != nil {
		if report.StatusList != nil {
			msg.PduSessionStatus = new(nas.PduSessionStatus)
			msg.PduSessionStatus.Set(*report.StatusList)
		}
		if report.ReactList != nil {
			msg.PduSessionReactivationResult = new(nas.PduSessionReactivationResult)
			msg.PduSessionReactivationResult.Set(*report.ReactList)
		}
		if len(report.ErrPduList) > 0 {
			msg.PduSessionReactivationResultErrorCause = new(nas.PduSessionReactivationResultErrorCause)
			msg.PduSessionReactivationResultErrorCause.Set(report.ErrPduList, report.ErrCauses)
		}
	}
	msg.SetSecurityHeader(nas.NasSecBoth)
	return nas.EncodeMm(ueCtx.getNasContext(), msg, isGpp)
}

func (ueCtx *UeContext) sendConfigurationUpdateCommand() error {
	msg := new(nas.ConfigurationUpdateCommand)
	ctx := ueCtx.configCtx
	flags := ctx.flags
	/*
		if slicing != nil {
			msg.NetworkSlicingIndication = newUint8(*slicing)
		}
		//TODO: fill information elements
	*/

	if flags.nitz {
		msg.FullNameForNetwork = nasNetworkName(amfctx.FullNetworkName())
		msg.ShortNameForNetwork = nasNetworkName(amfctx.ShortNetworkName())
		now := encodeUniversalTimeAndLocalTimeZone(time.Now())
		msg.UniversalTimeAndLocalTimeZone = now[:]
	}

	//must response
	ind := uint8(0x01)
	msg.ConfigurationUpdateIndication = &ind

	msg.SetSecurityHeader(nas.NasSecBoth)
	if pdu, err := nas.EncodeMm(ueCtx.getNasContext(), msg, flags.isGpp); err != nil {
		return utils.WrapError("Encode ConfigurationUpdateCommand", err)
	} else if err = ueCtx.sendNas(pdu, flags.isGpp); err != nil {
		return utils.WrapError("Send ConfigurationUpdateCommand", err)
	}
	ueCtx.Infof("ConfigurationUpdateCommand is sent to UE")
	//start timer and increase counter
	ctx.t3555.Start()
	ctx.t3555Cnt++
	return nil
}

func (ueCtx *UeContext) buildStatus5GMM(isGpp bool, cause uint8) (pdu []byte, err error) {
	msg := &nas.GmmStatus{
		GmmCause: cause,
	}
	nasCtx := ueCtx.getNasContext()
	if nasCtx != nil {
		msg.SetSecurityHeader(nas.NasSecBoth)
	} else {
		msg.SetSecurityHeader(nas.NasSecNone)
	}
	return nas.EncodeMm(nasCtx, msg, isGpp)
}

// send NAS DeregistrationRequest to UE
func (ueCtx *UeContext) sendDeregistrationRequest() error {
	ctx := ueCtx.deregCtx
	ranUe := ctx.ranUe

	targetRan := nas.AccessTypeBoth
	switch ctx.trigger.targetRan {
	case TARGET_GPP:
		targetRan = nas.AccessType3GPP
	case TARGET_NON_GPP:
		targetRan = nas.AccessTypeNon3GPP
	}

	msg := new(nas.DeregistrationRequestToUe)
	msg.DeRegistrationType.SetAccessType(targetRan)
	msg.DeRegistrationType.SetReregistration(ctx.trigger.retry)

	msg.SetSecurityHeader(nas.NasSecBoth)
	nasCtx := ueCtx.getNasContext()

	if pdu, err := nas.EncodeMm(nasCtx, msg, ranUe.IsGpp()); err == nil {
		if err = ranUe.SendNas(pdu); err == nil {
			ueCtx.Infof("DeregistrationRequest is sent to UE")
			ctx.t3522Cnt++
			ctx.t3522.Start()
		} else {
			return utils.WrapError("Send DeregistrationRequest", err)
		}
	} else {
		return utils.WrapError("Encode DeregistrationRequest", err)
	}
	return nil
}

// send NAS DeregistrationAccept to UE
func (ueCtx *UeContext) sendDeregistrationAccept(isGpp bool) error {
	ranUe := ueCtx.getRanUe(isGpp)
	msg := new(nas.DeregistrationAcceptFromUe)

	nasCtx := ueCtx.getNasContext()
	if nasCtx != nil {
		msg.SetSecurityHeader(nas.NasSecBoth)
	} else {
		msg.SetSecurityHeader(nas.NasSecNone)
	}

	if pdu, err := nas.EncodeMm(nasCtx, msg, isGpp); err == nil {
		if err = ranUe.SendNas(pdu); err != nil {
			return utils.WrapError("Send DeregistrationAccept", err)
		}
	} else {
		return utils.WrapError("Encode DeregistrationAccept", err)
	}
	return nil
}
