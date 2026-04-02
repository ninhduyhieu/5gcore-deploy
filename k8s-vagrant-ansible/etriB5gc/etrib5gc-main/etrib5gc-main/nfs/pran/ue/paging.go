package ue

import ()

// anType indicate amfUe send this msg for which accessType
// Paging Priority: is included only if the AMF receives an Namf_Communication_N1N2MessageTransfer message
// with an ARP value associated with
// priority services (e.g., MPS, MCS), as configured by the operator. (TS 23.502 4.2.3.3, TS 23.501 5.22.3)
// pagingOriginNon3GPP: TS 23.502 4.2.3.3 step 4b: If the UE is simultaneously registered over 3GPP and non-3GPP
// accesses in the same PLMN,
// the UE is in CM-IDLE state in both 3GPP access and non-3GPP access, and the PDU Session ID in step 3a
// is associated with non-3GPP access, the AMF sends a Paging message with associated access "non-3GPP" to
// NG-RAN node(s) via 3GPP access.
// more paging policy with 3gpp/non-3gpp access is described in TS 23.501 5.6.8
func SendPaging(ueCtx *UeContext, ngapBuf []byte) (err error) {
	// var pagingPriority *ngapType.PagingPriority

	// if ppi != nil {
	// pagingPriority = new(ngapType.PagingPriority)
	// pagingPriority.Value = aper.Enumerated(*ppi)
	// }
	// pkt, err := s.buildPaging(ue, pagingPriority, pagingOriginNon3GPP)
	// if err != nil {
	// 	ngaplog.Errorf("Build Paging failed : %s", err.Error())
	// }
	/*
		amf := s.backend.Context()
		ranpool := amf.AmfRanPool()
		taiList := ue.RegistrationArea[models.ACCESSTYPE_3GPP_ACCESS]
		ranpool.Range(func(key, value interface{}) bool {
			ran := value.(Ran)
			for _, item := range ran.SupportedTAList() {
				if context.InTaiList(item.Tai, taiList) {
					log.Infof("Send Paging to TAI(%+v, Tac:%+v)",
						item.Tai.PlmnId, item.Tai.Tac)
					SendToRan(ran, ngapBuf)
					break
				}
			}
			return true
		})

		cfg := amf.T3513Cfg()
		if cfg.Enable {
			ue.T3513 = context.NewTimer(cfg.ExpireTime, cfg.MaxRetryTimes, func(expireTimes int32) {
				log.Warnf("T3513 expires, retransmit Paging (retry: %d)", expireTimes)
				ranpool.Range(func(key, value interface{}) bool {
					ran := value.(Ran)
					for _, item := range ran.SupportedTAList() {
						if context.InTaiList(item.Tai, taiList) {
							SendToRan(ran, ngapBuf)
							break
						}
					}
					return true
				})
			}, func() {
				log.Warnf("T3513 expires %d times, abort paging procedure", cfg.MaxRetryTimes)
				ue.T3513 = nil // clear the timer
				if ue.OnGoing(models.ACCESSTYPE_3GPP_ACCESS).Procedure != context.OnGoingProcedureN2Handover {

					ue.CallbackClient().SendN1N2TransferFailureNotification(models.N1N2MESSAGETRANSFERCAUSE_UE_NOT_RESPONDING)
				}
			})
		}
	*/
	return
}

