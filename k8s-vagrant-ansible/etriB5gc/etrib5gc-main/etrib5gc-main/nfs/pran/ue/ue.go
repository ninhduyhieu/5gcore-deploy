package ue

import (
	"etrib5gc/common"
	"etrib5gc/internal/eventmux"
	"etrib5gc/mesh"
	"etrib5gc/nfs/pran/context"
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/amf/n2nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"net"
	//	"sync"
	"time"
)

const (
	MAX_PDU_SESSIONS uint8 = 16
)

const (
	CREATE_TIMEOUT time.Duration = 60 * time.Second //time to wait for initial uecontext setup (milliseconds)
)

type Ran interface {
	Conn() net.Conn
	Access() models.AccessType
	RanNets() []string //transport networks connected to gnB
	Send([]byte) error //send encoded Ngap message to Ran
}

type UeContext struct {
	*logrus.Entry

	ran    Ran                //client toward gnB
	amfCli sbi.ConsumerClient //client toward AMF/DAMF

	execSlot *eventmux.Slot[UeContext] //for sending event to event muxer

	ranNgapId int64          //sent by gnB
	localId   models.RanUeId //generate by PRAN
	amfUeId   int64          //sent by AMF/DAMF

	ambr *models.UeAmbr

	releaseCtx *ReleaseContext

	sessions [MAX_PDU_SESSIONS]*SmContext

	modifyJob   *UeCtxModifyContext     //pending ue context modification request
	setupJob    *UeCtxSetupContext      //pending ue context setup request
	handoverJob *HandoverRequestContext //pending handover request context

	//lock sync.Mutex
}

//create UeContext when receiving InitialUeMessage
func CreateUeContext(gnb Ran, msg *ies.InitialUEMessage) (err error) {
	if isUePoolClosed() {
		//Context has been closed, don't create any new UeContext
		err = fmt.Errorf("Application terminating")
		return
	}
	var loc *models.UserLocation
	if loc, err = locConvert(&msg.UserLocationInformation); err != nil {
		err = utils.WrapError("Parse Ngap UserLocationInformation", err)
		return
	}

	//find the designate AMF; if not found locate a default one (DAMF)
	var amfCli sbi.ConsumerClient
	if amfCli, err = context.FindAmf(msg.FiveGSTMSI, &msg.UserLocationInformation); err != nil {
		return utils.WrapError("Create a AMF/DAMF client", err)
	} else {
		//now create a new UeContext
		//generate identity at CU for the UE
		localId := models.RanUeId{
			Ran: _pool.ranId,
			Id:  _pool.ngapIdGen.Allocate(),
		}

		log.Infof("Create a new Ue context for to handle InitialUeMessage [ueId = %d]", localId.Id)
		ueCtx := &UeContext{
			ran:       gnb,
			amfCli:    amfCli,
			ranNgapId: msg.RANUENGAPID,
			localId:   localId,
		}
		ueCtx.Entry = log.WithFields(logrus.Fields{
			"cuUeId":    ueCtx.localId.Id,
			"ranNgapId": ueCtx.ranNgapId,
		})
		ueCtx.execSlot = _exec.MakeSlot(ueCtx)
		//add UeContext to pool
		_pool.add(ueCtx)

		//up to now, the UeContext shoudl be able to locate an AMF (either a
		//designated one or a default one)
		sbiMsg := &models.InitialUeMessage{
			RanUeId:     ueCtx.localId,
			NasPdu:      msg.NASPDU,
			Loc:         loc,
			Access:      ueCtx.ran.Access(),
			RanNets:     ueCtx.ran.RanNets(),
			NasSplit:    context.NasSplit(),
			NfSelection: context.GetNfSelection(),
			AmfRegion:   int16(context.AmfRegion()),
		}

		sbiMsg.RrcCause = int16(msg.RRCEstablishmentCause.Value)

		if msg.UEContextRequest != nil {
			sbiMsg.ContextRequest = msg.UEContextRequest.Value == ies.UEContextRequestRequested
		}

		var rsp *models.InitialUeMessageResponse
		callback := mesh.EndpointInfo()
		if rsp, err = n2nas.InitialUeMessage(ueCtx.amfCli, callback, sbiMsg); err != nil {
			err = utils.WrapError("Send InitialUeMessage to CORE", err)
			_pool.remove(ueCtx)
			return
		}
		ueCtx.Infof("InitialUeMessage forwarded to core, received AmfUeId=%d", rsp.AmfUeId)
		ueCtx.amfUeId = rsp.AmfUeId
	}

	return
}

//designate AMF send UeContext information for updating
func (ueCtx *UeContext) updateAmfClient(amfSet string, amfPointer uint8, plmnid models.PlmnId, amfUeId int64) (err error) {
	amf := common.AmfServiceName(&plmnid, amfSet)
	insId := common.AmfPointerString(amfPointer)
	if ueCtx.amfCli, err = mesh.ConsumerWithInstanceId(amf, insId); err != nil {
		return utils.WrapError("Create Amf consumer", err)
	}

	ueCtx.amfUeId = amfUeId
	return
}

func (ueCtx *UeContext) cuNgapId() int64 {
	return int64(ueCtx.localId.Id)
}

func (ueCtx *UeContext) UpdateRanInfo(gnb Ran, ngapId int64) {
	ueCtx.ran = gnb
	ueCtx.ranNgapId = ngapId
}
