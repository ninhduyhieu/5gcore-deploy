package uecontext

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/procs/iden"
	"fmt"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"sync"
)

// authenticated + securitymode established, now that we need to gather all
// remaining data for UeContext
func (ueCtx *UeContext) setupContext() {
	attCtx := ueCtx.attCtx
	//1. get PEI
	if len(ueCtx.pei) == 0 {
		ueCtx.Infof("Ue has no PEI, try to get it from UE")
		idType := nas.MobileIdentity5GSTypeImei
		attCtx.proc = iden.Start(&iden.Context{
			Log:      ueCtx.Entry,
			Callback: ueCtx.onIdentificationComplete,
			IdType:   idType,
			SendNasDl: func(pdu []byte) error {
				return ueCtx.sendNas(pdu, attCtx.isGpp)
			},
			IsGpp:   attCtx.isGpp,
			NasCtx:  ueCtx.getNasContext(),
			OnT3570: ueCtx.onT3570,
		})

		return
	}

	if attCtx.serviceRequest != nil {
		attCtx.handleServiceRequest()
	} else {
		//2. check for UE mandatory information
		if ueCtx.gmmCap == nil {
			ueCtx.Errorf("Request has no Capability5GMM")
			n1Cause := nas.Cause5GMMInvalidMandatoryInformation
			ueCtx.abortAttachmentProcedure(&n1Cause)
			return
		}

		//3. handle request according to the request type
		switch attCtx.regType {
		case nas.RegistrationType5GSInitialRegistration:
			attCtx.handleInitialRegistration()

		case nas.RegistrationType5GSMobilityRegistrationUpdating:
			ueCtx.Tracef("RegistrationType: Mobility Registration Updating")
			fallthrough
		case nas.RegistrationType5GSPeriodicRegistrationUpdating:
			attCtx.handleMobilityAndPeriodicRegistrationUpdating()

		//case nas.RegistrationType5GSEmergencyRegistration:
		//case nas.RegistrationType5GSReserved:
		//		ueCtx.Tracef("RegistrationType: Reserved")
		default:
			ueCtx.Errorf("RegistrationType: unknown or not supported")
			n1Cause := nas.Cause5GMMInformationElementNonExistentOrNotImplemented
			ueCtx.abortAttachmentProcedure(&n1Cause)
		}
	}
}

func (attCtx *AttachContext) handleInitialRegistration() {
	msg := attCtx.registrationRequest
	ueCtx := attCtx.ueCtx

	// get data from PCF/UDM
	tasks := []func(){}
	wg := new(sync.WaitGroup)
	if ueCtx.amData == nil {
		tasks = append(tasks, func() {
			if err := ueCtx.getAmData(); err != nil {
				ueCtx.Errorf("Fail to get AmData: %+v", err)
			}
			wg.Done()
		})
	}
	tasks = append(tasks, func() {
		if err := ueCtx.getSmData(); err != nil {
			ueCtx.Errorf("Fail to get SmData: %+v", err)
		}
		wg.Done()
	})
	tasks = append(tasks, func() {
		if err := ueCtx.getSmfSelectionData(); err != nil {
			ueCtx.Errorf("Fail to get SmfSelectionData: %+v", err)
		}
		wg.Done()
	})
	tasks = append(tasks, func() {
		if err := ueCtx.createAmPolicyAssociation(attCtx.isGpp); err != nil {
			ueCtx.Errorf("Fail to create AmPolicyAssociation: %+v", err)
		}
		wg.Done()
	})

	wg.Add(len(tasks))
	for _, t := range tasks {
		_uePool.pubWorkers.Go(t)
	}
	wg.Wait()

	//TODO:check if we get all data from UDM and PCF

	// determines allowed slices
	if err := ueCtx.determineAllowedSlices(msg.RequestedNssai, attCtx.isGpp); err != nil {
		ueCtx.Errorf("Ue can not be served by the AMF: %+v", err)
		ueCtx.abortAttachmentProcedure(nil)
		return
	}

	// Warn of not supporting MICO
	if msg.MicoIndication != nil {
		ueCtx.Warnf("Receive MICO Indication[RAAI: %d], Not Supported", uint8(*msg.MicoIndication))
	}
	// Store last visted TAI
	ueCtx.lastTai = msg.LastVisitedRegisteredTai

	// Set DRX
	if msg.RequestedDrxParameters != nil {
		ueCtx.setDrx(uint8(*msg.RequestedDrxParameters))
	}

	_uePool.pubWorkers.Go(func() {
		ueCtx.notifyRegistration(attCtx.isGpp)
	})

	//. Allocate registration area
	if attCtx.isGpp {
		ueCtx.allocateRegistrationArea()
	}

	//Assign LADN information
	ueCtx.assignLadnInfo()

	// Send registration accept (may use ngap in Non-3GPP access)
	attCtx.accept()

}

