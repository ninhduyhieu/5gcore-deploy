package context

import (
	"etrib5gc/common"
	"etrib5gc/logctx"
	"etrib5gc/nfs/upf/config"
	"github.com/reogac/sbi/models"
	"net"
)

type InfInfo struct {
	Addr net.IP
	Mtu  uint32
}

type UpfContext struct {
	logctx.LogWriter
	id       string
	plmnid   models.PlmnId
	slices   map[string]models.Snssai
	infs     map[string]InfInfo
	dnnList  []string // anchoring if non-empty
	dnaiList []string // anchoring if non-empty
}

var _ctx *UpfContext

func Init(cfg *config.Config) {
	if _ctx != nil {
		return
	}
	_ctx = &UpfContext{
		LogWriter: logctx.WithFields(logctx.Fields{
			"mod": "context",
		}),
		id:     cfg.Id,
		plmnid: models.PlmnId(cfg.PlmnId),
		infs:   make(map[string]InfInfo),
		slices: make(map[string]models.Snssai),
	}

	//build slice list
	for _, s := range cfg.Slices {
		_ctx.slices[s.String()] = s
	}

	//validate and build network interfaces
	for _, item := range cfg.IfList {
		if len(item.Name) == 0 {
			_ctx.Warnf("Empty network interface name")
			continue
		}
		ifInfo := InfInfo{
			Mtu: item.Mtu,
		}
		if ifInfo.Addr = net.ParseIP(item.Ip); len(ifInfo.Addr) == 0 {
			_ctx.Warnf("Fail to parse IP: %s", item.Ip)
			continue
		}
		if ifInfo.Mtu == 0 {
			ifInfo.Mtu = 1400
		}
		_ctx.infs[item.Name] = ifInfo
	}

	_ctx.Infof("%v", _ctx.slices)
	_ctx.Infof("%v", _ctx.infs)
}

func PlmnId() *models.PlmnId {
	return &_ctx.plmnid
}

func UpfInfo() (info common.UpfInfo) {
	info = common.UpfInfo{
		Slices: make(map[string]models.Snssai),
		INFs:   make(map[string]common.UpfInfInfo),
	}
	for _, s := range _ctx.slices {
		info.Slices[s.String()] = s
	}

	for netname, item := range _ctx.infs {
		info.INFs[netname] = common.UpfInfInfo{
			Addr: item.Addr.String(),
			Mtu:  item.Mtu,
		}
	}

	return
}

func HasSlice(slice *models.Snssai) bool {
	for _, s := range _ctx.slices {
		if common.IsSliceEqual(slice, &s) {
			return true
		}
	}
	return false
}

func Slices() (slices []models.Snssai) {
	for _, s := range _ctx.slices {
		slices = append(slices, s)
	}
	return
}
