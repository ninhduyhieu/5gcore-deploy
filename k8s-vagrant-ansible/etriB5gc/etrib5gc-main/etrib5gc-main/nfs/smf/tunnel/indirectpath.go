package tunnel

import (
	"fmt"
	"github.com/lvdund/ngap/ies"
	"net"
)

type IndirectPath struct {
}

func (p *IndirectPath) Release() {
	//TODO:
}

func (p *IndirectPath) UpdateRanInfo(ip net.IP, teid uint32) (err error) {
	//TODO

	err = fmt.Errorf("Update RanInfo for IndirectPath not implemented")
	return
}
func (p *IndirectPath) GetDlGtpTunnel() *ies.GTPTunnel {
	return &ies.GTPTunnel{
		//TODO: get IP, TEID from the first UPF
	}
}
func CreateIndirectPath(tunnel *Tunnel, targetRanTransportNets []string) (p *IndirectPath, err error) {
	//TODO
	err = fmt.Errorf("IndirectPath creation not implemented")
	return
}
