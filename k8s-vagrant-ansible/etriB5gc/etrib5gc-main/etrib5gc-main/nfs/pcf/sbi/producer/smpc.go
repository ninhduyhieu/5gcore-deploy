package producer

import (
	"github.com/reogac/sbi/models"
)

func (p *Producer) HandleDeleteSMPolicy(smPolicyId string, body *models.SmPolicyDeleteData) (prob *models.ProblemDetails) {
	p.Debugf("Receive a DeleteSMPolicy request")
	p.smPols.DeleteSmPolicy(smPolicyId)
	return
}

func (p *Producer) HandleGetSMPolicy(smPolicyId string) (rsp *models.SmPolicyControl, prob *models.ProblemDetails) {
	p.Debugf("Receive a GetSMPolicy request")
	if smPol := p.smPols.GetSmPolicy(smPolicyId); smPol == nil {
		prob = NoSmPolProb
	} else {
		rsp = smPol.SmPolicyControl()
	}
	return
}
func (p *Producer) HandleUpdateSMPolicy(smPolicyId string, body *models.SmPolicyUpdateContextData) (rsp *models.SmPolicyDecision, prob *models.ProblemDetails) {
	p.Debugf("Receive a UpdateSMPolicy request")
	if smPol := p.smPols.GetSmPolicy(smPolicyId); smPol == nil {
		prob = NoSmPolProb
	} else if err := smPol.UpdateSmPolicy(body); err != nil {
		p.Errorf("Update SmPolicy failed: %+v", err)
		prob = InternalProb
	} else {
		rsp = smPol.SmPolicyDecision()
	}

	return
}
func (p *Producer) HandleCreateSMPolicy(body *models.SmPolicyContextData) (headers map[string]string, rsp *models.SmPolicyDecision, prob *models.ProblemDetails) {
	p.Debugf("Receive a CreateSMPolicy request")

	if smPol, err := p.smPols.CreateSmPolicy(body); err != nil {
		p.Errorf("Create SmPolicy failed: %+v", err)
		prob = InternalProb
	} else {
		rsp = smPol.SmPolicyDecision()
		headers = make(map[string]string)
		headers["Location"] = smPol.Id()
	}
	return
}
