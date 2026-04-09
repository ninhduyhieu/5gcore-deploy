package ue

import (
	"encoding/hex"
	"etrib5gc/util/ngapconv"
	"fmt"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

/***************** Handover messages ****************/

func (ueCtx *UeContext) sendHandoverCommand(msg *models.HandoverCommand) {
	ngapMsg := &ies.HandoverCommand{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
		HandoverType: ies.HandoverType{
			Value: aper.Enumerated(msg.HandoverType),
		},
		TargetToSourceTransparentContainer: msg.TargetToSourceContent,
	}

	// NAS Security Parameters from NG-RAN [C-iftoEPS]
	if aper.Enumerated(msg.HandoverType) == ies.HandoverTypeFivegstoeps {
		ngapMsg.NASSecurityParametersFromNGRAN = []byte{} //TODO: do something here
	}
	switchedList := []ies.PDUSessionResourceHandoverItem{}
	releasedList := []ies.PDUSessionResourceToReleaseItemHOCmd{}
	for _, s := range msg.Sessions {
		switch s.N2SmInfoType {
		case models.N2SMINFOTYPE_HANDOVER_CMD:
			item := ies.PDUSessionResourceHandoverItem{}
			//TODO: set content
			switchedList = append(switchedList, item)
		case models.N2SMINFOTYPE_HANDOVER_PREP_FAIL:
			item := ies.PDUSessionResourceToReleaseItemHOCmd{}
			//TODO: set content
			releasedList = append(releasedList, item)
		default:
			//warning
		}
	}

	if len(switchedList) > 0 {
		ngapMsg.PDUSessionResourceHandoverList = switchedList
	}

	if len(releasedList) > 0 {
		ngapMsg.PDUSessionResourceToReleaseListHOCmd = releasedList
	}

	if err := ueCtx.sendNgapMsg(ngapMsg); err != nil {
		ueCtx.Errorf("Fail to send NGAP HandoverCommand: %+v", err)
	} else {
		ueCtx.Debug("NGAP HandoverCommand is sent to gnB")
	}
}
func (ueCtx *UeContext) sendHandoverPreparationFailure(msg *models.HandoverPreparationFailure) {
	ngapMsg := &ies.HandoverPreparationFailure{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	if msg != nil {
		ngapMsg.Cause = ngapCause(msg.Cause)
	}

	if err := ueCtx.sendNgapMsg(ngapMsg); err != nil {
		ueCtx.Errorf("Fail to send NGAP HandoverPreparationFailure: %+v", err)
	} else {
		ueCtx.Debug("NGAP HandoverPreparationFailure is sent to gnB")
	}
}

func (ueCtx *UeContext) sendHandoverRequest(msg *models.HandoverRequest) (err error) {
	var ngapSecCap *ies.UESecurityCapabilities
	if ngapSecCap, err = ngapUeSecurityCapability(&msg.UeSecurityCapability); err != nil {
		return utils.WrapError("Convert UeSecurityCapability into NGAP format", err)
	}
	var ngapSecCtx *ies.SecurityContext
	if ngapSecCtx, err = ngapSecurityContext(&msg.SecurityContext); err != nil {
		return utils.WrapError("Convert SecurityContext into NGAP format", err)
	}
	var allowedNssai []ies.AllowedNSSAIItem
	if allowedNssai, err = ngapconv.AllowedNssaiToNgap(msg.AllowedNssai.AllowedSnssaiList); err != nil {
		return utils.WrapError("Convert AllowedNssai into NGAP format", err)
	}
	ueAmbr := ngapUeAmbr(&msg.UeAmbr)
	var guami *ies.GUAMI
	if guami, err = ngapGuami(&msg.Guami); err != nil {
		return utils.WrapError("Convert GUAMI into NGAP format", err)
	}

	ngapMsg := &ies.HandoverRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		HandoverType: ies.HandoverType{
			aper.Enumerated(msg.HandoverType),
		},
		Cause:                              ngapCause(msg.Cause),
		SourceToTargetTransparentContainer: msg.SourceToTargetContent,
		UESecurityCapabilities:             *ngapSecCap,
		SecurityContext:                    *ngapSecCtx,
		AllowedNSSAI:                       allowedNssai,
		UEAggregateMaximumBitRate:          *ueAmbr,
		GUAMI:                              *guami,
	}

	if msg.NewSecInd {
		ngapMsg.NewSecurityContextInd = &ies.NewSecurityContextInd{
			Value: ies.NewSecurityContextIndTrue,
		}
	}

	if l := len(msg.MaskedImeisv); l > 0 {
		if l != 8 {
			return fmt.Errorf("MaskedImeiSV length [%d] must be 8", l)
		}
		ngapMsg.MaskedIMEISV = &aper.BitString{
			Bytes:   msg.MaskedImeisv, //64 bits (aper.BitString)
			NumBits: 64,
		}
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) sendPathSwitchAcknowledge(msg *models.PathSwitchAcknowledge) {

	var err error
	var ngapSecCap *ies.UESecurityCapabilities
	if ngapSecCap, err = ngapUeSecurityCapability(&msg.UeSecurityCapability); err != nil {
		ueCtx.Errorf("Fail to convert UeSecurityCapability into NGAP format: %+v", err)
		return
	}
	var ngapSecCtx *ies.SecurityContext
	if ngapSecCtx, err = ngapSecurityContext(&msg.SecurityContext); err != nil {
		ueCtx.Errorf("Fail to convert SecurityContext into NGAP format: %+v", err)
		return
	}
	var allowedNssai []ies.AllowedNSSAIItem
	if allowedNssai, err = ngapconv.AllowedNssaiToNgap(msg.AllowedNssai.AllowedSnssaiList); err != nil {
		ueCtx.Errorf("Fail to convert allowed Nssai to NGAP IE")
		return
	}

	ngapMsg := &ies.PathSwitchRequestAcknowledge{
		AMFUENGAPID:            ueCtx.cuNgapId(),
		RANUENGAPID:            ueCtx.ranNgapId,
		UESecurityCapabilities: ngapSecCap,
		SecurityContext:        *ngapSecCtx,
		AllowedNSSAI:           allowedNssai,
		//CoreNetworkAssistanceInformationForInactive: coreAssist,
		//RRCInactiveTransitionReportRequest:          rrcReport,
	}
	switchedSessions := []ies.PDUSessionResourceSwitchedItem{}
	releasedSessions := []ies.PDUSessionResourceReleasedItemPSAck{}
	for _, s := range msg.Sessions {
		switch s.N2SmInfoType {
		case models.N2SMINFOTYPE_PATH_SWITCH_REQ_ACK:
			switchedSessions = append(switchedSessions, ies.PDUSessionResourceSwitchedItem{
				PDUSessionID:                         int64(s.SessionId),
				PathSwitchRequestAcknowledgeTransfer: s.N2SmInfo,
			})
		case models.N2SMINFOTYPE_PATH_SWITCH_REQ_FAIL:
			releasedSessions = append(releasedSessions, ies.PDUSessionResourceReleasedItemPSAck{
				PDUSessionID:                          int64(s.SessionId),
				PathSwitchRequestUnsuccessfulTransfer: s.N2SmInfo,
			})

		default:
			continue
		}
	}
	if len(switchedSessions) == 0 {
		ueCtx.Errorf("No switched session in PathSwitchAcknowledge")
		return
	} else {
		ngapMsg.PDUSessionResourceSwitchedList = switchedSessions
	}
	if len(releasedSessions) > 0 {
		ngapMsg.PDUSessionResourceReleasedListPSAck = releasedSessions
	}

	// New Security Context Indicator (optional)
	if msg.NewSecInd != nil && *msg.NewSecInd {
		ngapMsg.NewSecurityContextInd = &ies.NewSecurityContextInd{
			Value: ies.NewSecurityContextIndTrue,
		}
	}

	if err := ueCtx.sendNgapMsg(ngapMsg); err != nil {
		ueCtx.Errorf("Fail to send NGAP PathSwitchAcknowledge: %+v", err)
	} else {
		ueCtx.Debug("NGAP PathSwitchAcknowledge is sent to gnB")
	}
}

func (ueCtx *UeContext) sendPathSwitchFailure(msg *models.PathSwitchFailure) {

	ngapMsg := &ies.PathSwitchRequestFailure{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}
	if msg != nil {
		sessions := []ies.PDUSessionResourceReleasedItemPSFail{}
		for _, s := range msg.Sessions {
			if s.N2SmInfoType == models.N2SMINFOTYPE_PATH_SWITCH_REQ_FAIL {
				item := &ies.PDUSessionResourceReleasedItemPSFail{
					PDUSessionID:                          int64(s.SessionId),
					PathSwitchRequestUnsuccessfulTransfer: s.N2SmInfo,
				}
				sessions = append(sessions, *item)
			}
		}
		if len(sessions) > 0 {
			ngapMsg.PDUSessionResourceReleasedListPSFail = sessions
		}
	}
	if err := ueCtx.sendNgapMsg(ngapMsg); err != nil {
		ueCtx.Errorf("Fail to send NGAP PathSwitchRequestFailure: %+v", err)
	} else {
		ueCtx.Debug("NGAP PathSwitchFailure is sent to gnB")
	}

}

func (ueCtx *UeContext) sendHandoverCancelAcknowledge(msg *models.HandoverCancelAcknowledge) {
	ngapMsg := &ies.HandoverCancelAcknowledge{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	if err := ueCtx.sendNgapMsg(ngapMsg); err != nil {
		ueCtx.Errorf("Fail to send HandoverCancelAcknowledge: %+v", err)
	} else {
		ueCtx.Debug("NGAP HandoverCancelAcknowledge is sent to gnB")
	}
	return
}

// RanStatusTransferTransparentContainer from Uplink Ran Configuration Transfer
func (ueCtx *UeContext) sendDownlinkRanStatusTransfer(container ies.RANStatusTransferTransparentContainer) {

	ngapMsg := &ies.DownlinkRANStatusTransfer{
		AMFUENGAPID:                           ueCtx.cuNgapId(),
		RANUENGAPID:                           ueCtx.ranNgapId,
		RANStatusTransferTransparentContainer: container,
	}

	if err := ueCtx.sendNgapMsg(ngapMsg); err != nil {
		ueCtx.Errorf("Fail to send NGAP HandoverCancelAcknowledge: %+v", err)
	}
	ueCtx.Debug("NGAP DownlinkRanStatusTransfer is sent to gnB")
	return
}

/***************** NAS related messages ****************/

// Send downlink Nas message to gnB
func (ueCtx *UeContext) sendDownlinkNasTransport(nasPdu []byte) error {

	//	mobilityRestrictionList *ies.MobilityRestrictionList
	ngapMsg := &ies.DownlinkNASTransport{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
		NASPDU:      nasPdu,
	}

	// Old AMF (optional)
	// RAN Paging Priority (optional)
	// Mobility Restriction List (optional)
	// Index to RAT/Frequency Selection Priority (optional)
	// UE Aggregate Maximum Bit Rate (optional)
	// Allowed NSSAI (optional)

	return ueCtx.sendNgapMsg(ngapMsg)
}

/***************** PDU session management messages ****************/
func (ueCtx *UeContext) sendPduSessionResourceSetupRequest(sessionId uint8, n2SmInfo []byte, n1Sm []byte, snssai *models.Snssai) (err error) {

	if ueCtx.ambr == nil {
		return fmt.Errorf("UeAmbr not initialized")
	}
	// TODO: Ran Paging Priority (optional)
	ngapMsg := &ies.PDUSessionResourceSetupRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	// Ran Paging Priority (optional)
	// PDU Session Resource Setup Request list
	pduList := []ies.PDUSessionResourceSetupItemSUReq{
		ies.PDUSessionResourceSetupItemSUReq{
			PDUSessionID:                           int64(sessionId),
			PDUSessionResourceSetupRequestTransfer: n2SmInfo,
		},
	}
	if snssai != nil {
		var ngapSnssai *ies.SNSSAI
		if ngapSnssai, err = ngapconv.SNssaiToNgap(*snssai); err != nil {
			return utils.WrapError("Convert Snssai to NGAP IE", err)
		}
		pduList[0].SNSSAI = *ngapSnssai
	}

	if len(n1Sm) > 0 {
		//ueCtx.Infof("Send N1Sm downlink: %x(%d)", n1Sm, len(n1Sm))
		pduList[0].PDUSessionNASPDU = n1Sm
	}
	ngapMsg.PDUSessionResourceSetupListSUReq = pduList
	ngapMsg.UEAggregateMaximumBitRate = ngapUeAmbr(ueCtx.ambr)
	//NOTE: AMBR should be sent in UeContext Init or modify/update

	return ueCtx.sendNgapMsg(ngapMsg)
}

//TS138.413-V15.3.0 8.2.2

func (ueCtx *UeContext) sendPduSessionResourceReleaseCommand(sessionId uint8, n2SmInfo []byte, n1Sm []byte) error {
	ngapMsg := &ies.PDUSessionResourceReleaseCommand{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	// NAS-PDU (optional)
	if len(n1Sm) > 0 {
		ngapMsg.NASPDU = n1Sm
	}

	// PDUSessionResourceToReleaseListRelCmd
	pduList := []ies.PDUSessionResourceToReleaseItemRelCmd{
		ies.PDUSessionResourceToReleaseItemRelCmd{
			PDUSessionID:                             int64(sessionId),
			PDUSessionResourceReleaseCommandTransfer: n2SmInfo,
		},
	}

	ngapMsg.PDUSessionResourceToReleaseListRelCmd = pduList

	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) sendPduSessionResourceModifyRequest(sbiMsg *models.SessionResourceModifyRequest) error {
	ngapMsg := &ies.PDUSessionResourceModifyRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}
	// Ran Paging Priority (optional)
	// PDU Session Resource Modify Request List
	pduList := []ies.PDUSessionResourceModifyItemModReq{
		ies.PDUSessionResourceModifyItemModReq{
			PDUSessionID:                            int64(sbiMsg.SessionId),
			PDUSessionResourceModifyRequestTransfer: sbiMsg.Transfer,
			NASPDU:                                  sbiMsg.N1Sm,
		},
	}

	ngapMsg.PDUSessionResourceModifyListModReq = pduList

	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) sendPduSessionResourceModifyConfirm(sessionId uint8, transfer []byte, success bool) error {

	ngapMsg := &ies.PDUSessionResourceModifyConfirm{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	// PDU Session Resource Modify Confirm List
	if success {
		pduList := []ies.PDUSessionResourceModifyItemModCfm{
			ies.PDUSessionResourceModifyItemModCfm{
				PDUSessionID:                            int64(sessionId),
				PDUSessionResourceModifyConfirmTransfer: transfer,
			},
		}

		ngapMsg.PDUSessionResourceModifyListModCfm = pduList
	} else {
		pduList := []ies.PDUSessionResourceFailedToModifyItemModCfm{
			ies.PDUSessionResourceFailedToModifyItemModCfm{
				PDUSessionID: int64(sessionId),
				PDUSessionResourceModifyIndicationUnsuccessfulTransfer: transfer,
			},
		}
		ngapMsg.PDUSessionResourceFailedToModifyListModCfm = pduList
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

/***************** UeContext related messages ****************/
func (ueCtx *UeContext) sendInitialContextSetupRequest(sbiMsg *models.UeContextSetupRequest) (err error) {
	ueCtx.ambr = &sbiMsg.UeAmbr
	ngapMsg := &ies.InitialContextSetupRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}
	// Old AMF (optional)
	if len(sbiMsg.OldAmf) > 0 {
		ngapMsg.OldAMF = []byte(sbiMsg.OldAmf)
	}
	// UE Aggregate Maximum Bit Rate (conditional: if pdu session resource setup)
	// The subscribed UE-AMBR is a subscription parameter which is
	// retrieved from UDM and provided to the (R)AN by the AMF
	ngapMsg.UEAggregateMaximumBitRate = ngapUeAmbr(&sbiMsg.UeAmbr)
	if len(sbiMsg.N2SmInfoDownlinks) > 0 {
		// PDU Session Resource Setup Request List
		sessionList := []ies.PDUSessionResourceSetupItemCxtReq{}
		var item *ies.PDUSessionResourceSetupItemCxtReq
		var ngapSnssai *ies.SNSSAI
		for _, s := range sbiMsg.N2SmInfoDownlinks {
			item = &ies.PDUSessionResourceSetupItemCxtReq{
				PDUSessionID: int64(s.SessionId),
			}
			if ngapSnssai, err = ngapconv.SNssaiToNgap(*s.Snssai); err != nil {
				err = utils.WrapError("Convert SNssai into Ngap: format", err)
				return
			}
			item.SNSSAI = *ngapSnssai
			item.PDUSessionResourceSetupRequestTransfer = s.N2SmInfo
			if len(s.NasPdu) > 0 {
				item.NASPDU = s.NasPdu
			}
			sessionList = append(sessionList, *item)
		}

		ngapMsg.PDUSessionResourceSetupListCxtReq = sessionList

	} /* else {
		ueCtx.Warnf("No PDU session in UeContextSetupRequest")
	}
	*/
	// GUAMI
	var guami *ies.GUAMI
	if guami, err = ngapGuami(&sbiMsg.Guami); err != nil {
		return utils.WrapError("Convert Guami into Ngap format", err)
	}
	ngapMsg.GUAMI = *guami

	// Allowed NSSAI
	if len(sbiMsg.AllowedNssai) > 0 {
		var allowedNssai []ies.AllowedNSSAIItem
		if allowedNssai, err = ngapconv.AllowedNssaiToNgap(sbiMsg.AllowedNssai); err != nil {
			return utils.WrapError("Convert allowedNssai into Ngap format", err)
		}
		ngapMsg.AllowedNSSAI = allowedNssai

	} else {
		ueCtx.Warnf("Empty AllowedSnssai")
	}

	// UE Security Capabilities
	var ueSecCap *ies.UESecurityCapabilities

	if ueSecCap, err = ngapUeSecurityCapability(&sbiMsg.UeSecCap); err != nil {
		return utils.WrapError("Convert UeSecurityCapability into Ngap format", err)
	}
	ngapMsg.UESecurityCapabilities = *ueSecCap

	// Security Key
	var securityKey aper.BitString
	if securityKey, err = ngapconv.ByteToBitString(sbiMsg.SecKey, 256); err != nil {
		return utils.WrapError("Convert security key into Ngap format", err)
	}

	ngapMsg.SecurityKey = securityKey

	// NAS-PDU (optional)
	if len(sbiMsg.NasPdu) > 0 {
		ueCtx.Tracef("Set nas pdu, len=%d", len(sbiMsg.NasPdu))
		ngapMsg.NASPDU = sbiMsg.NasPdu
	}

	// UE Radio Capability (optional)
	if len(sbiMsg.UeRadCap) > 0 {
		if ngapMsg.UERadioCapability, err = hex.DecodeString(sbiMsg.UeRadCap); err != nil {
			return utils.WrapError("Hex convert UERadioCapability into binary", err)
		}
	}

	// Core Network Assistance Information (optional)
	// Trace Activation (optional)
	// Mobility Restriction List (optional)
	// Masked IMEISV (optional)
	// TS 38.413 9.3.1.54; TS 23.003 6.2; TS 23.501 5.9.3
	// last 4 digits of the SNR masked by setting the corresponding bits to 1.
	// The first to fourth bits correspond to the first digit of the IMEISV,
	// the fifth to eighth bits correspond to the second digit of the IMEISV, and so on
	// Emergency Fallback indicator (optional)
	// RRC Inactive Transition Report Request (optional)
	// UE Radio Capability for Paging (optional)
	// Index to RAT/Frequency Selection Priority (optional)

	if sbiMsg.Rfsp != nil {
		ngapMsg.IndexToRFSP = sbiMsg.Rfsp
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

// 8.3.3
func (ueCtx *UeContext) sendUeContextReleaseCommand(sbiMsg *models.UeContextReleaseCommand) error {
	ngapMsg := &ies.UEContextReleaseCommand{
		UENGAPIDs: ies.UENGAPIDs{
			Choice: ies.UENGAPIDsPresentUeNgapIdPair,
			UENGAPIDpair: &ies.UENGAPIDpair{
				AMFUENGAPID: ueCtx.cuNgapId(),
				RANUENGAPID: ueCtx.ranNgapId,
			},
		},
		Cause: ngapCause(sbiMsg.Cause),
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

// 8.3.4
func (ueCtx *UeContext) sendUeContextModificationRequest(sbiMsg *models.UeContextModifyRequest) error {
	ngapMsg := &ies.UEContextModificationRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}
	// Ran Paging Priority (optional)
	// Security Key (optional)
	// Index to RAT/Frequency Selection Priority (optional)
	if sbiMsg.Rfsp != nil {
		ngapMsg.IndexToRFSP = sbiMsg.Rfsp
	}
	// UE Aggregate Maximum Bit Rate (optional)
	ngapMsg.UEAggregateMaximumBitRate = ngapUeAmbr(sbiMsg.UeAmbr)
	// UE Security Capabilities (optional)
	// Core Network Assistance Information (optional)
	// Emergency Fallback Indicator (optional)
	// New AMF UE NGAP ID (optional)
	if sbiMsg.OldAmfNgapId != nil {
		ngapMsg.NewAMFUENGAPID = newInt64(ueCtx.cuNgapId())
	}
	// RRC Inactive Transition Report Request (optional)
	if sbiMsg.RrcStatusReport != nil {
		ngapMsg.RRCInactiveTransitionReportRequest = &ies.RRCInactiveTransitionReportRequest{
			Value: aper.Enumerated(*sbiMsg.RrcStatusReport),
		}
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

/**************** OTHER messages *****************/

// AOI List is from SMF
// The SMF may subscribe to the UE mobility event notification from the AMF
// (e.g. location reporting, UE moving into or out of Area Of Interest) TS 23.502 4.3.2.2.1 Step.17
// The Location Reporting Control message shall identify the UE for which reports are requested and may include
// Reporting Type, Location Reporting Level, Area Of Interest and Request Reference ID
// TS 23.502 4.10 LocationReportingProcedure
// The AMF may request the NG-RAN location reporting with event reporting type (e.g. UE location or UE presence
// in Area of Interest), reporting mode and its related parameters (e.g. number of reporting) TS 23.501 5.4.7
// Location Reference ID To Be Cancelled IE shall be present if the Event Type IE is set to "Stop UE presence
// in the area of interest". otherwise set it to 0
func (ueCtx *UeContext) sendLocationReportingControl(
	AOIList []ies.AreaOfInterestItem,
	LocationReportingReferenceIDToBeCancelled int64,
	eventType ies.EventType) error {

	if eventType.Value == ies.EventTypeStopuepresenceinareaofinterest {
		if LocationReportingReferenceIDToBeCancelled < 1 || LocationReportingReferenceIDToBeCancelled > 64 {
			return fmt.Errorf("LocationReportingReferenceIDToBeCancelled out of range (should be 1 ~ 64)")
		}
	}
	ngapMsg := &ies.LocationReportingControl{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
		LocationReportingRequestType: ies.LocationReportingRequestType{
			EventType: eventType,
			ReportArea: ies.ReportArea{
				Value: ies.ReportAreaCell,
			},
			AreaOfInterestList: AOIList,
		},
	}

	// location reference ID to be Cancelled [Conditional]
	if eventType.Value == ies.EventTypeStopuepresenceinareaofinterest {
		ngapMsg.LocationReportingRequestType.LocationReportingReferenceIDToBeCancelled = newInt64(LocationReportingReferenceIDToBeCancelled)
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

// NRPPa PDU is a pdu from LMF to RAN defined in TS 23.502 4.13.5.5 step 3
// NRPPa PDU is by pass
func (ueCtx *UeContext) sendDownlinkUEAssociatedNRPPaTransport(rId string, nRPPaPDU []byte) error {

	if len(nRPPaPDU) == 0 {
		return fmt.Errorf("Length of NRPPA-PDU is 0")
	}
	ngapMsg := &ies.DownlinkUEAssociatedNRPPaTransport{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
		NRPPaPDU:    nRPPaPDU,
	}

	// Routing ID
	routingId, err := hex.DecodeString(rId)
	if err != nil {
		return utils.WrapError("Decode ue routing id", err)
	}

	ngapMsg.RoutingID = []byte(routingId)

	return ueCtx.sendNgapMsg(ngapMsg)
}

// NRPPa PDU is by pass
// NRPPa PDU is from LMF define in 4.13.5.6
func (ueCtx *UeContext) sendDownlinkNonUEAssociatedNRPPATransport(rId string, nRPPaPDU []byte) error {

	if len(nRPPaPDU) == 0 {
		return fmt.Errorf("Length of NRPPA-PDU is 0")
	}
	ngapMsg := &ies.DownlinkNonUEAssociatedNRPPaTransport{
		NRPPaPDU: nRPPaPDU,
	}

	// Routing ID
	routingId, err := hex.DecodeString(rId)
	if err != nil {
		return utils.WrapError("Decode ue routing id", err)
	}

	ngapMsg.RoutingID = routingId

	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) sendUERadioCapabilityCheckRequest() error {

	ngapMsg := &ies.UERadioCapabilityCheckRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}
	// TODO:UE Radio Capability(optional)

	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) SendUETNLABindingReleaseRequest() error {

	ngapMsg := &ies.UETNLABindingReleaseRequest{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) sendDeactivateTrace(anType models.AccessType) error {
	ngapMsg := &ies.DeactivateTrace{
		AMFUENGAPID: ueCtx.cuNgapId(),
		RANUENGAPID: ueCtx.ranNgapId,
	}

	//TODO: add more info
	return ueCtx.sendNgapMsg(ngapMsg)
}

func (ueCtx *UeContext) sendNgapMsg(msg ngap.NgapMessageEncoder) error {
	if packet, err := ngap.NgapEncode(msg); err == nil {
		if err = ueCtx.ran.Send(packet); err != nil {
			err = utils.WrapError("Send encoded NGAP message", err)
		}
	} else {
		return utils.WrapError("Encode NGAP message", err)
	}
	return nil
}