/*
func buildPaging(
	ueCtx *UeContext, pagingPriority *ngapType.PagingPriority, pagingOriginNon3GPP bool) ([]byte, error) {
		// TODO: Paging DRX (optional)

		var pdu ngapType.NGAPPDU
		pdu.Present = ngapType.NGAPPDUPresentInitiatingMessage
		pdu.InitiatingMessage = new(ngapType.InitiatingMessage)

		initiatingMessage := pdu.InitiatingMessage
		initiatingMessage.ProcedureCode.Value = ngapType.ProcedureCodePaging
		initiatingMessage.Criticality.Value = ngapType.CriticalityPresentIgnore

		initiatingMessage.Value.Present = ngapType.InitiatingMessagePresentPaging
		initiatingMessage.Value.Paging = new(ngapType.Paging)

		paging := initiatingMessage.Value.Paging
		pagingIEs := &paging.ProtocolIEs

		// UE Paging Identity
		ie := ngapType.PagingIEs{}
		ie.Id.Value = ngapType.ProtocolIEIDUEPagingIdentity
		ie.Criticality.Value = ngapType.CriticalityPresentIgnore
		ie.Value.Present = ngapType.PagingIEsPresentUEPagingIdentity
		ie.Value.UEPagingIdentity = new(ngapType.UEPagingIdentity)

		uePagingIdentity := ie.Value.UEPagingIdentity
		uePagingIdentity.Present = ngapType.UEPagingIdentityPresentFiveGSTMSI
		uePagingIdentity.FiveGSTMSI = new(ngapType.FiveGSTMSI)

		var amfID string
		var tmsi string
		if len(ue.Guti) == 19 {
			amfID = ue.Guti[5:11]
			tmsi = ue.Guti[11:]
		} else {
			amfID = ue.Guti[6:12]
			tmsi = ue.Guti[12:]
		}
		_, amfSetID, amfPointer := ngapconv.AmfIdToNgap(amfID)

		var err error
		uePagingIdentity.FiveGSTMSI.AMFSetID.Value = amfSetID
		uePagingIdentity.FiveGSTMSI.AMFPointer.Value = amfPointer
		uePagingIdentity.FiveGSTMSI.FiveGTMSI.Value, err = hex.DecodeString(tmsi)
		if err != nil {
			//logger.NgapLog.Errorf("[Build Error] DecodeString tmsi error: %+v", err)
		}

		pagingIEs.List = append(pagingIEs.List, ie)

		// Paging DRX (optional)

		// TAI List for Paging
		ie = ngapType.PagingIEs{}
		ie.Id.Value = ngapType.ProtocolIEIDTAIListForPaging
		ie.Criticality.Value = ngapType.CriticalityPresentIgnore
		ie.Value.Present = ngapType.PagingIEsPresentTAIListForPaging
		ie.Value.TAIListForPaging = new(ngapType.TAIListForPaging)
		// tungtq
		taiListForPaging := ie.Value.TAIListForPaging
		if ue.RegistrationArea[models.ACCESSTYPE_3GPP_ACCESS] == nil {
			err = fmt.Errorf("Registration Area of Ue[%s] is empty", ue.Supi)
			return nil, err
		} else {
			for _, tai := range ue.RegistrationArea[models.ACCESSTYPE_3GPP_ACCESS] {
				var tac []byte
				taiListforPagingItem := ngapType.TAIListForPagingItem{}
				taiListforPagingItem.TAI.PLMNIdentity = ngapconv.PlmnIdToNgap(*tai.PlmnId)
				tac, err = hex.DecodeString(tai.Tac)
				if err != nil {
					//logger.NgapLog.Errorf("[Build Error] DecodeString tai.Tac error: %+v", err)
				}
				taiListforPagingItem.TAI.TAC.Value = tac
				taiListForPaging.List = append(taiListForPaging.List, taiListforPagingItem)
			}
		}
		//tungtq end
		pagingIEs.List = append(pagingIEs.List, ie)

		// Paging Priority (optional)
		if pagingPriority != nil {
			ie = ngapType.PagingIEs{}
			ie.Id.Value = ngapType.ProtocolIEIDPagingPriority
			ie.Criticality.Value = ngapType.CriticalityPresentIgnore
			ie.Value.Present = ngapType.PagingIEsPresentPagingPriority
			ie.Value.PagingPriority = pagingPriority
			pagingIEs.List = append(pagingIEs.List, ie)
		}

		// UE Radio Capability for Paging (optional)
		if ue.UeRadioCapabilityForPaging != nil {
			ie = ngapType.PagingIEs{}
			ie.Id.Value = ngapType.ProtocolIEIDUERadioCapabilityForPaging
			ie.Criticality.Value = ngapType.CriticalityPresentIgnore
			ie.Value.Present = ngapType.PagingIEsPresentUERadioCapabilityForPaging
			ie.Value.UERadioCapabilityForPaging = new(ngapType.UERadioCapabilityForPaging)
			uERadioCapabilityForPaging := ie.Value.UERadioCapabilityForPaging
			if ue.UeRadioCapabilityForPaging.NR != "" {
				uERadioCapabilityForPaging.UERadioCapabilityForPagingOfNR.Value, err =
					hex.DecodeString(ue.UeRadioCapabilityForPaging.NR)
				if err != nil {
					//logger.NgapLog.Errorf(
					//	"[Build Error] DecodeString ue.UeRadioCapabilityForPaging.NR error: %+v", err)
				}
			}
			if ue.UeRadioCapabilityForPaging.EUTRA != "" {
				uERadioCapabilityForPaging.UERadioCapabilityForPagingOfEUTRA.Value, err =
					hex.DecodeString(ue.UeRadioCapabilityForPaging.EUTRA)
				if err != nil {
					//logger.NgapLog.Errorf("[Build Error] DecodeString ue.UeRadioCapabilityForPaging.EUTRA error: %+v", err)
				}
			}
			pagingIEs.List = append(pagingIEs.List, ie)
		}

		// Assistance Data for Paing (optional)
		if ue.InfoOnRecommendedCellsAndRanNodesForPaging != nil {
			ie = ngapType.PagingIEs{}
			ie.Id.Value = ngapType.ProtocolIEIDAssistanceDataForPaging
			ie.Criticality.Value = ngapType.CriticalityPresentIgnore
			ie.Value.Present = ngapType.PagingIEsPresentAssistanceDataForPaging
			ie.Value.AssistanceDataForPaging = new(ngapType.AssistanceDataForPaging)

			assistanceDataForPaging := ie.Value.AssistanceDataForPaging
			assistanceDataForPaging.AssistanceDataForRecommendedCells =
				new(ngapType.AssistanceDataForRecommendedCells)
			recommendedCellList := &assistanceDataForPaging.
				AssistanceDataForRecommendedCells.RecommendedCellsForPaging.RecommendedCellList

			for _, recommendedCell := range ue.InfoOnRecommendedCellsAndRanNodesForPaging.RecommendedCells {
				recommendedCellItem := ngapType.RecommendedCellItem{}
				switch recommendedCell.NgRanCGI.Present {
				case context.NgRanCgiPresentNRCGI:
					recommendedCellItem.NGRANCGI.Present = ngapType.NGRANCGIPresentNRCGI
					recommendedCellItem.NGRANCGI.NRCGI = new(ngapType.NRCGI)
					nrCGI := recommendedCellItem.NGRANCGI.NRCGI
					//nrCGI.PLMNIdentity =
					//ngapconv.PlmnIdToNgap(*recommendedCell.NgRanCGI.NRCGI.PlmnId)
					//tungtq
					nrCGI.NRCellIdentity.Value = ngapconv.HexToBitString(recommendedCell.NgRanCGI.NRCGI.NrCellId, 36)
				case context.NgRanCgiPresentEUTRACGI:
					recommendedCellItem.NGRANCGI.Present = ngapType.NGRANCGIPresentEUTRACGI
					recommendedCellItem.NGRANCGI.EUTRACGI = new(ngapType.EUTRACGI)
					eutraCGI := recommendedCellItem.NGRANCGI.EUTRACGI
					//eutraCGI.PLMNIdentity =
					//ngapconv.PlmnIdToNgap(*recommendedCell.NgRanCGI.EUTRACGI.PlmnId)//tungtq
					eutraCGI.EUTRACellIdentity.Value =
						ngapconv.HexToBitString(recommendedCell.NgRanCGI.EUTRACGI.EutraCellId, 28)
				}

				if recommendedCell.TimeStayedInCell != nil {
					recommendedCellItem.TimeStayedInCell = recommendedCell.TimeStayedInCell
				}
				recommendedCellList.List = append(recommendedCellList.List, recommendedCellItem)
			}

			// TODO: Paging Attempt Information (optional): provided by AMF (TS 23.502 4.2.3.3, TS 38.300 9.2.5)
			pagingIEs.List = append(pagingIEs.List, ie)
		}

		// Paging Origin (optional)
		if pagingOriginNon3GPP {
			ie = ngapType.PagingIEs{}
			ie.Id.Value = ngapType.ProtocolIEIDPagingOrigin
			ie.Criticality.Value = ngapType.CriticalityPresentIgnore
			ie.Value.Present = ngapType.PagingIEsPresentPagingOrigin
			ie.Value.PagingOrigin = new(ngapType.PagingOrigin)
			ie.Value.PagingOrigin.Value = ngapType.PagingOriginPresentNon3gpp
			pagingIEs.List = append(pagingIEs.List, ie)
		}

		return libngap.Encoder(pdu)
	return nil, nil
}
*/
