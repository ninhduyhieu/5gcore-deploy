package context

import (
	"etrib5gc/common"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"net"
)

type ServerAddr struct {
	v4, v6 net.IP
}

type DataNetwork struct {
	UeIpAllocator
	dns   *ServerAddr
	pcscf *ServerAddr
}

func (ctx *SmfContext) addDataNetwork(cfg models.DataNetworkConfiguration) {
	if len(cfg.Name) == 0 {
		return
	}
	if _, ok := ctx.dataNetworks[cfg.Name]; ok {
		return
	}

	var ipAllocator UeIpAllocator
	if len(cfg.DhcpServer) > 0 {

		ipAllocator = &dhcpClient{
			dnn:        cfg.Name,
			dhcpServer: cfg.DhcpServer,
		}

	} else {
		if _, cidr, err := net.ParseCIDR(cfg.Cidr); err != nil {
			ctx.Errorf("Fail to parse CIDR: %s", cfg.Cidr)
		} else if cfg.IpRange == nil || *cfg.IpRange <= 0 {
			ctx.Errorf("Invalid Ip range")
		} else {
			pools := &ueIpAllocator{
				dnn:     cfg.Name,
				cidr:    *cidr,
				ipRange: *cfg.IpRange,
				pools:   make(map[int]UeIpPool),
			}

			for _, id := range cfg.PoolIndexList {
				pools.addPool(int(id))
			}
			ipAllocator = pools
		}
	}
	if ipAllocator != nil {
		ctx.dataNetworks[cfg.Name] = &DataNetwork{
			UeIpAllocator: ipAllocator,
			dns:           toServerAddr(cfg.Dns),
			pcscf:         toServerAddr(cfg.Pcscf),
		}
	}
}

func toServerAddr(info *models.IpAddr) *ServerAddr {
	if info != nil {
		var addr ServerAddr
		addr.v4 = net.ParseIP(info.Ipv4Addr)
		addr.v6 = net.ParseIP(info.Ipv6Addr)
		if len(addr.v4) > 0 || len(addr.v6) > 0 {
			return &addr
		}
	}
	return nil
}

func GetSessionType(reqType *uint8, dnn string, smData *models.SessionManagementSubscriptionData) (acceptCause, rejectCause *uint8, sessionType uint8) {
	sessionType = nas.PduSessionTypeIpv4
	acceptCause = common.ValuePointer[uint8](nas.Cause5GSMPDUSessionTypeIPv4OnlyAllowed)
	//TODO: get from DnnConfig + data network configuration
	return
}

/*

func (smCtx *SmContext) getDnnConfig() *models.DnnConfiguration {
	if smCtx.smData != nil {
		if cfg, ok := smCtx.smData.DnnConfigurations[smCtx.dnn]; ok {
			return &cfg
		}
	}
	return nil
}

func (smCtx *SmContext) checkPduSessionType(reqType uint8) (acceptCause, rejectCause *uint8) {

	//just for testing
	smCtx.sessionType = nas.PduSessionTypeIpv4
	return

	//TODO: configure SMF with DNN configuration
	var ok bool
	var dnnConfig models.DnnConfiguration
	if dnnConfig, ok = smCtx.smData.DnnConfigurations[smCtx.dnn]; !ok {
		rejectCause = new(uint8)
		*rejectCause = nas.Cause5GSMProtocolErrorUnspecified
		return
	}

	v4 := false
	v6 := false
	eth := false
	for _, allowedtype := range dnnConfig.PduSessionTypes.AllowedSessionTypes {
		switch models.PduSessionType(allowedtype) {
		case models.PDUSESSIONTYPE_IPV4:
			v4 = true
		case models.PDUSESSIONTYPE_IPV6:
			v6 = true
		case models.PDUSESSIONTYPE_IPV4V6:
			v4 = true
			v6 = true
		case models.PDUSESSIONTYPE_ETHERNET:
			eth = true
		}
	}
	if v4 {
		v4 = context.IsIpv4SessionSupported()
	}
	if v6 {
		v6 = context.IsIpv4SessionSupported()
	}
	if eth {
		eth = context.IsEthernetSessionSupported()
	}

	switch reqType {
	case nas.PduSessionTypeIpv4:
		if v4 {
			smCtx.sessionType = nas.PduSessionTypeIpv4
		} else {
			rejectCause = new(uint8)
			*rejectCause = nas.Cause5GSMProtocolErrorUnspecified
		}
	case nas.PduSessionTypeIpv6:
		if v6 {
			smCtx.sessionType = nas.PduSessionTypeIpv6
		} else {
			rejectCause = new(uint8)
			*rejectCause = nas.Cause5GSMProtocolErrorUnspecified
		}
	case nas.PduSessionTypeIpv4Ipv6:
		if v4 && v6 {
			smCtx.sessionType = nas.PduSessionTypeIpv4Ipv6
		} else if v4 {
			smCtx.sessionType = nas.PduSessionTypeIpv4
			acceptCause = new(uint8)
			*acceptCause = nas.Cause5GSMPDUSessionTypeIPv4OnlyAllowed
		} else if v6 {
			smCtx.sessionType = nas.PduSessionTypeIpv6
			acceptCause = new(uint8)
			*acceptCause = nas.Cause5GSMPDUSessionTypeIPv4OnlyAllowed
		} else {
			rejectCause = new(uint8)
			*rejectCause = nas.Cause5GSMProtocolErrorUnspecified
		}
	case nas.PduSessionTypeEthernet:
		if eth {
			smCtx.sessionType = nas.PduSessionTypeEthernet
		} else {
			rejectCause = new(uint8)
			*rejectCause = nas.Cause5GSMProtocolErrorUnspecified
		}
	default:
		rejectCause = new(uint8)
		*rejectCause = nas.Cause5GSMUnknownPDUSessionType
	}
	return
}
*/
