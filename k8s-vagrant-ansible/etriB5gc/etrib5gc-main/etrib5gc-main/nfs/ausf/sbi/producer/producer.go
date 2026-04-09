package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/ausf/ueauth"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
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

	return []httpw.RouteGroup{
		{
			Name:   ueauth.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[ueauth.Producer](ueauth.Routes(), prod),
		},
	}
}
