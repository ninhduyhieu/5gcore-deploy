package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/smf/n1n2ul"
	"github.com/reogac/sbi/apis/smf/pdu"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
)

type producerImpl struct {
	*logrus.Entry
}

func RouteGroups() []httpw.RouteGroup {
	prod := &producerImpl{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "prod",
		}),
	}

	return []httpw.RouteGroup{
		{
			Name:   pdu.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[pdu.Producer](pdu.Routes(), prod),
		},
		{
			Name:   n1n2ul.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[n1n2ul.Producer](n1n2ul.Routes(), prod),
		},
	}
}
