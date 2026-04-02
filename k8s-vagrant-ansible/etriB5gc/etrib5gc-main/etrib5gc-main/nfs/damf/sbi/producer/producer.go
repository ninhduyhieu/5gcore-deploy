package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/amf/n2nas"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
	"net/http"
)

type producerImpl struct {
	*logrus.Entry
}

var notFoundProblem *models.ProblemDetails = &models.ProblemDetails{
	Detail: "Uecontext not found",
	Status: http.StatusNotFound,
}

func RouteGroups() []httpw.RouteGroup {
	prod := &producerImpl{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "producer",
		}),
	}
	return []httpw.RouteGroup{
		{
			Name:   n2nas.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[n2nas.Producer](n2nas.Routes(), prod),
		},
	}
}
