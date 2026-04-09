package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/amf/callback"
	"github.com/reogac/sbi/apis/amf/comm"
	"github.com/reogac/sbi/apis/amf/handover"
	"github.com/reogac/sbi/apis/amf/n2nas"
	"github.com/reogac/sbi/apis/amf/uectx"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
)

type producerImpl struct {
	*logrus.Entry
}

func RouteGroups() []httpw.RouteGroup {
	prod := &producerImpl{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "sbi",
		}),
	}

	return []httpw.RouteGroup{
		{
			Name:   uectx.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[uectx.Producer](uectx.Routes(), prod),
		},
		{
			Name:   comm.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[comm.Producer](comm.Routes(), prod),
		},
		{
			Name:   handover.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[handover.Producer](handover.Routes(), prod),
		}, {
			Name:   n2nas.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[n2nas.Producer](n2nas.Routes(), prod),
		}, {
			Name:   callback.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[callback.Producer](callback.Routes(), prod),
		},
	}
}
