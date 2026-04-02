package context

import (
	"etrib5gc/common"
	"etrib5gc/nfs/nsm/config"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils/ipalloc"
	"net"
)

type Slice struct {
	id           models.Snssai
	dataNetworks []string
}

type ServerAddr struct {
	v4 net.IP
	v6 net.IP
}

type PoolOwner struct {
	uuid          string
	poolIndexList []int16
}

type DataNetwork struct {
	name       string
	slices     []models.Snssai
	cidr       net.IPNet
	ipRange    int64    //ue ip address range for a single pool
	poolStatus []string //all ue Ip address pools
	dhcpServer string   //external dhcp server for allocating Ue Ip
	dns, pcscf *ServerAddr
}

func (dn *DataNetwork) hasSlice(slice *models.Snssai) bool {
	for _, s := range dn.slices {
		if common.IsSliceEqual(&s, slice) {
			return true
		}
	}
	return false
}

func (dn *DataNetwork) getNextIdlePool() int {
	for i, owner := range dn.poolStatus {
		if len(owner) == 0 {
			return i
		}
	}
	return -1
}

func (dn *DataNetwork) setPoolOwner(id int, uuid string) {
	if id >= 0 && id < len(dn.poolStatus) {
		dn.poolStatus[id] = uuid
	}
}

func (dn *DataNetwork) removePoolOwner(uuid string) {
	for i, owner := range dn.poolStatus {
		if owner == uuid {
			dn.poolStatus[i] = ""
		}
	}
}

func (dn *DataNetwork) buildConfigurationModel(uuid string) (cfg models.DataNetworkConfiguration) {

	cfg.Name = dn.name
	cfg.Cidr = dn.cidr.String()
	if dn.dns != nil {
		cfg.Dns = &models.IpAddr{
			Ipv4Addr: dn.dns.v4.String(),
			Ipv6Addr: dn.dns.v6.String(),
		}
	}

	if dn.pcscf != nil {
		cfg.Pcscf = &models.IpAddr{
			Ipv4Addr: dn.pcscf.v4.String(),
			Ipv6Addr: dn.pcscf.v6.String(),
		}
	}

	if len(dn.poolStatus) > 0 {
		if id := dn.getNextIdlePool(); id != -1 { //there is an idle IP pool
			cfg.PoolIndexList = []int16{int16(id)}
			dn.setPoolOwner(id, uuid) //mark the pool as being occupied
		}

		cfg.IpRange = common.ValuePointer[int64](dn.ipRange)
	} else {
		cfg.DhcpServer = dn.dhcpServer
	}
	return
}

func (ctx *NsmContext) createDataNetwork(cfg config.DataNetworkConfig) (dn *DataNetwork) {
	if len(cfg.Name) == 0 {
		log.Warnf("Build data network info: Empty data network name")
		return nil
	}

	if len(cfg.Cidr) == 0 {
		log.Warnf("Build data network info: Empty data network CIDR")
		return nil
	}

	dn = &DataNetwork{
		name:       cfg.Name,
		dhcpServer: cfg.DhcpServer,
	}

	if cfg.Dns != nil {
		v4 := net.ParseIP(cfg.Dns.V4)
		v6 := net.ParseIP(cfg.Dns.V6)
		if len(v4) > 0 || len(v6) > 0 {
			dn.dns = &ServerAddr{
				v4: v4,
				v6: v6,
			}
		}
	}

	if cfg.Pcscf != nil {
		v4 := net.ParseIP(cfg.Pcscf.V4)
		v6 := net.ParseIP(cfg.Pcscf.V6)
		if len(v4) > 0 || len(v6) > 0 {
			dn.pcscf = &ServerAddr{
				v4: v4,
				v6: v6,
			}
		}
	}

	if _, cidr, err := net.ParseCIDR(cfg.Cidr); err != nil {
		log.Warnf("Build data network info: Fail to parse CIDR: %s", cfg.Cidr)
		return nil
	} else {
		dn.cidr = *cidr
	}

	if len(cfg.DhcpServer) == 0 { //Data network with Ue IP Pools
		if numPools := int64(cfg.NumIpPools); numPools <= 0 {
			log.Warnf("Build data network info: Number of IP pools not set")
			return nil
		} else {
			maxIp := ipalloc.MaxIp(dn.cidr)
			dn.ipRange = maxIp / numPools
			if maxIp%numPools == 0 {
				dn.poolStatus = make([]string, cfg.NumIpPools)
			} else {
				dn.poolStatus = make([]string, cfg.NumIpPools+1)
			}
		}
	}

	return dn
}
