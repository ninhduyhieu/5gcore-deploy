package producer

import (
	"context"
	"etrib5gc/nfs/udm/ue"
	"github.com/reogac/sbi/apis/udm/uecm"
	"github.com/reogac/sbi/models"
)

func (p *Producer) HandleRetrieveSmfRegistration(params *uecm.RetrieveSmfRegistrationParams) (rsp *models.SmfRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGet3GppSmsfRegistration(params *uecm.Get3GppSmsfRegistrationParams) (rsp *models.SmsfRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetNon3GppSmsfRegistration(params *uecm.GetNon3GppSmsfRegistrationParams) (rsp *models.SmsfRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleIpSmGwDeregistration(ueId string) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetNwdafRegistration(params *uecm.GetNwdafRegistrationParams) (rsp *[]models.NwdafRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandlePeiUpdate(ueId string, body *models.PeiUpdateInfo) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetNon3GppRegistration(params *uecm.GetNon3GppRegistrationParams) (rsp *models.AmfNon3GppAccessRegistration, prob *models.ProblemDetails) {

	return
}

func (p *Producer) HandleNon3GppRegistration(ueId string, body *models.AmfNon3GppAccessRegistration) (rsp *models.AmfNon3GppAccessRegistration, prob *models.ProblemDetails) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if err := ue.CreateNon3GppRegistration(ctx, ueId, body); err != nil {
		p.Errorf("Fail to handle AmfNon3GppAccessRegistration for %s: %+v", ueId, err)
		prob = p.internalError("Fail to handle AmfNon3GppAccessRegistration")
	}

	return
}

func (p *Producer) HandleNwdafDeregistration(params *uecm.NwdafDeregistrationParams) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleSmfDeregistration(params *uecm.SmfDeregistrationParams) (prob *models.ProblemDetails) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if err := ue.DeregisterPduSession(ctx, params.UeId, params.PduSessionId); err != nil {
		p.Errorf("Fail to remove Pdu session: %+v", err)
		prob = p.internalError("Fail to remove Pdu session")
	}

	return
}

func (p *Producer) HandleThreeGppSmsfRegistration(ueId string, body *models.SmsfRegistration) (headers map[string]string, rsp *models.SmsfRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleNon3GppSmsfDeregistration(params *uecm.Non3GppSmsfDeregistrationParams) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUpdateSmfRegistration(params *uecm.UpdateSmfRegistrationParams, body *models.SmfRegistrationModification) (rsp *models.PatchResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleThreeGppSmsfDeregistration(params *uecm.ThreeGppSmsfDeregistrationParams) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleSendRoutingInfoSm(ueId string, body *models.RoutingInfoSmRequest) (rsp *models.RoutingInfoSmResponse, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleDeregAMF(ueId string, body *models.AmfDeregInfo) (prob *models.ProblemDetails) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if err := ue.DeregisterUe(ctx, ueId); err != nil {
		p.Errorf("Fail to deregister UE %s: %+v", ueId, err)
		prob = p.internalError("Fail to deregister UE")
	}

	return
}

func (p *Producer) HandleUpdateRoamingInformation(ueId string, body *models.RoamingInfoUpdate) (rsp *models.RoamingInfoUpdate, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleIpSmGwRegistration(ueId string, body *models.IpSmGwRegistration) (rsp *models.IpSmGwRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUpdateNwdafRegistration(params *uecm.UpdateNwdafRegistrationParams, body *models.NwdafRegistrationModification) (rsp *models.Schema, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleRegistration(params *uecm.RegistrationParams, body *models.SmfRegistration) (rsp *models.SmfRegistration, prob *models.ProblemDetails) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if rsp, err = ue.RegisterPduSession(ctx, params.UeId, params.PduSessionId, body); err != nil {
		p.Errorf("Fail to register Pdu session: %+v", err)
		prob = p.internalError("Fail to register Pdu session")
	}
	return
}

func (p *Producer) HandleNon3GppSmsfRegistration(ueId string, body *models.SmsfRegistration) (headers map[string]string, rsp *models.SmsfRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetIpSmGwRegistration(ueId string) (rsp *models.IpSmGwRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUpdate3GppRegistration(params *uecm.Update3GppRegistrationParams, body *models.Amf3GppAccessRegistrationModification) (rsp *models.PatchResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetLocationInfo(params *uecm.GetLocationInfoParams) (rsp *models.LocationInfo, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetRegistrations(params *uecm.GetRegistrationsParams) (rsp *models.RegistrationDataSets, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleUpdateNon3GppRegistration(params *uecm.UpdateNon3GppRegistrationParams, body *models.AmfNon3GppAccessRegistrationModification) (rsp *models.PatchResult, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleTriggerPCSCFRestoration(body *models.TriggerRequest) (prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGet3GppRegistration(params *uecm.Get3GppRegistrationParams) (rsp *models.Amf3GppAccessRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleThreeGppRegistration(ueId string, body *models.Amf3GppAccessRegistration) (rsp *models.Amf3GppAccessRegistration, prob *models.ProblemDetails) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()
	if err := ue.Create3GppRegistration(ctx, ueId, body); err != nil {
		p.Errorf("Fail to handle Amf3GppAccessRegistration for UE %s: %+v", ueId, err)
		prob = p.internalError("Fail to handle Amf3GppAccessRegistration")
	}

	return
}

func (p *Producer) HandleNwdafRegistration(params *uecm.NwdafRegistrationParams, body *models.NwdafRegistration) (rsp *models.NwdafRegistration, prob *models.ProblemDetails) {
	return
}

func (p *Producer) HandleGetSmfRegistration(params *uecm.GetSmfRegistrationParams) (rsp *models.SmfRegistrationInfo, prob *models.ProblemDetails) {
	return
}
