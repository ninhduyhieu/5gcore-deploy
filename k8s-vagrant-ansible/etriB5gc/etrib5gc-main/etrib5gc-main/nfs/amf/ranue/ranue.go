package ranue

import (
	amfctx "etrib5gc/nfs/amf/context"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
)

type UeContext interface {
	HandleHandoverRequired(bool, interface{}, *models.HandoverRequired) (*models.HandoverCommand, *models.HandoverPreparationFailure, error)
	HandleHandoverNotify(bool, *models.HandoverNotify) error
	HandleHandoverCancel(bool, *models.HandoverCancel) (*models.HandoverCancelAcknowledge, error)
	DoPathSwitch(bool, *models.PathSwitchRequest) (*models.PathSwitchAcknowledge, *models.PathSwitchFailure, error)
	HandleNasUplink(bool, []byte) error
	ReceiveUeContextReleaseRequest(bool, *models.UeContextReleaseRequest)
	ForwardN2SmInfoUplink(*models.N2SmInfoUplinkTransport)
}

type RanUe struct {
	*logrus.Entry
	ue         UeContext //point to the shared UE context (supi, suci, pei etc.)
	isReleased bool      //to marked if PRAN/gnB or Core have triggered RanUe context release

	//RAN information
	ranNets  []string //associate user plane RAN networks (N3)
	rrcCause string
	ranInfo  *models.EndpointInfo
	nasSplit bool
	ranUeId  models.RanUeId     //identity at PRAN
	ranCli   sbi.ConsumerClient //consumer to PRAN

	isGpp bool //3GPP access or Non-3GPP access

	localId int64 //local identity

	metrics *RanUeMetrics

	isUeIdSent bool //has AmfUeId sent to PRAN?
}

// fill in information for RanUe from InitialUeMessage
func newRanUe(ueCtx UeContext, cli sbi.ConsumerClient, msg *models.InitialUeMessage, callback *models.EndpointInfo) *RanUe {
	ranUe := &RanUe{
		Entry: log.WithFields(logrus.Fields{
			"ranUeId": msg.RanUeId.String(),
		}),
		ue:       ueCtx,
		ranCli:   cli,
		ranNets:  msg.RanNets,
		ranUeId:  msg.RanUeId,
		ranInfo:  callback,
		nasSplit: msg.NasSplit,
		isGpp:    isGpp(msg.Access),
		metrics:  createRanUeMetrics(),
	}
	return ranUe
}

// create a target RanUe from source RanUe and HandoverRequire message
func CreateHandoverRanUe(ueCtx UeContext, source *RanUe, msg *models.HandoverRequired) (ranUe *RanUe, err error) {
	var gnb *amfctx.Ran
	if gnb = amfctx.FindRan(&msg.TargetId); gnb == nil {
		err = fmt.Errorf("Fail to find target Ran")
		return
	}

	ranUe = &RanUe{
		ue:      ueCtx,
		isGpp:   source.isGpp,
		ranCli:  gnb.Client(),
		ranInfo: gnb.RanInfo(),
		metrics: copyRanUeMetrics(source.metrics),
	}

	_ranUePool.add(ranUe)
	ranUe.Entry = log.WithFields(logrus.Fields{
		"amfUeId": ranUe.localId,
		"ranUeId": ranUe.ranUeId.String(),
	})
	return
}

func (ranUe *RanUe) accessType() models.AccessType {
	return getAccessType(ranUe.isGpp)
}

func (ranUe *RanUe) RanInfo() (access models.AccessType, ranNets []string, ranInfo *models.RanUeInfo) {
	access = ranUe.accessType()
	ranNets = ranUe.ranNets
	if ranUe.nasSplit {
		ranInfo = &models.RanUeInfo{
			RanInfo: *ranUe.ranInfo,
			RanUeId: ranUe.ranUeId,
		}
	}
	return
}

// build AmfUeInfo to notify PRAN of UeContext at AMF
func (ranUe *RanUe) buildAmfUeInfo() *models.AmfUeContextInfo {
	if !ranUe.isUeIdSent {
		ranUe.isUeIdSent = true
		return &models.AmfUeContextInfo{
			PlmnId:     *amfctx.PlmnId(),
			AmfUeId:    ranUe.localId,
			AmfSet:     amfctx.AmfSet(),
			AmfPointer: int16(amfctx.AmfPointer()),
		}
	}
	return nil
}

func (ranUe *RanUe) AmfUeId() int64 {
	return ranUe.localId
}

func (ranUe *RanUe) IsGpp() bool {
	return ranUe.isGpp
}

func (ranUe *RanUe) IsReleased() bool {
	return ranUe.isReleased
}
