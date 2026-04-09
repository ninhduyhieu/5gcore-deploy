package ue

import (
	"context"
	"etrib5gc/common"
	"etrib5gc/internal/fsm"
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/udm/sdm"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/reogac/nas"
)

type NasProc interface {
	ReceiveN1Mm(context.Context, *nas.DecodedGmmMessage)
	Close()
}

type UeContext struct {
	*logrus.Entry
	state *fsm.State

	ranUeId   models.RanUeId //id at RAN
	amfUeId   int64          //id at AMF
	amData    *models.AccessAndMobilitySubscriptionData
	amfRegion uint8
	isGpp     bool

	proc NasProc //authentication procedure or identification procedure

	suci *nas.SupiImsi //from registration request

	supiId *nas.SupiImsi //from ausf/udm

	metadata map[string]string //accumulated NF selection options for UeContext

	requestedSlices *nas.Nssai //requested slices from RegistrationRequest

	gmm *nas.DecodedGmmMessage

	ranCli sbi.ConsumerClient
	amfCli sbi.ConsumerClient

	msg *models.InitialUeMessage //initial Ue message wrapper

	ranInfo models.EndpointInfo //pran information to be forwarded to AMF
}

func newUeContext(ranInfo models.EndpointInfo, msg *models.InitialUeMessage) (*UeContext, error) {
	ueCtx := &UeContext{
		isGpp:     msg.Access == models.ACCESSTYPE_3GPP_ACCESS,
		amfRegion: uint8(msg.AmfRegion),
		ranUeId:   msg.RanUeId,
		msg:       msg,
		ranInfo:   ranInfo,
		metadata:  make(map[string]string),
		Entry: log.WithFields(logrus.Fields{
			"ranUeId": msg.RanUeId.String(),
		}),
	}

	ueCtx.state = fsm.CreateState(_sm, UE_AUTHENTICATING, ueCtx)
	//copy NfSelection options
	for k, v := range msg.NfSelection {
		ueCtx.metadata[k] = v
	}

	var nasMsg nas.NasMessage
	var err error
	//Decode plain Nas Message
	if nasMsg, err = nas.Decode(nil, msg.NasPdu, ueCtx.isGpp); err != nil {
		return nil, utils.WrapError("Decode NAS", err)
	} else if nasMsg.Gmm == nil {
		return nil, fmt.Errorf("N1Mm message not presented")
	} else {
		ueCtx.gmm = nasMsg.Gmm
	}

	//create pran client
	if ueCtx.ranCli, err = mesh.ConsumerFromEndpoint(&ranInfo); err != nil {
		return nil, utils.WrapError("Create PRAN consumer", err)
	}

	return ueCtx, nil
}

func (ueCtx *UeContext) GetStateId() string {
	return fmt.Sprintf("%d", ueCtx.amfUeId)
}

func (ueCtx *UeContext) sendEvent(ctx context.Context, ev *fsm.EventData) chan error {
	return _sm.SendEvent(ctx, ueCtx.state, ev)
}

// call this externally, never do so in a callback handler.
func (ueCtx *UeContext) sendEventEx(ctx context.Context, ev *fsm.EventData) (prob *models.ProblemDetails) {
	if err := <-ueCtx.sendEvent(ctx, ev); err != nil {
		ueCtx.Errorf("Event not handled: %s", err.Error())
		prob = &models.ProblemDetails{
			Detail: "UeContext not in a valid state to handle the request",
			Status: http.StatusConflict,
		}
	}
	return
}
func (ueCtx *UeContext) plmnId() *models.PlmnId {
	mcc, mnc := ueCtx.suci.PlmnId.Get()
	return &models.PlmnId{
		Mcc: mcc,
		Mnc: mnc,
	}
}

func (ueCtx *UeContext) getAmData() error {
	plmnId := ueCtx.plmnId()
	//create udm client
	sid := common.UdmServiceName(plmnId)
	udmCli, err := mesh.Consumer(sid, ueCtx.metadata)
	if err != nil {
		return utils.WrapError("Create Udm consumer", err)
	}

	params := sdm.GetAmDataParams{
		PlmnId: &models.PlmnIdNid{
			Mcc: plmnId.Mcc,
			Mnc: plmnId.Mnc,
		},
		Supi: ueCtx.msg.AuthCtx.Supi,
	}
	if _, amData, err := sdm.GetAmData(udmCli, params); err != nil { //ignore header
		ueCtx.Errorf("Fail to get AmData from UDM: %+v", err)
		return err
	} else {
		ueCtx.Tracef("Received Accesss and Mobility data from UDM")
		ueCtx.amData = amData
	}
	return nil
}