func (attCtx *AttachContext) handleMobilityAndPeriodicRegistrationUpdating() {
	msg := attCtx.registrationRequest
	ueCtx := attCtx.ueCtx

	//1. Check UpdateType5GS
	if msg.UpdateType != nil && uint8(*msg.UpdateType) == nas.NGRanRadioCapabilityUpdateNeeded {
		ueCtx.resetUeRadioCap()
	}

	//5. Store last visited Tai
	ueCtx.lastTai = msg.LastVisitedRegisteredTai

	//6. Warn that MICO is not supported
	if msg.MicoIndication != nil {
		ueCtx.Warnf("Receive MICO Indication[RAAI: %d], Not Supported", *msg.MicoIndication)
	}

	//7. Set DRX
	if msg.RequestedDrxParameters != nil {
		ueCtx.setDrx(*msg.RequestedDrxParameters)
	}
	/*
		//NOTE: communicate with UDM if AMF relocation
		//NOTE: update policy with PCF if location change

	*/

	attCtx.report = ueCtx.sessions.sync(attCtx.isGpp, msg.UplinkDataStatus, msg.PduSessionStatus, msg.AllowedPduSessionStatus)

	//. Allocate registration area
	if attCtx.isGpp {
		ueCtx.allocateRegistrationArea()
	}
	//. Assign LADN information
	ueCtx.assignLadnInfo()

	//14. send registration accept
	attCtx.accept()

}

func (attCtx *AttachContext) handleServiceRequest() {
	msg := attCtx.serviceRequest
	ueCtx := attCtx.ueCtx

	serviceType := uint8(msg.ServiceType)

	switch serviceType {
	case nas.ServiceTypeSignalling:
		//no PDU session synchronization
		attCtx.accept()
		return
	case nas.ServiceTypeEmergencyServices, nas.ServiceTypeEmergencyServicesFallback:
		ueCtx.Warnf("emergency service is not supported")

	case nas.ServiceTypeMobileTerminatedServices: // Trigger by Network
	case nas.ServiceTypeHighPriorityAccess:

	case nas.ServiceTypeData:
		if !ueCtx.isReEstablishPduSessionAllowed() {
			ueCtx.abortAttachmentProcedure(nil)
			return
		}

	default:
		n1cause := uint8(nas.Cause5GMM5GSServicesNotAllowed)
		ueCtx.Errorf("Service Type[%d] is not supported", serviceType)
		ueCtx.abortAttachmentProcedure(&n1cause)
		return
	}
	//coordinate synchronize PDU sessions between SMFs and UE
	attCtx.report = ueCtx.sessions.sync(attCtx.isGpp, msg.UplinkDataStatus, msg.PduSessionStatus, msg.AllowedPduSessionStatus)

	attCtx.accept()
}

// handle ConfigurationUpdateComplete timer expires
func (ueCtx *UeContext) handleT3555() {
	ueCtx.Warnf("T3555 to wait for ConfigurationUpdateComplete is expired")
	if ctx := ueCtx.configCtx; ctx != nil {
		if ctx.t3555Cnt >= MAX_T3555_CNT {
			ueCtx.Errorf("ConfigurationUpdateComplete not received")
			ueCtx.configCtx = nil
			//TODO: start deregistration
		} else {
			if err := ueCtx.sendConfigurationUpdateCommand(); err != nil {
				ueCtx.Errorf("Fail to send ConfigurationUpdateCommand: %+v", err)
				//TODO: start deregistration
			}
		}

	}

}
func (ueCtx *UeContext) handleConfigurationUpdateComplete(ctx context.Context, isGpp bool, msg *nas.ConfigurationUpdateComplete) {
	ueCtx.Infof("Handle NAS ConfigurationUpdateComplete")
	if ueCtx.configCtx != nil {
		ueCtx.configCtx.t3555.Stop()
		ueCtx.configCtx = nil
	}
}

// handle RegistrationComplete timer expires
func (ueCtx *UeContext) handleT3550() {
	ueCtx.Warnf("T3550 to wait for RegistrationComplete is expired")
	attCtx := ueCtx.attCtx
	if attCtx.t3550Cnt >= MAX_T3550_CNT {
		ueCtx.Errorf("RegistrationComplete not received")
		ueCtx.abortAttachmentProcedure(nil)
	} else {
		if err := ueCtx.sendNas(attCtx.nasPdu, attCtx.isGpp); err != nil {
			ueCtx.Errorf("Fail to send RegistrationAccept: %+v", err)
			ueCtx.abortAttachmentProcedure(nil)
		} else {
			ueCtx.Infof("RegitrationAccept for UE was sent to gnB")
			attCtx.t3550.Start()
			attCtx.t3550Cnt++
		}
	}
}

