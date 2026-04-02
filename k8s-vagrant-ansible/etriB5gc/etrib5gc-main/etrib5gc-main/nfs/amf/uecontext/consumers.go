package uecontext

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"etrib5gc/mesh"
	amfctx "etrib5gc/nfs/amf/context"
	"fmt"
	"github.com/reogac/sbi/apis/pcf/ampol"
	"github.com/reogac/sbi/apis/udm/sdm"
	"github.com/reogac/sbi/apis/udm/uecm"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func (ueCtx *UeContext) connectUdm() (err error) {
	sid := common.UdmServiceName(ueCtx.plmnId)
	if ueCtx.udmCli, err = mesh.Consumer(sid, ueCtx.metadata); err != nil {
		return utils.WrapError("Create Udm consumer", err)
	}

	return nil
}

func (ueCtx *UeContext) connectPcf() (err error) {
	sid := common.PcfServiceName(ueCtx.plmnId)
	if ueCtx.pcfCli, err = mesh.Consumer(sid, ueCtx.metadata); err != nil {
		err = utils.WrapError("Create Udm consumer", err)
	}

	return
}

func (ueCtx *UeContext) getUeContextInSmfData() (err error) {
	ueCtx.Debugf("Get UeContextInSmf from UDM")
	params := sdm.GetUeCtxInSmfDataParams{
		Supi: ueCtx.supi,
	}
	if ueCtx.ueInSmf, err = sdm.GetUeCtxInSmfData(ueCtx.udmCli, params); err != nil {
		return utils.WrapError("Send GetUeCtxInSmfData request to UDM", err)
	}

	return nil
}

func (ueCtx *UeContext) getSmfSelectionData() (err error) {
	ueCtx.Debugf("Get SmfSelectionData from UDM")
	params := sdm.GetSmfSelDataParams{
		PlmnId: &models.PlmnId{
			Mcc: ueCtx.plmnId.Mcc,
			Mnc: ueCtx.plmnId.Mnc,
		},
		Supi: ueCtx.supi,
	}
	if _, ueCtx.smfSel, err = sdm.GetSmfSelData(ueCtx.udmCli, params); err != nil {
		return utils.WrapError("Send GetSmData request to UDM", err)
	} //TODO: process response headers

	return nil
}

func (ueCtx *UeContext) getSmData() (err error) {
	ueCtx.Debugf("Get SmData from UDM")
	params := sdm.GetSmDataParams{
		PlmnId: &models.PlmnId{
			Mcc: ueCtx.plmnId.Mcc,
			Mnc: ueCtx.plmnId.Mnc,
		},
		Supi: ueCtx.supi,
	}
	if _, ueCtx.smData, err = sdm.GetSmData(ueCtx.udmCli, params); err != nil {
		return utils.WrapError("Send GetSmData request to UDM", err)
	} //TODO: process response headers
	return nil
}

func (ueCtx *UeContext) getAmData() (err error) {
	ueCtx.Infof("Get AmData from UDM")
	params := sdm.GetAmDataParams{
		PlmnId: &models.PlmnIdNid{
			Mcc: ueCtx.plmnId.Mcc,
			Mnc: ueCtx.plmnId.Mnc,
		},
		Supi: ueCtx.supi,
	}
	if _, ueCtx.amData, err = sdm.GetAmData(ueCtx.udmCli, params); err != nil {
		return utils.WrapError("Send GetAmData request to UDM", err)
	} //TODO: process response headers
	return
}

func (ueCtx *UeContext) notifyRegistration(isGpp bool) {
	ueCtx.Debugf("Notify UDM of UE's registration success")
	var err error
	if isGpp {
		_, err = uecm.ThreeGppRegistration(ueCtx.udmCli, ueCtx.supi, &models.Amf3GppAccessRegistration{
			Supi:             ueCtx.supi,
			Guami:            amfctx.GetGuami(),
			AmfInstanceId:    "TODO",
			DeregCallbackUri: "TODO",
			RatType:          "TODO",
		})
	} else {
		_, err = uecm.Non3GppRegistration(ueCtx.udmCli, ueCtx.supi, &models.AmfNon3GppAccessRegistration{
			Supi:             ueCtx.supi,
			Guami:            amfctx.GetGuami(),
			AmfInstanceId:    "TODO",
			DeregCallbackUri: "TODO",
			RatType:          "TODO",
		})
	}

	if err != nil {
		//deregiser UE if failed to notify UDM
		trigger := &DeregistrationTrigger{
			retry: false,
		}
		if isGpp {
			trigger.targetRan = TARGET_GPP
		} else {
			trigger.targetRan = TARGET_NON_GPP
		}
		ueCtx.sendEvent(context.TODO(), fsm.NewEventData(DeregisterEvent, trigger))
	}
}

func (ueCtx *UeContext) notifyUdmOfDeregistration() error {
	ueCtx.Debugf("Notify UDM of deregistration status")
	if err := uecm.DeregAMF(ueCtx.udmCli, ueCtx.supi, &models.AmfDeregInfo{
		DeregReason: models.DEREGISTRATIONREASON_UE_INITIAL_REGISTRATION,
	}); err != nil {
		return utils.WrapError("Send UE's AMF deregistration to UDM", err)
	}
	return nil
}

func (ueCtx *UeContext) createAmPolicyAssociation(isGpp bool) (err error) {
	ueCtx.Debugf("Send an AmPolicyAssociation request to PCF")
	plmnId := amfctx.PlmnId()
	request := &models.PolicyAssociationRequest{
		AccessType: getAccessType(isGpp),
		Supi:       ueCtx.supi,
		Pei:        ueCtx.pei,
		Gpsi:       ueCtx.gpsi,
		ServingPlmn: &models.PlmnIdNid{
			Mcc: plmnId.Mcc,
			Mnc: plmnId.Mnc,
		},
	}
	var amPolicy *models.PolicyAssociation
	var headers map[string]string
	if headers, amPolicy, err = ampol.CreateIndividualAMPolicyAssociation(ueCtx.pcfCli, request); err == nil {
		ueCtx.amPolicy = amPolicy
		if ueCtx.amPolicyId = extractAmPolId(headers); len(ueCtx.amPolicyId) == 0 {
			return fmt.Errorf("PCF send no AmPolicyId")
		}
	} else {
		return utils.WrapError("Send AmPolicyAssociation request to PCF", err)
	}

	return
}

func (ueCtx *UeContext) terminateAmPol() error {
	ueCtx.Debugf("Terminate AmPolicy")
	if ueCtx.amPolicy == nil || len(ueCtx.amPolicyId) == 0 {
		return nil
	}
	if err := ampol.DeleteIndividualAMPolicyAssociation(ueCtx.pcfCli, ueCtx.amPolicyId); err != nil {
		return utils.WrapError("Terminate AmPolicy", err)
	}
	return nil
}
