package producer

import (
	"context"
	"etrib5gc/nfs/udm/ue"
	"github.com/reogac/sbi/apis/udm/sdm"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *Producer) HandleGetAmData(params *sdm.GetAmDataParams) (headers map[string]string, rsp *models.AccessAndMobilitySubscriptionData, prob *models.ProblemDetails) {
	p.Debugf("Receive a GetAmData request")
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if rsp, err = ue.GetAmData(ctx, params.Supi, params.PlmnId, params.SupportedFeatures); err != nil {
		p.Errorf("GetDmData fails:%+v", err)
		prob = &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: err.Error(),
		}
	}
	return
}

func (p *Producer) HandleGetEcrData(params *sdm.GetEcrDataParams) (headers map[string]string, rsp *models.EnhancedCoverageRestrictionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetV2xData(params *sdm.GetV2xDataParams) (headers map[string]string, rsp *models.V2xSubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetMbsData(params *sdm.GetMbsDataParams) (headers map[string]string, rsp *models.MbsSubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleSNSSAIsAck(supi string, body *models.AcknowledgeInfo) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleSubscribeToSharedData(body *models.SdmSubscription) (headers map[string]string, rsp *models.SdmSubscription, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetNSSAI(params *sdm.GetNSSAIParams) (headers map[string]string, rsp *models.Nssai, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetLcsBcaData(params *sdm.GetLcsBcaDataParams) (headers map[string]string, rsp *models.LcsBroadcastAssistanceTypesData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleModify(params *sdm.ModifyParams, body *models.SdmSubsModification) (rsp *models.Schema, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetProseData(params *sdm.GetProseDataParams) (headers map[string]string, rsp *models.ProseSubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetUcData(params *sdm.GetUcDataParams) (headers map[string]string, rsp *models.UcSubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleSubscribe(ueId string, body *models.SdmSubscription) (headers map[string]string, rsp *models.SdmSubscription, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUnsubscribe(params *sdm.UnsubscribeParams) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetMultipleIdentifiers(params *sdm.GetMultipleIdentifiersParams) (headers map[string]string, rsp *map[string]models.SupiInfo, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetDataSets(params *sdm.GetDataSetsParams) (headers map[string]string, rsp *models.SubscriptionDataSets, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetSmfSelData(params *sdm.GetSmfSelDataParams) (headers map[string]string, rsp *models.SmfSelectionSubscriptionData, prob *models.ProblemDetails) {
	p.Debugf("Receive a GetSmfSelData request")
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if rsp, err = ue.GetSmfSelData(ctx, params); err != nil {
		p.Errorf("GetSmfSelData fails:%+v", err)
		prob = p.internalError("Fail to get SmfSelectionSubscriptionData")
	}
	return
}

func (p *Producer) HandleCAGAck(supi string, body *models.AcknowledgeInfo) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUnsubscribeForSharedData(subscriptionId string) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetTraceConfigData(params *sdm.GetTraceConfigDataParams) (headers map[string]string, rsp *models.TraceDataResponse, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetSupiOrGpsi(params *sdm.GetSupiOrGpsiParams) (headers map[string]string, rsp *models.IdTranslationResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetUeCtxInAmfData(params *sdm.GetUeCtxInAmfDataParams) (rsp *models.UeContextInAmfData, prob *models.ProblemDetails) {
	p.Debugf("Receive a GetUeCtxInAmfData")
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	var err error
	if rsp, err = ue.GetUeCtxInAmfData(ctx, params.Supi); err != nil {
		p.Errorf("GetUeCtxInAmfData fails:%+v", err)
		prob = p.internalError("Fail to get UeContextInAmfData")
	}
	return
}

func (p *Producer) HandleGetUeCtxInSmfData(params *sdm.GetUeCtxInSmfDataParams) (rsp *models.UeContextInSmfData, prob *models.ProblemDetails) {
	p.Debugf("Receive a GetUeCtxInSmfData Request")

	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	var err error
	if rsp, err = ue.GetUeCtxInSmfData(ctx, params.Supi); err != nil {
		p.Errorf("GetUeCtxInSmfData fails:%+v", err)
		prob = p.internalError("Fail to get UeContextInSmfData")
	}
	return
}
func (p *Producer) HandleGetSmData(params *sdm.GetSmDataParams) (headers map[string]string, rsp *models.SmSubsData, prob *models.ProblemDetails) {
	p.Debugf("Receive a GetSmData Request")
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if rsp, err = ue.GetSmData(ctx, params); err != nil {
		p.Errorf("GetSmData fails:%+v", err)
		prob = p.internalError("Fail to get SmData")
	}

	return
}

func (p *Producer) HandleGetSmsMngtData(params *sdm.GetSmsMngtDataParams) (headers map[string]string, rsp *models.SmsManagementSubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUpuAck(supi string, body *models.AcknowledgeInfo) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetSmsData(params *sdm.GetSmsDataParams) (headers map[string]string, rsp *models.SmsSubscriptionData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetLcsPrivacyData(params *sdm.GetLcsPrivacyDataParams) (headers map[string]string, rsp *models.LcsPrivacyData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleSorAckInfo(supi string, body *models.AcknowledgeInfo) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUpdateSORInfo(supi string, body *models.SorUpdateInfo) (rsp *models.SorInfo, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetUeCtxInSmsfData(params *sdm.GetUeCtxInSmsfDataParams) (rsp *models.UeContextInSmsfData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetSharedData(params *sdm.GetSharedDataParams) (headers map[string]string, rsp *[]models.SharedData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetIndividualSharedData(params *sdm.GetIndividualSharedDataParams) (headers map[string]string, rsp *models.SharedData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetLcsMoData(params *sdm.GetLcsMoDataParams) (headers map[string]string, rsp *models.LcsMoData, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetGroupIdentifiers(params *sdm.GetGroupIdentifiersParams) (headers map[string]string, rsp *models.GroupIdentifiers, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleModifySharedDataSubs(params *sdm.ModifySharedDataSubsParams, body *models.SdmSubsModification) (rsp *models.Schema, prob *models.ProblemDetails) {
	return
}
