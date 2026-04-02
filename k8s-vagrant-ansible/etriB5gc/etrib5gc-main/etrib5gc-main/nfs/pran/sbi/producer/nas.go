package producer

import (
	"etrib5gc/nfs/pran/ue"
	"fmt"
	"github.com/reogac/sbi/models"
	"net/http"
)

// AMF send downlink Nas toward UE
func (p *producerImpl) HandleNasDl(ueId int64, msg *models.NasDownlinkTransport) (prob *models.ProblemDetails) {
	p.Debugf("Receive NasDl [ueId=%d]", ueId)
	if ueCtx := ue.FindWithLocalId(ueId); ueCtx == nil {
		p.Errorf("UeContext not found [id= %d] to handle Nas downlink", ueId)
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("UE context not found for %d", ueId),
		}
	} else {
		ue.ReceiveSbiEvent[models.NasDownlinkTransport](ueCtx, ue.NAS_DOWNLINK, msg)
	}
	return
}
