package producer

import (
	"etrib5gc/nfs/nsm/context"
	"github.com/reogac/sbi/apis/nsm/conf"
	"github.com/reogac/sbi/models"
	"net/http"
)

func (p *Producer) HandleGetNssfConfiguration() (rsp *models.NssfConfiguration, prob *models.ProblemDetails) {
	p.Debugf("Receive a NSSF configuration request")
	rsp = context.GetNssfConfiguration()
	return
}

func (p *Producer) HandleGetSessionManagementConfiguration(params *conf.GetSessionManagementConfigurationParams) (rsp *models.SessionManagementConfiguration, prob *models.ProblemDetails) {
	p.Debugf("Receive a session management configuration request")
	if rsp = context.GetSessionManagementConfiguration(params.Uuid, params.Slice); rsp == nil {
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: "Session management configuration not found for the slice",
		}
	}
	return
}
func (p *Producer) HandleGetUdrConfiguration() (rsp *models.UdrConfiguration, prob *models.ProblemDetails) {
	p.Debugf("Receive an UDR configuration request")
	rsp = context.GetUdrConfiguration()
	return
}

func (p *Producer) HandleGetUdmConfiguration() (rsp *models.UdmConfiguration, prob *models.ProblemDetails) {
	p.Debugf("Receive an UDM configuration request")
	rsp = context.GetUdmConfiguration()
	return
}

func (p *Producer) HandleGetUserPlaneConfiguration(body *models.UserPlaneConfigurationRequest) (rsp *models.UserPlaneConfigurationResponse, prob *models.ProblemDetails) {
	p.Debugf("Receive an User Plane configuration request")
	if rsp = context.GetUserPlaneConfiguration(body.Slices); rsp == nil {
		prob = &models.ProblemDetails{
			Status: http.StatusNotFound,
			Detail: "User plane configuration not found for slices",
		}
	}
	return
}
func (p *Producer) HandleGetSeppConfiguration() (rsp *models.SeppConfiguration, prob *models.ProblemDetails) {
	p.Debugf("Receive a SEPP configuration request")
	rsp = context.GetSeppConfiguration()
	return
}
