package producer

import (
	"github.com/reogac/sbi/models"
)

func (p *Producer) HandleReportObservedEventTriggersForIndividualAMPolicyAssociation(polAssoId string, body *models.PolicyAssociationUpdateRequest) (rsp *models.PolicyUpdate, prob *models.ProblemDetails) {
	p.Debugf("Receive a ReportObservedEventTriggers")
	return
}

func (p *Producer) HandleCreateIndividualAMPolicyAssociation(body *models.PolicyAssociationRequest) (headers map[string]string, rsp *models.PolicyAssociation, prob *models.ProblemDetails) {
	p.Debugf("Receive a CreateIndividualAMPolicyAssociation request")
	if amPol, err := p.amPols.CreateAmPolicy(body); err != nil {
		prob = AmPolNotCreatedProb
	} else {
		rsp = amPol.PolicyAssociation()
		headers = make(map[string]string)
		headers["Location"] = amPol.Id()
	}

	return
}

func (p *Producer) HandleDeleteIndividualAMPolicyAssociation(polAssoId string) (prob *models.ProblemDetails) {
	p.Debugf("Receive a DeleteIndividualAMPolicyAssociation request")
	p.amPols.DeleteAmPolicy(polAssoId)
	return
}

func (p *Producer) HandleReadIndividualAMPolicyAssociation(polAssoId string) (rsp *models.PolicyAssociation, prob *models.ProblemDetails) {
	p.Debugf("Receive a ReadIndividualAMPolicyAssociation request")
	if amPol := p.amPols.GetAmPolicy(polAssoId); amPol == nil {
		prob = NoAmPolProb
	} else {
		rsp = amPol.PolicyAssociation()
	}

	return
}
