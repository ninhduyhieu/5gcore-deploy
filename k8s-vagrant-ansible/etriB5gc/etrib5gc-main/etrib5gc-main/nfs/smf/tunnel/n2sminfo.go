package tunnel

import (
	"encoding/binary"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
)

func (t *Tunnel) FillPduSessionResourceSetupRequest(msg *ies.PDUSessionResourceSetupRequestTransfer) {
	//NOTE: here we should send multiple IP+TEID in cases where we have
	//different RAN facing UPFs (multi-paths)
	teid := make([]byte, 4)
	anNode := t.defaultPath.nodes[0]
	teidValue := anNode.uplink.GetTeid()
	t.Infof("Teid toward RAN = %d", teidValue)
	binary.BigEndian.PutUint32(teid, teidValue)
	n3ip := anNode.uplink.GetIp().To4()

	// UL NG-U UP TNL Information
	//NOTE: it seems that the address must be ipv4, otherwise the uransim gnb
	//will not decode it correctly

	//smCtx.Infof("Notify gnB of Uplink information: IP=%s, TEID = %d", n3ip.String(), smCtx.tunnel.Teid2Ran())

	msg.ULNGUUPTNLInformation.Choice = ies.UPTransportLayerInformationPresentGtptunnel
	msg.ULNGUUPTNLInformation.GTPTunnel = &ies.GTPTunnel{
		TransportLayerAddress: aper.BitString{
			Bytes:   n3ip,
			NumBits: 32,
		},
		GTPTEID: teid,
	}

	// PDU Session Aggregate Maximum Bit Rate
	// This IE is Conditional and shall be present when at least one NonGBR QoS flow is being setup.
	// TODO: should check if there is at least one NonGBR QoS flow

	//var srule *models.SessionRule

	var ulvalue, dlvalue int64
	/*
		if dlvalue, err = ngapconv.UEAmbrToInt64(srule.AuthSessAmbr.Downlink); err != nil {
			return
		}
		if ulvalue, err = ngapconv.UEAmbrToInt64(srule.AuthSessAmbr.Uplink); err != nil {
			return
		}
	*/
	dlvalue = 1000000
	ulvalue = 1000000
	msg.PDUSessionAggregateMaximumBitRate = &ies.PDUSessionAggregateMaximumBitRate{
		PDUSessionAggregateMaximumBitRateDL: dlvalue,
		PDUSessionAggregateMaximumBitRateUL: ulvalue,
	}

	// QoS Flow Setup Request List
	for _, p := range t.getAllPaths() {
		/*
			if !t.isDefaultPath(p) {
				continue
			}
		*/
		msg.QosFlowSetupRequestList = append(msg.QosFlowSetupRequestList, p.buildNgapQosFlowSetupRequestItem())
	}
}

// TS 38.413 9.3.4.9
func (t *Tunnel) FillPathSwitchRequestAcknowledge(msg *ies.PathSwitchRequestAcknowledgeTransfer) {
	teid := make([]byte, 4)
	anNode := t.defaultPath.nodes[0]
	binary.BigEndian.PutUint32(teid, anNode.uplink.GetTeid())
	n3ip := t.ranIp.To4() //TODO: Ipv4 or Ipv6 depending on the session type

	// UL NG-U UP TNL Information(optional) TS 38.413 9.3.2.2
	msg.ULNGUUPTNLInformation = &ies.UPTransportLayerInformation{
		Choice:    ies.UPTransportLayerInformationPresentGtptunnel,
		GTPTunnel: new(ies.GTPTunnel),
	}

	gtpTunnel := msg.ULNGUUPTNLInformation.GTPTunnel
	gtpTunnel.GTPTEID = teid
	gtpTunnel.TransportLayerAddress = aper.BitString{
		Bytes:   n3ip,
		NumBits: 32,
	}

}

// after receiving PduSessionResourceSetupResponse from gnB
func (t *Tunnel) UpdateQosFlowStatus(okQfiList, failedQfiList []uint8) {
	//TODO: update qos flow status
}
