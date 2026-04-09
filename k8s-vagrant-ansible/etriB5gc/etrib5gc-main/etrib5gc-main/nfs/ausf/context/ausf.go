package context

import (
	"etrib5gc/logctx"
	"etrib5gc/nfs/ausf/config"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
)

type AusfContext struct {
	group  string
	plmnId models.PlmnId
	authId int64
}

var _ctx *AusfContext
var log *logrus.Entry

func Init(cfg *config.Config) {
	if _ctx != nil {
		return
	}

	log = logctx.Entry(logrus.Fields{
		"mod": "context",
	})

	_ctx = &AusfContext{
		group:  cfg.Group,
		plmnId: models.PlmnId(cfg.PlmnId),
	}
}

func IsNetworkAuthorized(netname string) bool {
	return true
}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnId
}

func AllocateAuthId() int64 {
	_ctx.authId++
	return _ctx.authId
}
