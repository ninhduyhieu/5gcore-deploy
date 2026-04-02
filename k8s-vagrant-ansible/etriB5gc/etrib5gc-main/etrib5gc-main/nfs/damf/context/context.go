package context

import (
	"etrib5gc/logctx"
	"etrib5gc/nfs/damf/config"
	"github.com/reogac/sbi/models"
	"github.com/sirupsen/logrus"
)

const (
	T3502_VALUE uint8  = 100 //miliseconds
	T3560_VALUE uint16 = 100 //miliseconds
	MAX_NUM_UES uint   = 10000
)

type DamfContext struct {
	plmnid    models.PlmnId
	t3502     uint8
	t3560     uint16
	maxNumUes uint
}

var _ctx *DamfContext
var log *logrus.Entry

func Init(cfg *config.Config) {
	if _ctx != nil {
		return
	}

	log = logctx.Entry(logrus.Fields{
		"mod": "context",
	})

	_ctx = &DamfContext{
		t3502:     cfg.T3502,
		t3560:     cfg.T3560,
		maxNumUes: MAX_NUM_UES,
		plmnid:    models.PlmnId(cfg.PlmnId),
	}
	if _ctx.t3502 == 0 {
		_ctx.t3502 = T3502_VALUE
	}
	if _ctx.t3560 == 0 {
		_ctx.t3560 = T3560_VALUE
	}
}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnid
}
func MaxNumUes() int {
	return int(_ctx.maxNumUes)
}

func GetT3502() uint8 {
	return _ctx.t3502
}

func GetT3560() uint16 {
	return _ctx.t3560
}

func GetAbba() []byte {
	return []byte{0x00, 0x00}
}
