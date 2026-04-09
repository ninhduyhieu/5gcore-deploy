package uecontext

import (
	"etrib5gc/common"
	"etrib5gc/mesh"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/sm"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

// receive notification of SmContext deletion from SMF
func (ueCtx *UeContext) HandleSmContextStatusNotify(sessionId uint8, msg *models.SmContextStatusNotification) error {
	if smCtx := ueCtx.findSmContext(sessionId); smCtx == nil {
		return fmt.Errorf("SmContext not found for session %d", sessionId)
	} else {
		ueCtx.Infof("SMF notify of SmContext deletion [%d]", sessionId)
		ueCtx.sessions.remove(sessionId)
	}
	return nil
}

func (ueCtx *UeContext) createSmContextCreateData(isGpp bool) *models.SmContextCreateData {
	plmnId := amfctx.PlmnId()
	dat := &models.SmContextCreateData{
		Supi: ueCtx.supi,
		Pei:  ueCtx.pei,
		ServingNetwork: models.PlmnIdNid{
			Mnc: plmnId.Mnc,
			Mcc: plmnId.Mcc,
		},
	}
	//dat.Gpsi =ueCtx.gpsi
	dat.AnType, dat.RanTransportNets, dat.RanUeInfo = ueCtx.getRanUeInfo(isGpp)
	//dat.ServingNfId =context.NfId
	//dat.Guami =&context.ServedGuamiList[0]
	return dat
}

// create SmContext, ask SMF to create SmContext if needs and forward N1Sm
// error means N1Sm was not forward (and SmContext was not created)
func (ueCtx *UeContext) createSmContext(isGpp bool, msg *nas.UlNasTransport, preExist bool) error {
	sessionId := uint8(*msg.PduSessionId)
	//make sure session id is valid
	if sessionId == 0 || sessionId >= MAX_PDU_SESSIONS {
		return fmt.Errorf("Invalid session number[1-15]: %d", sessionId)
	}

	if preExist { //SmContext existed at an SMF, load SmContext information (from UDM) then forward N1Sm to the SMF
		ueCtx.Tracef("Load smcontext[id=%d] from data", sessionId)
		if smCtx, err := ueCtx.loadSmContext(sessionId, isGpp); err != nil {
			return utils.WrapError("Load SmContext from existing", err)
		} else {
			//add SmContext
			ueCtx.addSmContext(smCtx)
			ueCtx.forwardN1Sm(smCtx, msg.PayloadContainer)
			return nil
		}
	}
	//SmContext does not exist, create one locally then ask SMF to create
	//(forward N1SM too)
	var requestedSnssai *models.Snssai
	if msg.SNssai != nil {
		requestedSnssai = &models.Snssai{
			Sst: int(msg.SNssai.Sst),
			Sd:  msg.SNssai.GetSd(),
		}
	}
	if cli, snssai, err := ueCtx.selectSmf(requestedSnssai, isGpp); err != nil {
		return utils.WrapError("Select SMF", err)
	} else {
		ueCtx.Infof("SMF was selected (snssai=%s) for session %d", snssai.AllowedSnssai.String(), sessionId)
		var dnn string
		if msg.Dnn != nil { //a Dnn is specified for the session
			dnn = msg.Dnn.String()
			ueCtx.Infof("UE requested DNN %s for session %d", dnn, sessionId)
		} else {
			dnn = ueCtx.findDnn(&snssai.AllowedSnssai, isGpp)
		}
		isHomeRouted := false
		if hSlice := snssai.MappedHomeSnssai; hSlice != nil && amfctx.IsHomeRouted(*ueCtx.plmnId, *hSlice) {
			isHomeRouted = true
		}
		//finally create the SmContext
		smCtx := sm.NewSmContext(ueCtx.Entry, isGpp, sessionId, *snssai, isHomeRouted, dnn, cli)
		var rsp *models.PostSmContextsResponse
		var ersp *models.PostSmContextsErrorResponse
		//ask SMF to create context and handle the N1Sm in a goroutine
		if rsp, ersp, err = smCtx.CreateContextAtSmf(ueCtx.createSmContextCreateData(isGpp), msg.PayloadContainer); err != nil {
			return utils.WrapError("Create SmContext at SMF", err)
		} else if ersp != nil {
			//NOTE: there can be N2SmInfo and N1Sm, it seems the specification
			//is muddy here, we does not handle this case. We just assume that
			//the SMF fail to create SmContext and ignore any N1/N2 message
			if ersp.JsonData != nil && len(ersp.JsonData.Error.Detail) > 0 {
				return fmt.Errorf("N1Sm was forwarded but SmContext was not created at SMF: %s", ersp.JsonData.Error.Detail)
			} else {
				return fmt.Errorf("N1Sm was forwarded but SmContext was not created at SMF")
			}
		} else if rsp != nil {
			ueCtx.addSmContext(smCtx)
			ueCtx.Infof("N1Sm was forwarded and SmContext was created at SMF")

			n2SmInfo := rsp.BinaryDataN2SmInformation
			n2SmInfoType := rsp.JsonData.N2SmInfoType
			//send downlink responses (N1,N2) from SMF toward RAN
			if err = ueCtx.sendN2SmInfoDownlink(smCtx.IsGpp(), smCtx, []byte{}, n2SmInfo, n2SmInfoType); err != nil {
				ueCtx.Errorf("Fail to send N1N2 from PostSmContextsRequest response downlink: %+v", err)
			}
		}
	}
	return nil
}

func (ueCtx *UeContext) connectExistingSmf(plmnId models.PlmnId, smfInsId string) (sbi.ConsumerClient, error) {
	//TODO
	return nil, fmt.Errorf("Connect existing SMF not implemted")
}

func (ueCtx *UeContext) selectSmf(reqSlice *models.Snssai, isGpp bool) (sbi.ConsumerClient, *models.AllowedSnssai, error) {
	slices := ueCtx.getAllowedSlices(isGpp)
	var snssai *models.AllowedSnssai
	if reqSlice != nil {
		// get allowes Snssai for a home slice
		for _, s := range slices {
			if common.IsSliceEqual(&s.AllowedSnssai, reqSlice) {
				//NOTE: UERANSIM not yet support roaming, it wil send home SNSSAI
				snssai = &s
				break
			}
		}
	}

	if snssai == nil {
		//request slice is not in the list of allowed slices, then pick one of
		//the allowed slice
		for _, s := range slices {
			if cli, err := ueCtx.createSmfClient(&s); err == nil {
				return cli, &s, err
			}
		}
		return nil, nil, fmt.Errorf("Can't select SMF")
	} else {
		if cli, err := ueCtx.createSmfClient(snssai); err != nil {
			return nil, nil, utils.WrapError("Create Smf client", err)
		} else {
			return cli, snssai, nil
		}
	}
}

// create Smf client
func (ueCtx *UeContext) createSmfClient(snssai *models.AllowedSnssai) (sbi.ConsumerClient, error) {
	var smfId string

	if hSlice := snssai.MappedHomeSnssai; hSlice != nil && amfctx.IsHomeRouted(*ueCtx.plmnId, *hSlice) {
		smfId = common.SmfServiceName(ueCtx.plmnId, snssai.MappedHomeSnssai) //Home-routed
	} else {
		smfId = common.SmfServiceName(amfctx.PlmnId(), &snssai.AllowedSnssai) //Local-Break-Out
	}
	ueCtx.Debugf("Create SMF client to %s", smfId)
	options := ueCtx.smfSelectionOptions()
	if cli, err := mesh.Consumer(smfId, options); err != nil {
		return nil, utils.WrapError("Create sbi consumer client", err)
	} else {
		return cli, nil
	}
}

// create a SmContext from Ue context data in Smf received from Udm
func (ueCtx *UeContext) loadSmContext(id uint8, isGpp bool) (*sm.SmContext, error) {
	if ueCtx.ueInSmf == nil {
		return nil, fmt.Errorf("No UeInSmf data")
	}

	if session, ok := ueCtx.ueInSmf.PduSessions[fmt.Sprintf("%d", id)]; ok {
		if cli, err := ueCtx.connectExistingSmf(session.PlmnId, session.SmfInstanceId); err != nil {
			return nil, utils.WrapError("Connect existing SMF", err)
		} else {
			plmnId := amfctx.PlmnId()
			var snssai *models.AllowedSnssai
			if session.SingleNssai != nil {
				if snssai = ueCtx.getAllowedSnssai(session.SingleNssai, isGpp); snssai == nil {
					return nil, fmt.Errorf("Slice [%s] is not allowed", session.SingleNssai.String())
				}
			} else {
				//here we mandate the the existing session must have an
				//assigned slice
				return nil, fmt.Errorf("Session has no slice information")
			}

			dnn := session.Dnn
			if len(dnn) == 0 {
				dnn = ueCtx.findDnn(snssai.MappedHomeSnssai, isGpp)
			}
			//if SMF's plmn Id is different from serving network then the
			//session is home routed
			isHomeRouted := plmnId.Mnc != session.PlmnId.Mnc || plmnId.Mcc != session.PlmnId.Mcc

			return sm.NewSmContext(ueCtx.Entry, isGpp, id, *snssai, isHomeRouted, dnn, cli), nil
		}
	} else {
		return nil, fmt.Errorf("No session information")
	}
}

func (ueCtx *UeContext) releaseSmContext(smCtx *sm.SmContext, cause models.Cause) {
	id := smCtx.GetId()
	if rsp, err := smCtx.ReleaseSmContext(&models.SmContextReleaseData{
		Cause:             cause,
		NgApCause:         &models.NgApCause{},
		FiveGMmCauseValue: nil,
	}, "", nil); err != nil {
		ueCtx.Errorf("Smf fails to release SmContext %d: %+v", id, err)
	} else {
		ueCtx.Warnf("SmContextReleasedData for session %d is not handled", id)
		_ = rsp //TODO: handle response
	}
	ueCtx.deleteSmContext(id)
}

// change access type during mobility update/reconnecting
func (ueCtx *UeContext) changeSessionAccess(smCtx *sm.SmContext, report *SyncPduSessionReport) {
	id := smCtx.GetId()
	msg := &models.SmContextUpdateData{
		AnTypeCanBeChanged: new(bool),
	}
	*msg.AnTypeCanBeChanged = true

	if rsp, ersp, err := smCtx.SendUpdateSmContext(msg, nil, nil); err != nil {
		report.ReactList[id] = true
		report.ErrPduList = append(report.ErrPduList, id)
		report.ErrCauses = append(report.ErrCauses, nas.Cause5GMMProtocolErrorUnspecified)
	} else {
		if n2, n1 := ueCtx.processUpdateSmContextResponses(smCtx, rsp, ersp); n2 != nil {
			report.N2SmInfoList = append(report.N2SmInfoList, *n2)
		} else if len(n1) > 0 {
			report.N1MsgList = append(report.N1MsgList, n1)
		}
		if ersp != nil {
			report.ReactList[id] = true
			report.ErrPduList = append(report.ErrPduList, id)
			cause := nas.Cause5GMMProtocolErrorUnspecified

			switch ersp.JsonData.Error.Cause {
			case "OUT_OF_LADN_SERVICE_AREA":
				cause = nas.Cause5GMMLADNNotAvailable
			case "PRIORITIZED_SERVICES_ONLY":
				cause = nas.Cause5GMMRestrictedServiceArea
			case "DNN_CONGESTION", "S-NSSAI_CONGESTION":
				cause = nas.Cause5GMMInsufficientUserPlaneResourcesForThePDUSession
			}
			report.ErrCauses = append(report.ErrCauses, cause)
		}

	}
}

// reactive session during mobility update/reconnecting
func (ueCtx *UeContext) reactivateSession(smCtx *sm.SmContext, report *SyncPduSessionReport) {
	id := smCtx.GetId()
	msg := &models.SmContextUpdateData{
		UpCnxState: models.UPCNXSTATE_ACTIVATING,
		//TODO: add access, location, etc
	}
	if rsp, ersp, err := smCtx.SendUpdateSmContext(msg, nil, nil); err != nil {
		ueCtx.Errorf("Fail to activate session %d: %+v", id, err)
		report.ErrPduList = append(report.ErrPduList, id)
		report.ErrCauses = append(report.ErrCauses, nas.Cause5GMMProtocolErrorUnspecified)
		report.ReactList[id] = true
	} else {
		if ersp != nil {
			report.ErrPduList = append(report.ErrPduList, id)
			report.ErrCauses = append(report.ErrCauses, nas.Cause5GMMProtocolErrorUnspecified)
			report.ReactList[id] = true
		}
		if n2, n1 := ueCtx.processUpdateSmContextResponses(smCtx, rsp, ersp); n2 != nil {
			report.N2SmInfoList = append(report.N2SmInfoList, *n2)
		} else if len(n1) > 0 {
			report.N1MsgList = append(report.N1MsgList, n1)
		}

	}
}
