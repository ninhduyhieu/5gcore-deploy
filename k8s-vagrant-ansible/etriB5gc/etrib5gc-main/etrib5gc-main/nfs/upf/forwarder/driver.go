package forwarder

import (
	"fmt"
	//	"net"
	"sync"

	"etrib5gc/nfs/upf/config"
	"etrib5gc/nfs/upf/report"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/utils"
)

const (
	GTP_DEFAULT_PORT uint16 = 2152
	DEFAULT_MTU      uint32 = 1400
)

type Driver interface {
	Close()

	CreatePDR(uint64, *ie.CreatePDR) error
	UpdatePDR(uint64, *ie.UpdatePDR) error
	RemovePDR(uint64, *ie.RemovePDR) error

	CreateFAR(uint64, *ie.CreateFAR) error
	UpdateFAR(uint64, *ie.UpdateFAR) error
	RemoveFAR(uint64, *ie.RemoveFAR) error

	CreateQER(uint64, *ie.CreateQER) error
	UpdateQER(uint64, *ie.UpdateQER) error
	RemoveQER(uint64, *ie.RemoveQER) error

	CreateURR(uint64, *ie.CreateURR) error
	UpdateURR(uint64, *ie.UpdateURR) ([]report.USAReport, error)
	RemoveURR(uint64, *ie.RemoveURR) ([]report.USAReport, error)
	QueryURR(uint64, uint32) ([]report.USAReport, error)

	CreateBAR(uint64, *ie.CreateBAR) error
	UpdateBAR(uint64, *ie.UpdateBARSessionModificationRequest) error
	RemoveBAR(uint64, *ie.RemoveBAR) error

	HandleReport(report.Handler)
	AddRoutes([]string)
}

func NewDriver(wg *sync.WaitGroup, cfg *config.Config) (d Driver, err error) {
	var gtpPort uint16 = GTP_DEFAULT_PORT
	if cfg.GtpPort != nil {
		gtpPort = *cfg.GtpPort
	}
	/*
		var gtpuAddr string
		var mtu uint32
		for _, info := range cfg.IfList {
			mtu = info.Mtu
			gtpuAddr = fmt.Sprintf("%s:%d", info.Ip, gtpPort)
			break
		}
		if gtpuAddr == "" {
			err = fmt.Errorf("not found GTP address")
			return
		}
	*/
	mtu := DEFAULT_MTU
	gtpuAddr := fmt.Sprintf("127.0.0.12:%d", gtpPort)

	var driver *Gtp5g
	if driver, err = OpenGtp5g(wg, gtpPort, gtpuAddr, mtu, cfg.DevName); err != nil {
		err = utils.WrapError("open Gtp5g", err)
		return
	}

	d = driver
	return
}
