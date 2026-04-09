package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"etrib5gc/nfs/pcf/ampc"
	"etrib5gc/nfs/pcf/smpc"
	"github.com/reogac/sbi/apis/pcf/ampol"
	"github.com/reogac/sbi/apis/pcf/smpol"
	"github.com/reogac/utils/httpw"
	"github.com/sirupsen/logrus"
)

type Producer struct {
	amPols *ampc.AmPolicies
	smPols *smpc.SmPolicies
	*logrus.Entry
}

func RouteGroups() []httpw.RouteGroup {
	prod := &Producer{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "producer",
		}),
		amPols: ampc.GetAmPolicies(),
		smPols: smpc.GetSmPolicies(),
	}
	//ee.Service(prod),
	//pa.Service(prod),
	//btdpc.Service(prod),

	return []httpw.RouteGroup{
		{
			Name:   ampol.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[ampol.Producer](ampol.Routes(), prod),
		},
		{
			Name:   smpol.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[smpol.Producer](smpol.Routes(), prod),
		},
	}
}
