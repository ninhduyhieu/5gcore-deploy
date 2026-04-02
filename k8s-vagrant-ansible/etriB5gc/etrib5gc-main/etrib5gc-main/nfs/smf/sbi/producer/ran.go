package producer

import (
	"github.com/reogac/sbi/models"
)

func (p *producerImpl) HandleSessionResourceNotify(smCtxRef string, msg *models.SessionResourceNotification) (prob *models.ProblemDetails) {
	p.Debugf("Receive SessionResourceNotification")
	/*
		var err error
		if ranue := ranuecontext.FindRanUe(ueid); ranue == nil {
			err = fmt.Errorf("RanUe not found [amfUeid=%d]", ueid)
			p.Error(err.Error())
			prob = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Detail: err.Error()}
			p.Error(err.Error())
			return
		} else {
			prob = ranue.HandlePduSessResNot(msg)
		}
	*/
	return
}

func (p *producerImpl) HandleSessionResourceModifyIndication(smCtxRef string, msg *models.SessionResourceModifyIndication) (rsp *models.SessionResourceModifyConfirm, prob *models.ProblemDetails) {
	p.Debugf("Receive SessionResourceModifyIndication")
	/*
		var err error
		if ranue := ranuecontext.FindRanUe(ueid); ranue == nil {
			err = fmt.Errorf("RanUe not found [amfUeid=%d]", ueid)
			p.Error(err.Error())
			prob = &models.ProblemDetails{
				Status: http.StatusNotFound,
				Detail: err.Error()}
			p.Error(err.Error())
			return
		} else {
			prob = ranue.HandlePduSessResModInd(msg)
		}
	*/

	return
}

func (p *producerImpl) HandleN1SmUplink(ref string, pdu []byte) (prob *models.ProblemDetails) {
	p.Debugf("Receive N1SmUplink")
	return
}
