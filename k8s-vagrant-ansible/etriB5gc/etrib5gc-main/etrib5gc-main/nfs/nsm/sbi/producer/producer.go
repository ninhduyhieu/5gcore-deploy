package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/nsm/amfman"
	"github.com/reogac/sbi/apis/nsm/conf"
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
			Name:   amfman.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[amfman.Producer](amfman.Routes(), prod),
		},
		{
			Name:   conf.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[conf.Producer](conf.Routes(), prod),
		},
	}
}
