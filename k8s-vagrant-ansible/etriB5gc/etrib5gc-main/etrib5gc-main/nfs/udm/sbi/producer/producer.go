package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/udm/sdm"
	"github.com/reogac/sbi/apis/udm/ueauth"
	"github.com/reogac/sbi/apis/udm/uecm"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const (
	REQUEST_TIMEOUT time.Duration = 5 * time.Second
)

type Producer struct {
	*logrus.Entry
}

func RouteGroups() []httpw.RouteGroup {
	prod := &Producer{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "prod",
		}),
	}
	//	ee.Service(prod),
	//	pp.Service(prod),
	//	mt.Service(prod),
	//	niddau.Service(prod),

	return []httpw.RouteGroup{
		{
			Name:   sdm.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[sdm.Producer](sdm.Routes(), prod),
		},
		{
			Name:   ueauth.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[ueauth.Producer](ueauth.Routes(), prod),
		},
		{
			Name:   uecm.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[uecm.Producer](uecm.Routes(), prod),
		},
	}
}

func (p *Producer) ueNotFound(supi string) *models.ProblemDetails {
	p.Errorf("UeContext[supi=%s] not existed", supi)
	return &models.ProblemDetails{
		Status: http.StatusNotFound,
		Detail: "UeContext not existed",
	}
}
func (p *Producer) internalError(details string) *models.ProblemDetails {
	return &models.ProblemDetails{
		Status: http.StatusInternalServerError,
		Detail: details,
	}
}
