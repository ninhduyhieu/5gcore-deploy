package producer

import (
	"etrib5gc/logctx"
	"etrib5gc/mesh/httpdp"
	"github.com/reogac/sbi/apis/upf/n4sbi"
	"github.com/reogac/utils/httpw"
)

type Producer struct {
	logctx.LogWriter
}

func RouteGroups() []httpw.RouteGroup {
	prod := &Producer{
		LogWriter: logctx.WithFields(logctx.Fields{
			"mod": "producer",
		}),
	}
	return []httpw.RouteGroup{
		{
			Name:   n4sbi.PATH_ROOT,
			Routes: httpdp.MakeHttpRoutes[n4sbi.Producer](n4sbi.Routes(), prod),
		},
	}

}
