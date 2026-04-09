package sm

import (
	"etrib5gc/mesh"
	"fmt"
	"github.com/reogac/sbi"
	"github.com/reogac/sbi/apis/smf/pdu"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
)

type SmContext struct {
	*logrus.Entry
	id           uint8
	ref          string
	isGpp        bool
	isHomeRouted bool
	smfCli       sbi.ConsumerClient
	snssai       models.AllowedSnssai
	dnn          string
	statusUri    string
	upCnxState   models.UpCnxState
}

func NewSmContext(l *logrus.Entry, isGpp bool, sId uint8, snssai models.AllowedSnssai, isHomeRouted bool, dnn string, cli sbi.ConsumerClient) *SmContext {
	smCtx := &SmContext{
		Entry: l.WithFields(
			logrus.Fields{
				"sessionId": fmt.Sprintf("%d", sId),
			},
		),
		id:           sId,
		isGpp:        isGpp,
		dnn:          dnn,
		snssai:       snssai,
		smfCli:       cli,
		isHomeRouted: isHomeRouted,
	}
	return smCtx
}

// request SMF to create SmContext
func (smCtx *SmContext) CreateContextAtSmf(info *models.SmContextCreateData, n1Sm []byte) (*models.PostSmContextsResponse, *models.PostSmContextsErrorResponse, error) {
	callback := mesh.EndpointInfo()

	info.PduSessionId = new(int)
	*info.PduSessionId = int(smCtx.id)
	if smCtx.isHomeRouted { //HOME-ROUTED
		info.SNssai = smCtx.snssai.MappedHomeSnssai
	} else { //LBO
		info.SNssai = &smCtx.snssai.AllowedSnssai
	}
	info.Dnn = smCtx.dnn
	//dat.ServingNfId =context.NfId
	//dat.Guami =&context.ServedGuamiList[0]
	info.SmContextStatusUri = smCtx.statusUri

	req := &models.PostSmContextsRequest{
		BinaryDataN1SmMessage: n1Sm,
		JsonData:              info,
	}

	if headers, rsp, ersp, err := pdu.PostSmContexts(smCtx.smfCli, callback, req); err != nil {
		return nil, nil, utils.WrapError("Creater SmContext at SMF", err)
	} else {
		if smCtx.ref = extractSmContextRef(headers); len(smCtx.ref) == 0 {
			return nil, nil, fmt.Errorf("SMF does not send a SmContext reference value")
		}

		return rsp, ersp, nil
	}
}

// ask SMF to release session
func (smCtx *SmContext) ReleaseSmContext(req *models.SmContextReleaseData, n2SmInfoType models.N2SmInfoType, n2SmInfo []byte) (*models.SmContextReleasedData, error) {
	smCtx.Debug("Ask SMF to release SmContext")

	req.N2SmInfoType = n2SmInfoType
	return pdu.ReleaseSmContext(smCtx.smfCli, smCtx.ref, &models.ReleaseSmContextRequest{
		JsonData:                  req,
		BinaryDataN2SmInformation: n2SmInfo,
	})
}

// send UpdaSmContextRequest to SMF
func (smCtx *SmContext) SendUpdateSmContext(msg *models.SmContextUpdateData, n1Sm []byte, n2SmInfo []byte) (*models.UpdateSmContextResponse, *models.UpdateSmContextErrorResponse, error) {
	if rsp, ersp, err := pdu.UpdateSmContext(smCtx.smfCli, smCtx.ref, &models.UpdateSmContextRequest{
		JsonData:                  msg,
		BinaryDataN1SmMessage:     n1Sm,
		BinaryDataN2SmInformation: n2SmInfo,
	}); err != nil {
		return nil, nil, utils.WrapError("Send UpdateSmContext to SMF", err)
	} else {
		if rsp != nil && rsp.JsonData != nil && len(rsp.JsonData.UpCnxState) > 0 {
			smCtx.upCnxState = rsp.JsonData.UpCnxState
		} else if ersp != nil && ersp.JsonData != nil && len(ersp.JsonData.UpCnxState) > 0 {
			smCtx.upCnxState = rsp.JsonData.UpCnxState
		}
		return rsp, ersp, nil
	}
}

// send UpdaSmContextRequest to SMF
func (smCtx *SmContext) UpdateSmContext(msg *models.SmContextUpdateData, n1Sm []byte, n2SmInfo []byte, fn func(*models.UpdateSmContextResponse, *models.UpdateSmContextErrorResponse)) error {
	if rsp, ersp, err := pdu.UpdateSmContext(smCtx.smfCli, smCtx.ref, &models.UpdateSmContextRequest{
		JsonData:                  msg,
		BinaryDataN1SmMessage:     n1Sm,
		BinaryDataN2SmInformation: n2SmInfo,
	}); err != nil {
		return utils.WrapError("Send UpdateSmContext", err)
	} else {
		if rsp != nil && rsp.JsonData != nil && len(rsp.JsonData.UpCnxState) > 0 {
			smCtx.upCnxState = rsp.JsonData.UpCnxState
		} else if ersp != nil && ersp.JsonData != nil && len(ersp.JsonData.UpCnxState) > 0 {
			smCtx.upCnxState = rsp.JsonData.UpCnxState
		}

		if fn != nil {
			fn(rsp, ersp)
		}
	}
	return nil
}

func (smCtx *SmContext) GetUpCnxState() models.UpCnxState {
	return smCtx.upCnxState
}

func (smCtx *SmContext) GetId() uint8 {
	return smCtx.id
}

func (smCtx *SmContext) IsGpp() bool {
	return smCtx.isGpp
}

func (smCtx *SmContext) GetSlice() *models.Snssai {
	return &smCtx.snssai.AllowedSnssai
}
