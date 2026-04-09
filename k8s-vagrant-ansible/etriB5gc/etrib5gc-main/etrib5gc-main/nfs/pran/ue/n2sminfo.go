package ue

import (
	"etrib5gc/nfs/pran/context"
	"github.com/reogac/sbi/apis/amf/uectx"
	"github.com/reogac/sbi/models"
)

// send uplink N2SmInfo to AMF or SMF
func (ueCtx *UeContext) sendN2SmInfoUplink(msg *models.N2SmInfoUplinkTransport) {
	if context.NasSplit() { //send N2SmInfo directly to SMF
		for _, item := range msg.Transfers {
			if smCtx := ueCtx.findSmContext(uint8(item.SessionId)); smCtx == nil {
				ueCtx.Errorf("Session %d not found to forward N2SmInfo to core", item.SessionId)
			} else {
				switch item.N2SmInfoType {
				case models.N2SMINFOTYPE_PDU_RES_SETUP_RSP:
					smCtx.receiveSetupResponse(item.N2SmInfo, true)
				case models.N2SMINFOTYPE_PDU_RES_SETUP_FAIL:
					smCtx.receiveSetupResponse(item.N2SmInfo, false)
				case models.N2SMINFOTYPE_PDU_RES_REL_RSP:
					smCtx.receiveReleaseResponse(item.N2SmInfo)
				case models.N2SMINFOTYPE_PDU_RES_MOD_RSP:
					smCtx.receiveModifyResponse(item.N2SmInfo, true)
				case models.N2SMINFOTYPE_PDU_RES_MOD_FAIL:
					smCtx.receiveModifyResponse(item.N2SmInfo, false)
				case models.N2SMINFOTYPE_PDU_RES_NTY:
					smCtx.receiveNotify(item.N2SmInfo, false)
				case models.N2SMINFOTYPE_PDU_RES_NTY_REL:
					smCtx.receiveNotify(item.N2SmInfo, true)
				case models.N2SMINFOTYPE_PDU_RES_MOD_IND:
					smCtx.receiveModifyIndication(item.N2SmInfo)
				default:
				}
			}
		}
	} else { //receive to AMF
		if err := uectx.N2SmInfoUplink(ueCtx.amfCli, ueCtx.amfUeId, msg); err != nil {
			ueCtx.Errorf("Fail to forward N2SmInfoUplink from gnB to core: %+v", err)
		} else {
			ueCtx.Info("N2SmInfoUplink from gnB is forwarded to core")
		}
	}
}
