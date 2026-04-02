package uecontext

import (
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	amfctx "etrib5gc/nfs/amf/context"
	"etrib5gc/nfs/amf/procs/secmode"
	"etrib5gc/nfs/amf/secctx"
	"etrib5gc/nfs/amf/sm"
	"etrib5gc/nfs/amf/types"
	"fmt"

	"sync"

	"github.com/reogac/nas"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

const (
	DEFAULT_UE_UPLINK_AMBR   int64 = 1000000
	DEFAULT_UE_DOWNLINK_AMBR int64 = 1000000
)

type Ladn struct {
	Dnn     string
	TaiList []models.Tai
}

type RanUe interface {
	AddToPool()
	IsGpp() bool
	IsReleased() bool
	SendNas([]byte) error
	ReleaseContext(models.N2Cause) (*models.UeContextReleaseComplete, error)
	SendInitialContextSetupRequest(*models.UeContextSetupRequest) (*models.UeContextSetupResponse, *models.UeContextSetupFailure, error)
	SendN2SmInfoDownlink([]models.N2SmInfoDownlinkContent, []byte) error
	SendHandoverRequest(*models.HandoverRequest) (*models.HandoverRequestAcknowledge, *models.HandoverRequestFailure, error)
	RanInfo() (models.AccessType, []string, *models.RanUeInfo)
	WriteInfo() *types.RanUeInfo
}

type RanFace struct {
	ranUe         RanUe
	registered    bool
	allowedSlices []models.AllowedSnssai
}

type ConfigUpdateFlags struct {
	isGpp bool
	nitz  bool
	//add other flags
}

type ConfigUpdateContext struct {
	flags    *ConfigUpdateFlags
	t3555    common.Timer //Wait for NAS ConfigurationUpdateComplete
	t3555Cnt int
}

type UeContext struct {
	*logrus.Entry

	state *fsm.State

	supi   string
	suci   string
	pei    string
	tmsi   uint32 //identity at AMF
	gpsi   string
	plmnId *models.PlmnId //home plmn id

	currentSecCtx, nonCurrentSecCtx *secctx.SecurityContext //associated with primaty authentication

	drx uint8

	abba []byte

	metadata map[string]string
	ladnInfo []Ladn

	lastTai *nas.TrackingAreaIdentity //last visited TAI for 3GPP access
	taList  []models.Tai              //registration are for 3GPP access

	amPolicy   *models.PolicyAssociation
	amPolicyId string

	smfSel  *models.SmfSelectionSubscriptionData      //smf selection information from UDM
	smData  *models.SmSubsData                        //session management subscription data from UDM
	amData  *models.AccessAndMobilitySubscriptionData //Access and Mobility subscription data from UDM
	ueInSmf *models.UeContextInSmfData                //sessions information from UDM

	rat models.RatType
	loc models.UserLocation

	gmmCap *nas.GmmCapability
	secCap *nas.UeSecurityCapability

	gpp, nonGpp RanFace

	sessions *PduSessions

	n1n2            *PendingN1N2
	n1n2Mu, ranUeMu sync.Mutex

	udmCli sbi.ConsumerClient
	pcfCli sbi.ConsumerClient

	attCtx   *AttachContext         //on-going registration context
	deregCtx *DeregistrationContext //on-going deregistration context

	hoCtx *HandoverContext //on-going handover context
	hoMu  sync.Mutex

	configCtx *ConfigUpdateContext
}

// find existing UEContext from received MobileIdentity or create a new
// UeContext
func GetUeContext(gmm *nas.DecodedGmmMessage) (ueCtx *UeContext, err error) {
	createUe := false
	switch gmm.MsgType {
	case nas.RegistrationRequestMsgType:
		mobileId := &gmm.RegistrationRequest.MobileIdentity
		if ueCtx = findUeByMobileId(mobileId); ueCtx == nil {
			createUe = true
			//For registration request, we will create a new UeContext
			if ueCtx, err = newUeContext(mobileId); err != nil {
				err = utils.WrapError("Create new UeContext", err)
			}
		}

	case nas.ServiceRequestMsgType:
		mobileId := &gmm.ServiceRequest.STmsi
		if idType := mobileId.GetType(); idType != nas.MobileIdentity5GSType5gSTmsi {
			err = fmt.Errorf("No Tmsi5Gs in service request")
			return
		}
		if ueCtx = findUeByMobileId(mobileId); ueCtx == nil {
			err = fmt.Errorf("UeContext not found with id=%s in ServiceRequest", mobileId.String())
		}

	case nas.DeregistrationRequestFromUeMsgType:
		mobileId := &gmm.DeregistrationRequestFromUe.MobileIdentity
		if ueCtx = findUeByMobileId(mobileId); ueCtx == nil {
			err = fmt.Errorf("UeContext not found with id=%s in DeregistrationRequestFromUe", mobileId.String())
		}

	default:
		err = fmt.Errorf("Unexpected Nas message in InitialUeMessage")
	}
	if ueCtx != nil && !createUe {
		ueCtx.Infof("UeContext is found")
	}
	return
}

func newUeContext(mobileId *nas.MobileIdentity) (ueCtx *UeContext, err error) {
	ueCtx = &UeContext{
		Entry:    log,
		abba:     []uint8{0x00, 0x00},
		metadata: make(map[string]string),
	}
	ueCtx.sessions = newPduSessions(ueCtx)

	ueCtx.state = fsm.CreateState(_sm, MM_IDLE, ueCtx)
	//set mobile Id
	idType := mobileId.GetType()
	switch idType {
	case nas.MobileIdentity5GSTypeSuci:
		suciId := mobileId.Id.(*nas.Suci)
		if suciId.GetSupiFormat() == nas.SupiFormatImsi { //only accept this kind of SUCI
			concealedSupi := suciId.Content.(*nas.SupiImsi)
			//set SUCI
			ueCtx.suci = concealedSupi.String()
			//set home PlmnId
			var plmnId models.PlmnId
			plmnId.Mcc, plmnId.Mnc = concealedSupi.PlmnId.Get()
			if err = ueCtx.setHomeNetwork(&plmnId); err != nil {
				return nil, utils.WrapError("Set Home Network for UeContext: %+v", err)
			}
		}

	case nas.MobileIdentity5GSTypeImei, nas.MobileIdentity5GSTypeImeisv:
		//Set PEI
		ueCtx.pei = mobileId.String()

	default: //ignore Guti/Tmsi5Gs
		//do nothing
	}
	_uePool.add(ueCtx)
	return
}

func (ueCtx *UeContext) attachRanUe(ranUe RanUe) {
	ueCtx.ranUeMu.Lock()
	var old RanUe
	if ranUe.IsGpp() {
		if old = ueCtx.gpp.ranUe; old != nil && old != ranUe {
			ueCtx.Warnf("There an old RanUe that need to detach")
		}
		ueCtx.gpp.ranUe = ranUe
	} else {
		if old = ueCtx.nonGpp.ranUe; old != nil && old != ranUe {
			ueCtx.Warnf("There an old RanUe that need to detach")
		}
		ueCtx.nonGpp.ranUe = ranUe
	}
	defer ueCtx.ranUeMu.Unlock()
	if old != nil {
		old.ReleaseContext(models.N2Cause{})
	}
	return
}

func (ueCtx *UeContext) getRanUe(isGpp bool) RanUe {
	ueCtx.ranUeMu.Lock()
	defer ueCtx.ranUeMu.Unlock()
	if isGpp {
		return ueCtx.gpp.ranUe
	}
	return ueCtx.nonGpp.ranUe
}

func (ueCtx *UeContext) setRegistrationStatus(isGpp bool, status bool) {
	if isGpp {
		ueCtx.gpp.registered = status
	}
	ueCtx.nonGpp.registered = status
}

func (ueCtx *UeContext) updateMetadata(options map[string]string) {
	for k, v := range options {
		ueCtx.metadata[k] = v
	}
}

// update id for the UE then ask AmfContext to update UePool
func (ueCtx *UeContext) updateId(id *nas.MobileIdentity) error {
	idType := id.GetType()
	switch idType {
	case nas.MobileIdentity5GSTypeSuci:
		suci := id.Id.(*nas.Suci)
		ueCtx.Tracef("Receive a SUCI: %s", suci.String())

		if suci.GetSupiFormat() != nas.SupiFormatImsi {
			return fmt.Errorf("Suci format %d not supported", suci.GetSupiFormat())
		}
		concealedSupi := suci.Content.(*nas.SupiImsi)
		var plmnId models.PlmnId
		plmnId.Mcc, plmnId.Mnc = concealedSupi.PlmnId.Get()
		if err := ueCtx.setHomeNetwork(&plmnId); err != nil {
			return utils.WrapError("Connect home network", err)
		}
		ueCtx.suci = concealedSupi.String()
		ueCtx.Infof("Set SUCI for Ue")

	case nas.MobileIdentity5GSTypeImei, nas.MobileIdentity5GSTypeImeisv:
		ueCtx.Tracef("IMEI: %d", idType)
		ueCtx.setPei(id.Id.String())
		ueCtx.Infof("Set PEI for Ue")

	default:
		return fmt.Errorf("Mobile identity type %d is not handled", idType)
	}
	return nil
}

func (ueCtx *UeContext) setHomeNetwork(plmnId *models.PlmnId) error {
	ueCtx.plmnId = plmnId
	if err := ueCtx.connectUdm(); err != nil {
		return err
	}

	if err := ueCtx.connectPcf(); err != nil {
		return err
	}
	return nil
}

func (ueCtx *UeContext) setSupi(supi string) {
	ueCtx.supi = supi
	ueCtx.Entry = ueCtx.WithFields(logrus.Fields{
		"supi": supi,
	})
	_uePool.updateUeSupi(supi, ueCtx)
}

func (ueCtx *UeContext) setPei(pei string) {
	ueCtx.pei = pei
	_uePool.updateUePei(pei, ueCtx)
}

// check if ueCtxis in allowed service area for pdu session establishment
func (ueCtx *UeContext) isReEstablishPduSessionAllowed() bool {
	//TODO: check if pdu session re-estabishment is allowed
	/*
		if ueCtx.AmPolicyAssociation != nil && ueCtx.AmPolicyAssociation.ServAreaRes != nil {
			switch ueCtx.AmPolicyAssociation.ServAreaRes.RestrictionType {
			case models.RestrictionType_ALLOWED_AREAS:
				allowReEstablishPduSession = context.TacInAreas(ueCtx.Tai.Tac, ueCtx.AmPolicyAssociation.ServAreaRes.Areas)
			case models.RestrictionType_NOT_ALLOWED_AREAS:
				allowReEstablishPduSession = !context.TacInAreas(ueCtx.Tai.Tac, ueCtx.AmPolicyAssociation.ServAreaRes.Areas)
			}
		}
	*/

	return true
}

func (ueCtx *UeContext) cmConnected(isGpp bool) bool {
	ueCtx.ranUeMu.Lock()
	defer ueCtx.ranUeMu.Unlock()
	if isGpp {
		return ueCtx.gpp.ranUe != nil
	} else {
		return ueCtx.nonGpp.ranUe != nil
	}
}

// get allowes Snssai that matches to a requested slice (in serving network)
func (ueCtx *UeContext) getAllowedSnssai(snssai *models.Snssai, isGpp bool) *models.AllowedSnssai {
	if slices := ueCtx.getAllowedSlices(isGpp); len(slices) > 0 {
		for _, s := range slices {
			if common.IsSliceEqual(&s.AllowedSnssai, snssai) {
				return &s
			}
		}
	}
	return nil
}

// check if an Snssai is in the allowed Snssai list for a given access
func (ueCtx *UeContext) isSnssaiAllowed(snssai *models.Snssai, isGpp bool) bool {
	return ueCtx.getAllowedSnssai(snssai, isGpp) != nil
}

func (ueCtx *UeContext) isCurrentSecurityContext(ngKsi *nas.KeySetIdentifier) bool {
	if secCtx := ueCtx.currentSecCtx; secCtx != nil {
		cNgKsi := secCtx.NgKsi()
		return cNgKsi.Id == ngKsi.Id && cNgKsi.Tsc == ngKsi.Tsc
	}
	return false
}

func (ueCtx *UeContext) createNonCurrentSecCtx(kamf []byte, ngKsi *models.NgKsi, isGpp bool) {
	ueCtx.nonCurrentSecCtx = secctx.NewSecurityContext(ngKsi.NasType(), kamf, isGpp)
}

func (ueCtx *UeContext) updateSecmode(secCtx *secctx.SecurityContext) {
	ueCtx.Infof("Add security context: %s", secCtx.NgKsi().String())
	ueCtx.currentSecCtx = secCtx
	ueCtx.nonCurrentSecCtx = nil
}

func (ueCtx *UeContext) deleteSecurityContext() {
	if ueCtx.currentSecCtx != nil {
		ueCtx.Infof("Delete security context: %s", ueCtx.currentSecCtx.NgKsi().String())
		ueCtx.currentSecCtx = nil
	}
	ueCtx.nonCurrentSecCtx = nil
}

// create a native security context without KSI confliction
func (ueCtx *UeContext) selectNewNgKsi(old *models.NgKsi) (ngKsi models.NgKsi) {
	if old != nil {
		ueCtx.Infof("Old NgKsi: %s", old.String())
	}
	ngKsi.Tsc = models.SCTYPE_NATIVE

	var i int
	for i = 0; i < 7; i++ {
		//check confliction with old one
		if old != nil && i == old.Ksi {
			continue
		}
		if current := ueCtx.currentSecCtx; current != nil {
			if current.NgKsi().Id == uint8(i) {
				continue
			}
		}
		ngKsi.Ksi = i
		break
	}
	return
}

func (ueCtx *UeContext) servingNetwork() string {
	return common.ServingNetworkName(amfctx.PlmnId())
}

// get registration status (for composing RegistrationAccept)
func (ueCtx *UeContext) registrationStatus(isGpp bool) (status uint8) {
	status = 0
	if isGpp {
		status |= nas.AccessType3GPP
		if ueCtx.nonGpp.registered {
			status |= nas.AccessTypeNon3GPP
		}
	} else {
		status |= nas.AccessTypeNon3GPP
		if ueCtx.gpp.registered {
			status |= nas.AccessType3GPP
		}
	}
	return
}

// buid a Nas Guti from Tmsi and local context information
func (ueCtx *UeContext) getNasGuti() *nas.Guti {
	return &nas.Guti{
		PlmnId: amfctx.NasPlmnId(),
		AmfId:  amfctx.NasAmfId(),
		Tmsi:   ueCtx.tmsi,
	}
}
func (ueCtx *UeContext) resetUeRadioCap() {
	//TODO: needs more investigation
	//ueCtx.UeRadioCapability = ""
	//ueCtx.UeRadioCapabilityForPaging = nil
}

func (ueCtx *UeContext) findSmContext(id uint8) (smCtx *sm.SmContext) {
	return ueCtx.sessions.find(id)
}

func (ueCtx *UeContext) addSmContext(smCtx *sm.SmContext) {
	ueCtx.sessions.add(smCtx)
}

func (ueCtx *UeContext) deleteSmContext(id uint8) {
	if ueCtx.sessions.remove(id) {
		ueCtx.Infof("SmContext [sid=%d] is removed", id)
	}
}

// find Dnn for a Session from subscription data of the Ue
func (ueCtx *UeContext) findDnn(snssai *models.Snssai, isGpp bool) string {
	if snssai != nil {
		//convert ssnssai to string
		id := fmt.Sprintf("%02x%s", snssai.Sst, snssai.Sd)
		if ueCtx.smfSel != nil {
			if info, ok := ueCtx.smfSel.SubscribedSnssaiInfos[id]; ok {
				for _, dnninfo := range info.DnnInfos {
					if dnninfo.DefaultDnnIndicator != nil && *dnninfo.DefaultDnnIndicator {
						return dnninfo.Dnn
					}
				}
			}
		}
	}
	return amfctx.DefaultDnn(isGpp) //NOTE: AMF should always has a default DNN
}

func (ueCtx *UeContext) getAmbr() models.UeAmbr {
	if ueCtx.amData.SubscribedUeAmbr != nil {
		//TODO: convert from string to int64
		return models.UeAmbr{
			Ul: DEFAULT_UE_UPLINK_AMBR,
			Dl: DEFAULT_UE_DOWNLINK_AMBR,
		}

	} else {
		return models.UeAmbr{
			Ul: DEFAULT_UE_UPLINK_AMBR,
			Dl: DEFAULT_UE_DOWNLINK_AMBR,
		}
	}
}

func (ueCtx *UeContext) updateCapabilities(msg *nas.RegistrationRequest) {
	if msg.UeSecurityCapability != nil {
		ueCtx.secCap = msg.UeSecurityCapability
	}

	if msg.GmmCapability != nil {
		ueCtx.gmmCap = msg.GmmCapability
	}
}

func (ueCtx *UeContext) updateSecurityCapability(secCap *models.UeSecurityCapability) {
	if secCap == nil || secCap.Nr == nil {
		return
	}
	b := (secCap.Nr.Enc[0] & 0x80 >> 7) == 1
	ueCtx.secCap.SetEA(1, b)

	b = (secCap.Nr.Enc[0] & 0x40 >> 6) == 1
	ueCtx.secCap.SetEA(2, b)

	b = (secCap.Nr.Enc[0] & 0x20 >> 5) == 1
	ueCtx.secCap.SetEA(3, b)

	b = (secCap.Nr.Int[0] & 0x80 >> 7) == 1
	ueCtx.secCap.SetIA(1, b)

	b = (secCap.Nr.Int[0] & 0x40 >> 6) == 1
	ueCtx.secCap.SetIA(2, b)

	b = (secCap.Nr.Int[0] & 0x20 >> 5) == 1
	ueCtx.secCap.SetIA(3, b)

}

func (ueCtx *UeContext) getUeSecurityCapability() (ueSecCap models.UeSecurityCapability) {
	ueSecCap = models.UeSecurityCapability{
		Nr: &models.SecurityCapability{
			Enc: []byte{0, 0},
			Int: []byte{0, 0},
		},
	}

	ueSecCap.Nr.Enc[0] |= b2uint8(ueCtx.secCap.GetEA(1)) << 7
	ueSecCap.Nr.Enc[0] |= b2uint8(ueCtx.secCap.GetEA(2)) << 6
	ueSecCap.Nr.Enc[0] |= b2uint8(ueCtx.secCap.GetEA(3)) << 5

	ueSecCap.Nr.Int[0] |= b2uint8(ueCtx.secCap.GetIA(1)) << 7
	ueSecCap.Nr.Int[0] |= b2uint8(ueCtx.secCap.GetIA(2)) << 6
	ueSecCap.Nr.Int[0] |= b2uint8(ueCtx.secCap.GetIA(3)) << 5
	ueSecCap.Eutra = &models.SecurityCapability{
		Enc: []byte{0, 0},
		Int: []byte{0, 0},
	}
	return
}

// delete authenticated information
func (ueCtx *UeContext) resetAuthenticationContext() {
	ueCtx.Warnf("Reset authentication context")
	ueCtx.supi = ""
	ueCtx.nonCurrentSecCtx = nil
	ueCtx.currentSecCtx = nil
}

func (ueCtx *UeContext) HasValidSecmode() bool {
	return ueCtx.currentSecCtx != nil
}

// check if we are in a secmode establishment procedure
func (ueCtx *UeContext) isSecmodeOnGoing() bool {
	if ueCtx.attCtx != nil && ueCtx.attCtx.proc != nil {
		_, ok := ueCtx.attCtx.proc.(*secmode.SecmodeProcedure)
		return ok
	}
	return false
}
func (ueCtx *UeContext) getNasContext() *nas.NasContext {
	if ueCtx.isSecmodeOnGoing() {
		return ueCtx.nonCurrentSecCtx.NasContext()
	} else {
		if ueCtx.currentSecCtx != nil {
			return ueCtx.currentSecCtx.NasContext()
		}
	}
	return nil
}

func (ueCtx *UeContext) getAllowedSlices(isGpp bool) []models.AllowedSnssai {
	if isGpp {
		return ueCtx.gpp.allowedSlices
	}
	return ueCtx.nonGpp.allowedSlices
}

func (ueCtx *UeContext) smfSelectionOptions() map[string]string {
	return ueCtx.metadata
}

func (ueCtx *UeContext) isRegistered(isGpp bool) bool {
	if isGpp {
		return ueCtx.gpp.registered
	}
	return ueCtx.nonGpp.registered
}

func (ueCtx *UeContext) getRanKey(isGpp bool) []byte {
	if isGpp {
		return ueCtx.currentSecCtx.Kgnb()
	}
	return ueCtx.currentSecCtx.Kn3iwf()
}

func (ueCtx *UeContext) getRanUeInfo(isGpp bool) (access models.AccessType, ranNets []string, ranInfo *models.RanUeInfo) {
	ueCtx.ranUeMu.Lock()
	defer ueCtx.ranUeMu.Unlock()
	if isGpp {
		return ueCtx.gpp.ranUe.RanInfo()
	} else {
		return ueCtx.nonGpp.ranUe.RanInfo()
	}
}
