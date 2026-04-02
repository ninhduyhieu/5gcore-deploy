package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/pran/handover"
	"github.com/reogac/sbi/apis/pran/n1n2dl"
	"github.com/reogac/sbi/apis/pran/nasdl"
	"github.com/reogac/sbi/apis/pran/subs"
	"github.com/reogac/sbi/apis/pran/uectx"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
	"net/http"
)

type producerImpl struct {
	*logrus.Entry
}

var internalProblem = &models.ProblemDetails{
	Status: http.StatusInternalServerError,
	Detail: "Fail to handle Sbi request",
}

func RouteGroups() []httpw.RouteGroup {
	prod := &producerImpl{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "producer",
		}),
	}
	return []httpw.RouteGroup{
		{
			Name:   uectx.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[uectx.Producer](uectx.Routes(), prod),
		},
		{
			Name:   nasdl.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[nasdl.Producer](nasdl.Routes(), prod),
		},
		{
			Name:   n1n2dl.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[n1n2dl.Producer](n1n2dl.Routes(), prod),
		},
		{
			Name:   handover.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[handover.Producer](handover.Routes(), prod),
		},
		{
			Name:   subs.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[subs.Producer](subs.Routes(), prod),
		},
	}
}