func (ueCtx *UeContext) handleRegistrationComplete(ctx context.Context, isGpp bool, msg *nas.RegistrationComplete) {
	ueCtx.Infof("Handle NAS RegistrationComplete")
	attCtx := ueCtx.attCtx
	attCtx.t3550.Stop()

	flags := &ConfigUpdateFlags{
		isGpp: isGpp,
		nitz:  true,
		//TODO: depending on the type of registration we may set different flag
		//values
	}
	ueCtx.state.SetNextEvent(fsm.NewEventData(RegisterDoneEvent, flags))
}

// Get AMData (if needed) then determine allowed slices for UE
func (ueCtx *UeContext) determineAllowedSlices(nasSlices *nas.Nssai, isGpp bool) (err error) {
	isRoaming := !common.IsPlmnIdEqual(amfctx.PlmnId(), ueCtx.plmnId)
	sliceMapping := amfctx.GetSliceMapping(*ueCtx.plmnId)
	var candidates, allowedSlices []models.AllowedSnssai
	if candidates, err = common.GetAllowedSlices(isRoaming, nasSlices, ueCtx.amData.Nssai, sliceMapping); err != nil {
		return utils.WrapError("Get allowed Snssai", err)
	}
	for _, s := range candidates {
		if amfctx.IsServingSlice(&s.AllowedSnssai) {
			allowedSlices = append(allowedSlices, s)
		}
	}

	if len(allowedSlices) == 0 {
		return fmt.Errorf("No allowed slice found")
	}

	if isGpp {
		ueCtx.gpp.allowedSlices = allowedSlices
	} else {
		ueCtx.nonGpp.allowedSlices = allowedSlices
	}
	return nil
}

// allocate registration area for a RanState
func (ueCtx *UeContext) allocateRegistrationArea() {
	//TODO: check if current Ue's TAI is in its allowed areas
	if ueCtx.lastTai != nil {
		ueCtx.Infof("Set last TAI as registration area")
		ueCtx.taList = []models.Tai{
			models.TaiFromNasTai(ueCtx.lastTai),
		}
		return
	}

	//ueCtx.Warnf("Allocate a dummy registration area: %s", ueCtx.plmnId.String())
	ueCtx.taList = []models.Tai{
		models.Tai{
			PlmnId: *amfctx.PlmnId(),
			Tac:    "000001",
		},
	}
}

// asiggn LADN information for RanState
func (ueCtx *UeContext) assignLadnInfo() {
	//ueCtx.Warn("Assign ladn information not implemented")
	//TODO:
	return
}

func (ueCtx *UeContext) setDrx(drx uint8) {
	switch drx {
	case nas.DRXcycleParameterT32:
		ueCtx.Tracef("Requested DRX: T = 32")
		ueCtx.drx = nas.DRXcycleParameterT32
	case nas.DRXcycleParameterT64:
		ueCtx.Tracef("Requested DRX: T = 64")
		ueCtx.drx = nas.DRXcycleParameterT64
	case nas.DRXcycleParameterT128:
		ueCtx.Tracef("Requested DRX: T = 128")
		ueCtx.drx = nas.DRXcycleParameterT128
	case nas.DRXcycleParameterT256:
		ueCtx.Tracef("Requested DRX: T = 256")
		ueCtx.drx = nas.DRXcycleParameterT256
	case nas.DRXValueNotSpecified:
		fallthrough
	default:
		ueCtx.drx = nas.DRXValueNotSpecified
		ueCtx.Tracef("Requested DRX: Value not specified")
	}
}

// Send an ContextSetupRequest to PRAN and handle its response
func (ueCtx *UeContext) sendInitialContextSetupRequest(isGpp bool, pduList []models.N2SmInfoDownlinkContent, nasPdu []byte) error {
	ueCtx.Debugf("Send an InitialContextSetupRequest")
	ranUe := ueCtx.getRanUe(isGpp)
	msg := &models.UeContextSetupRequest{
		N2SmInfoDownlinks: pduList,
		NasPdu:            nasPdu,
		SecKey:            ueCtx.getRanKey(isGpp),
	}

	msg.UeRadCap = "" //UE radio capability
	msg.Guami = amfctx.GetGuami()
	msg.UeAmbr = ueCtx.getAmbr()
	if isGpp {
		msg.AllowedNssai = ueCtx.gpp.allowedSlices
	} else {
		msg.AllowedNssai = ueCtx.nonGpp.allowedSlices
	}
	msg.UeSecCap = ueCtx.getUeSecurityCapability()
	if rsp, errRsp, err := ranUe.SendInitialContextSetupRequest(msg); err != nil {
		return utils.WrapError("Send InitialContextSetupRequest", err)
	} else if rsp != nil {
		//forward N2SmInfo uplink to SMF
		ueCtx.ForwardN2SmInfoUplink(&models.N2SmInfoUplinkTransport{
			Transfers: rsp.Transfers,
		})
	} else if errRsp != nil {
		ueCtx.Error("GnB return an error for the InitialContextSetupRequest")
		// deactivate sessions
		ueCtx.deactivatePduSessions(errRsp.FailedList)
	}
	return nil
}
