package ranue

import (
	"context"
	"github.com/reogac/sbi/apis/pran/n1n2dl"
	"github.com/reogac/sbi/apis/pran/nasdl"
	"github.com/reogac/sbi/models"
)

func (ranUe *RanUe) ReceiveNasErr(ctx context.Context, msg *models.UplinkNasError) {
	//TODO
	ranUe.Warnf("Uplink NasErr not handled")
}

func (ranUe *RanUe) ReceiveNasUplink(ctx context.Context, pdu []byte) error {
	ranUe.Debugf("Receive a NasUplink")
	return ranUe.ue.HandleNasUplink(ranUe.isGpp, pdu)
}

func (ranUe *RanUe) ReceiveN2SmInfoUplink(msg *models.N2SmInfoUplinkTransport) error {
	ranUe.Debugf("Receive a N2SmInfoUplink")
	ranUe.ue.ForwardN2SmInfoUplink(msg)
	return nil
}

// send N1MM to PRAN/gnB
func (ranUe *RanUe) SendNas(pdu []byte) (err error) {
	msg := &models.NasDownlinkTransport{
		NasPdu: pdu,
	}
	msg.AmfUeInfo = ranUe.buildAmfUeInfo()
	return nasdl.NasDl(ranUe.ranCli, ranUe.ranUeId.Id, msg)
}

// send N2SmInfo downlink, a nas message can be piggy-back
func (ranUe *RanUe) SendN2SmInfoDownlink(transfers []models.N2SmInfoDownlinkContent, nasPdu []byte) error {
	msg := &models.N2SmInfoDownlinkTransport{
		Transfers: transfers,
		NasPdu:    nasPdu,
	}
	return n1n2dl.N2SmInfoDownlink(ranUe.ranCli, ranUe.ranUeId.Id, msg)
}
