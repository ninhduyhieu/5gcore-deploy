package sm

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"etrib5gc/logctx"
	"etrib5gc/mesh"
	smfctx "etrib5gc/nfs/smf/context"
	"etrib5gc/nfs/smf/tunnel"
	"fmt"
	"net"
	"time"

	"github.com/reogac/nas"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

const (
	//SM_CONTEXT_ACTIVATING_TIMEOUT   time.Duration =  30*time.Second
	//SM_CONTEXT_DEACTIVATING_TIMEOUT time.Duration = 5 *time.Second
	//	T3591                           time.Duration = 5 *time.Second
	T3592 time.Duration = 5 * time.Second
	//	T3593    time.Duration = 5 *time.Second
	HO_TIMER time.Duration = 5 * time.Second
)

const (
	MAX_T3591_CNT int = 2 //modification
	MAX_T3592_CNT int = 2 //release
)
const (
	SSC_MODE_1 uint8 = iota + 0x01 //modification
	SSC_MODE_2
	SSC_MODE_3
)

type SmContext struct {
	state *fsm.State
	*logrus.Entry

	supi string
	pei  string

	pduSessionId     uint32 //pdu session id assigned by UE
	id               uint32 //identity assigned by SMF
	sessionType      uint8
	dnn              string
	ranTransportNets []string //transport networks of the UE's attached gnB
	snssai           *models.Snssai
	access           models.AccessType
	sscMode          uint8

	homePlmnId *models.PlmnId

	rat    models.RatType
	pco    *nas.ExtendedProtocolConfigurationOptions //from UE
	smData *models.SessionManagementSubscriptionData //from UDM

	maxUplink   models.MaxIntegrityProtectedDataRate //max data rate per ue for Uplink UP integrity protection (string value)
	maxDownlink models.MaxIntegrityProtectedDataRate //max data rate per ue for downlink UP integrity protection (string value)

	smPolId string //SM policy identity
	smPol   *models.SmPolicyDecision

	amfCli sbi.ConsumerClient
	pcfCli sbi.ConsumerClient
	udmCli sbi.ConsumerClient

	tunnel *tunnel.Tunnel
	ueIp   *smfctx.LeasedIp

	relCtx *ReleaseContext      //on-going release context
	modCtx *ModificationContext //on-going modification context
	hoCtx  *HandoverContext     //on-going handover context

	n1n2 *N1N2Sender

	upCnxState uint8

	exp Experiment
}

func createSmContext(callback models.EndpointInfo, info *models.SmContextCreateData) (smCtx *SmContext, err error) {
	smCtx = &SmContext{
		Entry: logctx.Entry(logrus.Fields{
			"mod":       "sm",
			"supi":      info.Supi,
			"sessionId": fmt.Sprintf("%d", *info.PduSessionId),
		}),
		supi:         info.Supi,
		sscMode:      SSC_MODE_1,
		pduSessionId: uint32(*info.PduSessionId),
		snssai:       info.SNssai,
		//	plmnId:           models.PlmnIdFromPlmnIdNid(&info.ServingNetwork),
		dnn:              info.Dnn,
		ranTransportNets: info.RanTransportNets,
		access:           info.AnType,
		upCnxState:       UPCNXSTATE_DEACTIVATED,
	}
	if smCtx.homePlmnId, err = plmnIdFromSupi(smCtx.supi); err != nil {
		err = utils.WrapError("Get home PlmnId from SUPI", err)
		return
	}

	smCtx.state = fsm.CreateState(_sm, SM_INACTIVE, smCtx)
	smCtx.exp.createdTime = time.Now()

	pcfId := common.PcfServiceName(smCtx.homePlmnId)
	smCtx.Tracef("Create PCF consumer [%s]", pcfId)
	if smCtx.pcfCli, err = mesh.Consumer(pcfId, nil); err != nil {
		err = utils.WrapError("Create Pcf consumer", err)
		return
	}
	udmId := common.UdmServiceName(smCtx.homePlmnId)
	smCtx.Tracef("Create UDM consumer [%s]", udmId)
	if smCtx.udmCli, err = mesh.Consumer(udmId, nil); err != nil {
		err = utils.WrapError("Create Udm consumer", err)
		return
	}

	smCtx.Tracef("Create callback AMF consumer [%s]", models.EndpointInfoToString(callback))
	if smCtx.amfCli, err = mesh.ConsumerFromEndpoint(&callback); err != nil {
		err = utils.WrapError("Create Amf consumer", err)
		return
	}
	smCtx.n1n2 = newN1N2Sender(smCtx)
	if info.RanUeInfo != nil {
		smCtx.Tracef("Nas split enabled")
		err = smCtx.n1n2.setRanUeContext(info.RanUeInfo)
	}

	smCtx.Tracef("SmContext created")
	return
}

func (smCtx *SmContext) sendEvent(ctx context.Context, event *fsm.EventData) chan error {
	return _sm.SendEvent(ctx, smCtx.state, event)
}

func (smCtx *SmContext) getPduSessionType() models.PduSessionType {
	return Nas2SbiPduSessionType(smCtx.sessionType)
}

func (smCtx *SmContext) idString() string {
	return fmt.Sprintf("smcontext-%d", smCtx.id)
}

func (smCtx *SmContext) supiRef() string {
	return smContextRef(smCtx.supi, smCtx.pduSessionId)
}

func (smCtx *SmContext) getUpCnxState() models.UpCnxState {
	switch smCtx.upCnxState {
	case UPCNXSTATE_ACTIVATED:
		return models.UPCNXSTATE_ACTIVATED
	case UPCNXSTATE_DEACTIVATED:
		return models.UPCNXSTATE_DEACTIVATED
	case UPCNXSTATE_ACTIVATING:
		return models.UPCNXSTATE_ACTIVATING
	case UPCNXSTATE_SUSPENDED:
		return models.UPCNXSTATE_SUSPENDED
	}
	return ""
}

func (smCtx *SmContext) UeIp() net.IP {
	return smCtx.ueIp.IP()
}

func (smCtx *SmContext) Dnn() string {
	return smCtx.dnn
}
