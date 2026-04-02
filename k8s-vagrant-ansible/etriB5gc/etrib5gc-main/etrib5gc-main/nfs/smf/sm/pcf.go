package sm

import (
	"github.com/reogac/sbi/apis/pcf/smpol"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
)

func (smCtx *SmContext) createSmPol() (err error) {
	if smCtx.smPol != nil {
		smCtx.Warnf("SmPolicy was created, do not request PCF")
		return
	}

	req := &models.SmPolicyContextData{}
	req.Supi = smCtx.supi
	req.Dnn = smCtx.dnn
	req.PduSessionId = int(smCtx.pduSessionId)
	req.AccessType = smCtx.access
	req.PduSessionType = smCtx.getPduSessionType()
	req.Pei = smCtx.pei
	req.SliceInfo = *smCtx.snssai
	//should be from subscribed data
	req.SubsSessAmbr = &models.Ambr{
		Uplink:   "100 Mbps",
		Downlink: "100 Mbps",
	}
	//should be from subscribed data
	req.SubsDefQos = &models.SubscribedDefaultQos{
		FiveQi: 9,
		Arp: models.Arp{
			PriorityLevel: 10,
			PreemptCap:    models.PREEMPTIONCAPABILITY_NOT_PREEMPT,
			PreemptVuln:   models.PREEMPTIONVULNERABILITY_NOT_PREEMPTABLE,
		},
		PriorityLevel: new(int),
	}
	*req.SubsDefQos.PriorityLevel = 10
	var headers map[string]string
	if headers, smCtx.smPol, err = smpol.CreateSMPolicy(smCtx.pcfCli, req); err != nil {
		err = utils.WrapError("Send CreateSMPolicy to PCF", err)
	} else {
		if smCtx.smPolId = extractIdFromLocation(headers); len(smCtx.smPolId) == 0 {
			smCtx.Warnf("PCF does not send SmPolId")
		} else {
			smCtx.Infof("SmPolId=%s", smCtx.smPolId)
		}
	}
	return
}

func (smCtx *SmContext) deleteSmPol() error {
	if len(smCtx.smPolId) > 0 {
		return smpol.DeleteSMPolicy(smCtx.pcfCli, smCtx.smPolId, &models.SmPolicyDeleteData{})
	}
	return nil
}
