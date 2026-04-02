package forwarder

import (
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/hashicorp/go-version"
	"github.com/khirono/go-nl"

	"etrib5gc/logctx"
	"etrib5gc/nfs/smf/dataplane"
	"etrib5gc/nfs/upf/forwarder/buffnetlink"
	"etrib5gc/nfs/upf/forwarder/perio"
	"etrib5gc/nfs/upf/gtpv1"
	"etrib5gc/nfs/upf/report"
	"github.com/reogac/pfcp"
	"github.com/reogac/pfcp/ie"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"strings"

	"github.com/free5gc/go-gtp5gnl"
)

const (
	expectedMinGtp5gVersion string = "0.8.1"
	expectedMaxGtp5gVersion string = "0.9.0"
)

type Gtp5g struct {
	*logrus.Entry
	mux      *nl.Mux
	link     *Gtp5gLink
	conn     *nl.Conn
	psConn   *nl.Conn
	client   *gtp5gnl.Client
	psClient *gtp5gnl.Client
	bsnl     *buffnetlink.Server
	ps       *perio.Server
	gtpPort  uint16
}

func OpenGtp5g(wg *sync.WaitGroup, gtpPort uint16, addr string, mtu uint32, devName string) (g *Gtp5g, err error) {
	g = &Gtp5g{
		Entry: logctx.Entry(
			logrus.Fields{
				"mod": "forwarder",
			}),
	}
	g.gtpPort = gtpPort
	mux, err := nl.NewMux()
	if err != nil {
		err = utils.WrapError("New Mux", err)
		return
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err1 := mux.Serve()
		if err1 != nil {
			g.Warnf("Mux Serve err: %+v", err1)
		}
	}()
	g.mux = mux

	link, err := OpenGtp5gLink(g.mux, devName, addr, mtu)
	if err != nil {
		g.Close()
		err = utils.WrapError("Open link", err)
		return
	}
	g.link = link

	conn, err := nl.Open(syscall.NETLINK_GENERIC)
	if err != nil {
		g.Close()
		err = utils.WrapError("Open netlink", err)
		return
	}
	g.conn = conn

	c, err := gtp5gnl.NewClient(g.conn, g.mux)
	if err != nil {
		g.client = nil
		g.Close()
		err = utils.WrapError("New client", err)
		return
	}
	g.client = c

	psConn, err := nl.Open(syscall.NETLINK_GENERIC)
	if err != nil {
		g.Close()
		err = utils.WrapError("Open ps netlink", err)
		return
	}
	g.psConn = psConn

	psc, err := gtp5gnl.NewClient(psConn, mux)
	if err != nil {
		g.Close()
		err = utils.WrapError("New ps client", err)
		return
	}
	g.psClient = psc

	err = g.checkVersion()
	if err != nil {
		g.Close()
		err = utils.WrapError("version mismatch", err)
		return
	}

	bsnl, err := buffnetlink.OpenServer(wg, c.Client, mux)
	if err != nil {
		g.Close()
		err = utils.WrapError("open buff(netlink) server", err)
		return
	}
	g.bsnl = bsnl

	ps, err := perio.OpenServer(wg)
	if err != nil {
		g.Close()
		err = utils.WrapError("open perio server", err)
		return
	}
	g.ps = ps

	g.Infof("Forwarder started")
	return
}

func (g *Gtp5g) AddRoutes(cidrList []string) {
	for _, cidr := range cidrList {
		if _, dst, err := net.ParseCIDR(cidr); err != nil {
			g.Errorf("Parse CIDR error: %+v", err)
		} else if err = g.link.RouteAdd(dst); err != nil {
			g.Errorf("Fail to add route to %s: %+v", cidr, err)
		}
	}
}

func (g *Gtp5g) Close() {
	if g.conn != nil {
		g.conn.Close()
	}
	if g.psConn != nil {
		g.psConn.Close()
	}
	if g.link != nil {
		g.link.Close()
	}
	if g.mux != nil {
		g.mux.Close()
	}
	if g.bsnl != nil {
		g.bsnl.Close()
	}
	if g.ps != nil {
		g.ps.Close()
	}
}

func (g *Gtp5g) checkVersion() error {
	// get gtp5g version
	gtp5gVer, err := gtp5gnl.GetVersion(g.client)
	if err != nil {
		return err
	}

	// compare version
	expMinVer, err := version.NewVersion(expectedMinGtp5gVersion)
	if err != nil {
		return utils.WrapError("Parse expectedMinGtp5gVersion", err)
	}
	expMaxVer, err := version.NewVersion(expectedMaxGtp5gVersion)
	if err != nil {
		return utils.WrapError("Parse expectedMaxGtp5gVersion", err)
	}
	nowVer, err := version.NewVersion(gtp5gVer)
	if err != nil {
		return utils.WrapError(fmt.Sprintf("Parse gtp5g version(%s)", gtp5gVer), err)
	}
	if nowVer.LessThan(expMinVer) || nowVer.GreaterThanOrEqual(expMaxVer) {
		return fmt.Errorf(
			"gtp5g version(%v) should be %s <= verion < %s , please update it",
			nowVer, expectedMinGtp5gVersion, expectedMaxGtp5gVersion)
	}

	return nil
}

func (g *Gtp5g) Link() *Gtp5gLink {
	return g.link
}

func (g *Gtp5g) newFlowDesc(s string, swapSrcDst bool) (nl.AttrList, error) {
	var attrs nl.AttrList
	fd, err := ParseFlowDesc(s)
	if err != nil {
		return nil, utils.WrapError("Parse FlowDesc", err)
	}
	if swapSrcDst {
		fd.Src, fd.Dst = fd.Dst, fd.Src
	}
	switch fd.Action {
	case "permit":
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FLOW_DESCRIPTION_ACTION,
			Value: nl.AttrU8(gtp5gnl.SDF_FILTER_PERMIT),
		})
	default:
		return nil, fmt.Errorf("not support action %v", fd.Action)
	}
	switch fd.Dir {
	case "in":
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FLOW_DESCRIPTION_DIRECTION,
			Value: nl.AttrU8(gtp5gnl.SDF_FILTER_IN),
		})
	case "out":
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FLOW_DESCRIPTION_DIRECTION,
			Value: nl.AttrU8(gtp5gnl.SDF_FILTER_OUT),
		})
	default:
		return nil, fmt.Errorf("not support dir %v", fd.Dir)
	}
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_PROTOCOL,
		Value: nl.AttrU8(fd.Proto),
	})
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_SRC_IPV4,
		Value: nl.AttrBytes(fd.Src.IP),
	})
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_SRC_MASK,
		Value: nl.AttrBytes(fd.Src.Mask),
	})
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_DEST_IPV4,
		Value: nl.AttrBytes(fd.Dst.IP),
	})
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_DEST_MASK,
		Value: nl.AttrBytes(fd.Dst.Mask),
	})
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_SRC_PORT,
		Value: nl.AttrBytes(convertSlice(fd.SrcPorts)),
	})
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.FLOW_DESCRIPTION_DEST_PORT,
		Value: nl.AttrBytes(convertSlice(fd.DstPorts)),
	})
	return attrs, nil
}

func convertSlice(ports [][]uint16) []byte {
	b := make([]byte, len(ports)*4)
	off := 0
	for _, p := range ports {
		x := (*uint32)(unsafe.Pointer(&b[off]))
		switch len(p) {
		case 1:
			*x = uint32(p[0])<<16 | uint32(p[0])
		case 2:
			*x = uint32(p[0])<<16 | uint32(p[1])
		}
		off += 4
	}
	return b
}

func (g *Gtp5g) newSdfFilter(i *ie.SDFFilter, srcIf uint8) (nl.AttrList, error) {
	var attrs nl.AttrList

	if i.HasFD() {
		swapSrcDst := srcIf == dataplane.SrcInterfaceAccess
		fd, err := g.newFlowDesc(string(i.FlowDescription), swapSrcDst)
		if err != nil {
			return nil, utils.WrapError("Create FlowDesc", err)
		}
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.SDF_FILTER_FLOW_DESCRIPTION,
			Value: fd,
		})
	}
	if i.HasTTC() {
		// TODO:
		// v.ToSTrafficClass string
		x := uint16(29)
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.SDF_FILTER_TOS_TRAFFIC_CLASS,
			Value: nl.AttrU16(x),
		})
	}
	if i.HasSPI() {
		// TODO:
		// v.SecurityParameterIndex string
		x := uint32(30)
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.SDF_FILTER_SECURITY_PARAMETER_INDEX,
			Value: nl.AttrU32(x),
		})
	}
	if i.HasFL() {
		// TODO:
		// v.FlowLabel string
		x := uint32(31)
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.SDF_FILTER_FLOW_LABEL,
			Value: nl.AttrU32(x),
		})
	}
	if i.HasBID() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.SDF_FILTER_SDF_FILTER_ID,
			Value: nl.AttrU32(i.SDFFilterID),
		})
	}

	return attrs, nil
}

func (g *Gtp5g) newPdi(i *ie.PDI) (attrs nl.AttrList, err error) {
	if i.UEIPaddress == nil {
		err = fmt.Errorf("not found UEIPaddress")
		return
	} else {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.PDI_UE_ADDR_IPV4,
			Value: nl.AttrBytes(i.UEIPaddress.V4()),
		})
	}

	if i.LocalFTEID != nil {
		attrs = append(attrs, nl.Attr{
			Type: gtp5gnl.PDI_F_TEID,
			Value: nl.AttrList{
				{
					Type:  gtp5gnl.F_TEID_I_TEID,
					Value: nl.AttrU32(i.LocalFTEID.Teid()),
				},
				{
					Type:  gtp5gnl.F_TEID_GTPU_ADDR_IPV4,
					Value: nl.AttrBytes(i.LocalFTEID.V4()),
				},
			},
		})
	}

	for _, b := range i.SDFFilter {
		if v, err1 := g.newSdfFilter(&b, i.SourceInterface); err1 == nil {
			attrs = append(attrs, nl.Attr{
				Type:  gtp5gnl.PDI_SDF_FILTER,
				Value: v,
			})
		} else {
			g.Warning("Invalid SDFFilter: %+v", err1)
		}
	}

	return
}

func (g *Gtp5g) CreatePDR(lSeid uint64, req *ie.CreatePDR) error {
	sb := new(strings.Builder)
	pdrid := uint64(req.PDRID)
	fmt.Fprintf(sb, "Create PDR %d\n", pdrid)
	var attrs []nl.Attr

	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.PDR_PRECEDENCE,
		Value: nl.AttrU32(req.Precedence),
	})

	if req.PDI != nil {
		sb.WriteString("HAS PDI:\n")
		if fteid := req.PDI.LocalFTEID; fteid != nil {
			fmt.Fprintf(sb, "%s:%d\n", fteid.V4().String(), fteid.Teid())
		}

		if ueIp := req.PDI.UEIPaddress; ueIp != nil {
			sb.WriteString(ueIp.V4().String())
		}

		sb.WriteString("\n")
		if v, err := g.newPdi(req.PDI); err == nil && v != nil {
			attrs = append(attrs, nl.Attr{
				Type:  gtp5gnl.PDR_PDI,
				Value: v,
			})
		}
	}

	if req.OuterHeaderRemoval != nil {
		sb.WriteString("HAS OuterHeaderRemoval\n")
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.PDR_OUTER_HEADER_REMOVAL,
			Value: nl.AttrU8(req.OuterHeaderRemoval.Desc),
		})
	}

	if req.FARID != nil {
		fmt.Fprintf(sb, "FARID=%d\n", *req.FARID)
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.PDR_FAR_ID,
			Value: nl.AttrU32(*req.FARID),
		})
	}

	if req.QERID != nil {
		fmt.Fprintf(sb, "QERID=%d\n", *req.QERID)
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.PDR_QER_ID,
			Value: nl.AttrU32(*req.QERID),
		})
	}
	if req.URRID != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.PDR_URR_ID,
			Value: nl.AttrU32(req.URRID[0]),
		})
	}

	// TODO:
	// Not in 3GPP spec, just used for routing
	// var roleAddrIpv4 net.IP
	// roleAddrIpv4 = net.IPv4(34, 35, 36, 37)
	// pdr.RoleAddrIpv4 = &roleAddrIpv4

	// TODO:
	// Not in 3GPP spec, just used for buffering
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.PDR_UNIX_SOCKET_PATH,
		Value: nl.AttrString(gtp5gnl.PdrAddrForNetlink),
	})
	g.Debug(sb.String())
	oid := gtp5gnl.OID{lSeid, pdrid}
	return gtp5gnl.CreatePDROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) UpdatePDR(lSeid uint64, req *ie.UpdatePDR) error {
	pdrid := uint64(req.PDRID)
	var attrs []nl.Attr

	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.PDR_PRECEDENCE,
		Value: nl.AttrU32(*req.Precedence),
	})

	// todo add pdi

	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.PDR_PDI,
		Value: nl.AttrU32(*req.QERID),
	})

	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.PDR_URR_ID,
		Value: nl.AttrU32(*req.URRID),
	})

	oid := gtp5gnl.OID{lSeid, pdrid}
	return gtp5gnl.UpdatePDROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) RemovePDR(lSeid uint64, req *ie.RemovePDR) error {
	pdrid := uint64(req.PDRID)

	oid := gtp5gnl.OID{lSeid, uint64(pdrid)}
	return gtp5gnl.RemovePDROID(g.client, g.link.link, oid)
}

func (g *Gtp5g) newForwardingParameter(req *ie.ForwardingParameters) (nl.AttrList, error) {
	var attrs nl.AttrList

	// OuterHeaderCreation
	if ohc := req.OuterHeaderCreation; ohc != nil {
		var hc nl.AttrList
		hc = append(hc, nl.Attr{
			Type:  gtp5gnl.OUTER_HEADER_CREATION_DESCRIPTION,
			Value: nl.AttrU16(ohc.Desc()),
		})
		if ohc.IsGtp() { //GTP-U
			hc = append(hc, nl.Attr{
				Type:  gtp5gnl.OUTER_HEADER_CREATION_O_TEID,
				Value: nl.AttrU32(ohc.Teid()),
			})
			// GTPv1-U port
			hc = append(hc, nl.Attr{
				Type:  gtp5gnl.OUTER_HEADER_CREATION_PORT,
				Value: nl.AttrU16(g.gtpPort),
			})
		} else { //UDP
			hc = append(hc, nl.Attr{
				Type:  gtp5gnl.OUTER_HEADER_CREATION_PORT,
				Value: nl.AttrU16(ohc.Port()),
			})
		}
		hc = append(hc, nl.Attr{
			Type:  gtp5gnl.OUTER_HEADER_CREATION_PEER_ADDR_IPV4,
			Value: nl.AttrBytes(ohc.IP()),
		})
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FORWARDING_PARAMETER_OUTER_HEADER_CREATION,
			Value: hc,
		})
	}
	// ForwardingPolicy
	if req.ForwardingPolicy != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FORWARDING_PARAMETER_FORWARDING_POLICY,
			Value: nl.AttrString(req.ForwardingPolicy),
		})
	}

	return attrs, nil
}

func (g *Gtp5g) newUpdateForwardingParameter(req *ie.UpdateForwardingParameters) (nl.AttrList, error) {
	var attrs nl.AttrList
	// OuterHeaderCreation
	if ohc := req.OuterHeaderCreation; ohc != nil {
		var hc nl.AttrList
		hc = append(hc, nl.Attr{
			Type:  gtp5gnl.OUTER_HEADER_CREATION_DESCRIPTION,
			Value: nl.AttrU16(ohc.Desc()),
		})
		if ohc.IsGtp() {
			hc = append(hc, nl.Attr{
				Type:  gtp5gnl.OUTER_HEADER_CREATION_O_TEID,
				Value: nl.AttrU32(ohc.Teid()),
			})
			// GTPv1-U port
			hc = append(hc, nl.Attr{
				Type:  gtp5gnl.OUTER_HEADER_CREATION_PORT,
				Value: nl.AttrU16(g.gtpPort),
			})
		} else {
			hc = append(hc, nl.Attr{
				Type:  gtp5gnl.OUTER_HEADER_CREATION_PORT,
				Value: nl.AttrU16(ohc.Port()),
			})
		}
		hc = append(hc, nl.Attr{
			Type:  gtp5gnl.OUTER_HEADER_CREATION_PEER_ADDR_IPV4,
			Value: nl.AttrBytes(ohc.IP()),
		})
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FORWARDING_PARAMETER_OUTER_HEADER_CREATION,
			Value: hc,
		})
	}
	// ForwardingPolicy
	if req.ForwardingPolicy != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FORWARDING_PARAMETER_FORWARDING_POLICY,
			Value: nl.AttrString(req.ForwardingPolicy), ///////////////////
		})
	}
	// PFCPSMReqFlags
	if req.PFCPSMReqFlags != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FORWARDING_PARAMETER_PFCPSM_REQ_FLAGS,
			Value: nl.AttrU8(dataplane.PFCPSMReqFlags_SNDEM),
		})
	}

	return attrs, nil
}

func (g *Gtp5g) CreateFAR(lSeid uint64, req *ie.CreateFAR) error {
	farid := uint64(req.FARID)

	var attrs []nl.Attr

	// ApplyAction
	var act ie.ApplyAction
	if err := act.UnmarshalBinary(req.ApplyAction.Bytes()); err != nil {
		return err
	} else {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FAR_APPLY_ACTION,
			Value: nl.AttrU16(act.Flags),
		})
	}
	//ForwardingParameters
	if req.ForwardingParameters == nil {
		return fmt.Errorf("Not Found CreateFAR-ForwardingParameters")
	} else if v, err := g.newForwardingParameter(req.ForwardingParameters); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FAR_FORWARDING_PARAMETER,
			Value: v,
		})
	}
	// BARID
	if req.BARID != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FAR_BAR_ID,
			Value: nl.AttrU8(*req.BARID),
		})
	}
	oid := gtp5gnl.OID{lSeid, farid}
	return gtp5gnl.CreateFAROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) UpdateFAR(lSeid uint64, req *ie.UpdateFAR) error {
	farid := uint64(req.FARID)
	var attrs []nl.Attr

	// ApplyAction
	var act ie.ApplyAction
	if err := act.UnmarshalBinary(req.ApplyAction.Bytes()); err != nil { ////////////////////////
		return err
	} else {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FAR_APPLY_ACTION,
			Value: nl.AttrU16(act.Flags),
		})
	}
	//UpdateForwardingParameters
	if req.UpdateForwardingparameters == nil {
		return fmt.Errorf("Not Found CreateFAR-UpdateForwardingparameters")
	} else if v, err := g.newUpdateForwardingParameter(req.UpdateForwardingparameters); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FAR_FORWARDING_PARAMETER,
			Value: v,
		})
	}
	// BARID
	if req.BARID != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.FAR_BAR_ID,
			Value: nl.AttrU8(*req.BARID),
		})
	}

	oid := gtp5gnl.OID{lSeid, farid}
	return gtp5gnl.UpdateFAROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) RemoveFAR(lSeid uint64, req *ie.RemoveFAR) error {
	oid := gtp5gnl.OID{lSeid, uint64(req.FARID)}
	return gtp5gnl.RemoveFAROID(g.client, g.link.link, oid)
}

func (g *Gtp5g) CreateQER(lSeid uint64, req *ie.CreateQER) error {
	var qerid uint64
	var attrs []nl.Attr
	// qerid
	qerid = uint64(req.QERID)

	// QERCorrelationID
	if req.QERCorrelationID != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_CORR_ID,
			Value: nl.AttrU32(*req.QERCorrelationID), ////////////////
		})
	}
	// GateStatus
	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.QER_CORR_ID,
		Value: nl.AttrU8(ie.GateStatusOpen), ////////////////
	})
	// MBR
	if req.MaximumBitrate != nil {
		attrs = append(attrs, nl.Attr{
			Type: gtp5gnl.QER_MBR,
			Value: nl.AttrList{
				{
					Type:  gtp5gnl.QER_MBR_UL_HIGH32,
					Value: nl.AttrU32(req.MaximumBitrate.Ul >> 8),
				},
				{
					Type:  gtp5gnl.QER_MBR_UL_LOW8,
					Value: nl.AttrU8(req.MaximumBitrate.Ul),
				},
				{
					Type:  gtp5gnl.QER_MBR_DL_HIGH32,
					Value: nl.AttrU32(req.MaximumBitrate.Dl >> 8),
				},
				{
					Type:  gtp5gnl.QER_MBR_DL_LOW8,
					Value: nl.AttrU8(req.MaximumBitrate.Dl),
				},
			},
		})
	}
	// GBR
	if req.GuaranteedBitrate != nil {
		attrs = append(attrs, nl.Attr{
			Type: gtp5gnl.QER_GBR,
			Value: nl.AttrList{
				{
					Type:  gtp5gnl.QER_GBR_UL_HIGH32,
					Value: nl.AttrU32(req.GuaranteedBitrate.Ul >> 8),
				},
				{
					Type:  gtp5gnl.QER_GBR_UL_LOW8,
					Value: nl.AttrU8(req.GuaranteedBitrate.Ul),
				},
				{
					Type:  gtp5gnl.QER_GBR_DL_HIGH32,
					Value: nl.AttrU32(req.GuaranteedBitrate.Dl >> 8),
				},
				{
					Type:  gtp5gnl.QER_GBR_DL_LOW8,
					Value: nl.AttrU8(req.GuaranteedBitrate.Dl),
				},
			},
		})
	}
	// QFI
	if req.QoSflowidentifier != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_QFI,
			Value: nl.AttrU8(*req.QoSflowidentifier), ////////////////
		})
	}
	// RQI
	if req.ReflectiveQoS != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_QFI,
			Value: nl.AttrU8(*req.ReflectiveQoS), ///////////////
		})
	}
	// PagingPolicyIndicator
	if req.PagingPolicyIndicator != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_QFI,
			Value: nl.AttrU8(*req.PagingPolicyIndicator), ///////////////
		})
	}

	oid := gtp5gnl.OID{lSeid, qerid}
	return gtp5gnl.CreateQEROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) UpdateQER(lSeid uint64, req *ie.UpdateQER) error {
	var qerid uint64
	var attrs []nl.Attr
	// qerid
	if req.QERID != 0 {
		qerid = uint64(req.QERID)
	}
	// QERCorrelationID
	if req.QERCorrelationID != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_CORR_ID,
			Value: nl.AttrU32(*req.QERCorrelationID), ////////////////
		})
	}
	// GateStatus
	if req.GateStatus != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_CORR_ID,
			Value: nl.AttrU8(ie.GateStatusOpen), ////////////////
		})
	}
	// MBR
	if req.MaximumBitrate.Dl != 0 && req.MaximumBitrate.Ul != 0 {
		attrs = append(attrs, nl.Attr{
			Type: gtp5gnl.QER_MBR,
			Value: nl.AttrList{
				{
					Type:  gtp5gnl.QER_MBR_UL_HIGH32,
					Value: nl.AttrU32(req.MaximumBitrate.Ul >> 8),
				},
				{
					Type:  gtp5gnl.QER_MBR_UL_LOW8,
					Value: nl.AttrU8(req.MaximumBitrate.Ul),
				},
				{
					Type:  gtp5gnl.QER_MBR_DL_HIGH32,
					Value: nl.AttrU32(req.MaximumBitrate.Dl >> 8),
				},
				{
					Type:  gtp5gnl.QER_MBR_DL_LOW8,
					Value: nl.AttrU8(req.MaximumBitrate.Dl),
				},
			},
		})
	}
	// GBR
	if req.GuaranteedBitrate.Dl != 0 && req.GuaranteedBitrate.Ul != 0 {
		attrs = append(attrs, nl.Attr{
			Type: gtp5gnl.QER_GBR,
			Value: nl.AttrList{
				{
					Type:  gtp5gnl.QER_GBR_UL_HIGH32,
					Value: nl.AttrU32(req.GuaranteedBitrate.Ul >> 8),
				},
				{
					Type:  gtp5gnl.QER_GBR_UL_LOW8,
					Value: nl.AttrU8(req.GuaranteedBitrate.Ul),
				},
				{
					Type:  gtp5gnl.QER_GBR_DL_HIGH32,
					Value: nl.AttrU32(req.GuaranteedBitrate.Dl >> 8),
				},
				{
					Type:  gtp5gnl.QER_GBR_DL_LOW8,
					Value: nl.AttrU8(req.GuaranteedBitrate.Dl),
				},
			},
		})
	}
	// QFI
	if req.QoSflowidentifier != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_QFI,
			Value: nl.AttrU8(*req.QoSflowidentifier), ////////////////
		})
	}
	// RQI
	if req.ReflectiveQoS != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_QFI,
			Value: nl.AttrU8(*req.ReflectiveQoS), ///////////////
		})
	}
	// PagingPolicyIndicator
	if req.PagingPolicyIndicator != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.QER_QFI,
			Value: nl.AttrU8(*req.PagingPolicyIndicator), ///////////////
		})
	}

	oid := gtp5gnl.OID{lSeid, qerid}
	return gtp5gnl.UpdateQEROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) RemoveQER(lSeid uint64, req *ie.RemoveQER) error {
	oid := gtp5gnl.OID{lSeid, uint64(req.QERID)}
	return gtp5gnl.RemoveQEROID(g.client, g.link.link, oid)
}

func (g *Gtp5g) newVolumeThreshold(v *ie.VolumeThreshold) (nl.AttrList, error) {
	var attrs nl.AttrList
	if v == nil {
		return nil, fmt.Errorf("no volume threshold")
	}

	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.URR_VOLUME_THRESHOLD_FLAG,
		Value: nl.AttrU8(v.Flags),
	})
	if v.HasTOVOL() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_THRESHOLD_TOVOL,
			Value: nl.AttrU64(v.TotalVolume),
		})
	}
	if v.HasULVOL() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_THRESHOLD_UVOL,
			Value: nl.AttrU64(v.UplinkVolume),
		})
	}
	if v.HasDLVOL() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_THRESHOLD_DVOL,
			Value: nl.AttrU64(v.DownlinkVolume),
		})
	}

	return attrs, nil
}

func (g *Gtp5g) newVolumeQuota(v *ie.VolumeQuota) (nl.AttrList, error) {
	var attrs nl.AttrList

	if v == nil {
		return nil, fmt.Errorf("no volume quota")
	}

	attrs = append(attrs, nl.Attr{
		Type:  gtp5gnl.URR_VOLUME_QUOTA_FLAG,
		Value: nl.AttrU8(v.Flags),
	})
	if v.HasTOVOL() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_QUOTA_TOVOL,
			Value: nl.AttrU64(v.TotalVolume),
		})
	}
	if v.HasULVOL() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_QUOTA_UVOL,
			Value: nl.AttrU64(v.UplinkVolume),
		})
	}
	if v.HasDLVOL() {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_QUOTA_DVOL,
			Value: nl.AttrU64(v.DownlinkVolume),
		})
	}

	return attrs, nil
}

func (g *Gtp5g) CreateURR(lSeid uint64, req *ie.CreateURR) error {
	var rptTrig report.ReportingTrigger
	var measurePeriod time.Duration
	var attrs []nl.Attr

	// MeasurementMethod
	if req.MeasurementMethod.Value != 0 {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_MEASUREMENT_METHOD,
			Value: nl.AttrU8(req.MeasurementMethod.Value),
		})
	}
	// ReportingTriggers
	if err := rptTrig.Unmarshal(pfcp.MarshalUint32(req.ReportingTriggers)); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_REPORTING_TRIGGER,
			Value: nl.AttrU32(rptTrig.Flags),
		})
	}
	// MeasurementPeriod
	if req.MeasurementPeriod != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_MEASUREMENT_PERIOD,
			Value: nl.AttrU32(*req.MeasurementPeriod),
		})
	}
	// MeasurementInformation
	if req.MeasurementInformation != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_MEASUREMENT_PERIOD,
			Value: nl.AttrU32(req.MeasurementInformation.Flag),
		})
	}
	// VolumeThreshold
	if v, err := g.newVolumeThreshold(req.VolumeThreshold); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_THRESHOLD,
			Value: v,
		})
	}
	// VolumeQuota
	if v, err := g.newVolumeQuota(req.VolumeQuota); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_QUOTA,
			Value: v,
		})
	}

	if rptTrig.PERIO() {
		if measurePeriod <= 0 {
			return fmt.Errorf("invalid measurement period")
		}
		g.ps.AddPeriodReportTimer(lSeid, uint32(req.URRID), measurePeriod)
	}

	oid := gtp5gnl.OID{lSeid, uint64(req.URRID)}
	return gtp5gnl.CreateURROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) UpdateURR(lSeid uint64, req *ie.UpdateURR) ([]report.USAReport, error) {
	var urrid uint64
	var attrs []nl.Attr
	var usars []report.USAReport
	var rptTrig report.ReportingTrigger

	// MeasurementMethod
	if req.MeasurementMethod != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_MEASUREMENT_METHOD,
			Value: nl.AttrU8(req.MeasurementMethod.Value),
		})
	}
	// ReportingTriggers
	if err := rptTrig.Unmarshal(pfcp.MarshalUint32(*req.ReportingTriggers)); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_REPORTING_TRIGGER,
			Value: nl.AttrU32(rptTrig.Flags),
		})
	}
	// MeasurementPeriod
	if req.MeasurementPeriod != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_MEASUREMENT_PERIOD,
			Value: nl.AttrU32(*req.MeasurementPeriod),
		})
	}
	// MeasurementInformation
	if req.MeasurementInformation != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_MEASUREMENT_PERIOD,
			Value: nl.AttrU32(req.MeasurementInformation.Flag),
		})
	}
	// VolumeThreshold
	if v, err := g.newVolumeThreshold(req.VolumeThreshold); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_THRESHOLD,
			Value: v,
		})
	}
	// VolumeQuota
	if v, err := g.newVolumeQuota(req.VolumeQuota); err == nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.URR_VOLUME_QUOTA,
			Value: v,
		})
	}

	// TODO: should apply PERIO updateURR and receive final report from old URR

	oid := gtp5gnl.OID{lSeid, urrid}
	rs, err := gtp5gnl.UpdateURROID(g.client, g.link.link, oid, attrs)
	if err != nil {
		return nil, err
	}

	if rs == nil {
		return nil, nil
	}

	for _, r := range rs {
		usar := report.USAReport{
			URRID:       r.URRID,
			QueryUrrRef: r.QueryUrrRef,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
		}

		usar.USARTrigger.Flags = r.USARTrigger
		usar.VolumMeasure = report.VolumeMeasure{
			TotalVolume:    r.VolMeasurement.TotalVolume,
			UplinkVolume:   r.VolMeasurement.UplinkVolume,
			DownlinkVolume: r.VolMeasurement.DownlinkVolume,
			TotalPktNum:    r.VolMeasurement.TotalPktNum,
			UplinkPktNum:   r.VolMeasurement.UplinkPktNum,
			DownlinkPktNum: r.VolMeasurement.DownlinkPktNum,
		}

		usars = append(usars, usar)
	}

	return usars, err
}

func (g *Gtp5g) RemoveURR(lSeid uint64, req *ie.RemoveURR) ([]report.USAReport, error) {
	var usars []report.USAReport

	g.ps.DelPeriodReportTimer(lSeid, uint32(req.URRID))

	oid := gtp5gnl.OID{lSeid, uint64(req.URRID)}
	rs, err := gtp5gnl.RemoveURROID(g.client, g.link.link, oid)
	if err != nil {
		return nil, err
	}

	if rs == nil {
		return nil, nil
	}

	for _, r := range rs {
		usar := report.USAReport{
			URRID:       r.URRID,
			QueryUrrRef: r.QueryUrrRef,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
		}

		usar.USARTrigger.Flags = r.USARTrigger
		usar.VolumMeasure = report.VolumeMeasure{
			TotalVolume:    r.VolMeasurement.TotalVolume,
			UplinkVolume:   r.VolMeasurement.UplinkVolume,
			DownlinkVolume: r.VolMeasurement.DownlinkVolume,
			TotalPktNum:    r.VolMeasurement.TotalPktNum,
			UplinkPktNum:   r.VolMeasurement.UplinkPktNum,
			DownlinkPktNum: r.VolMeasurement.DownlinkPktNum,
		}

		usars = append(usars, usar)
	}

	return usars, err
}

func (g *Gtp5g) CreateBAR(lSeid uint64, req *ie.CreateBAR) error {
	var attrs []nl.Attr

	// DownlinkDataNotificationDelay
	if req.DownlinkDataNotificationDelay != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.BAR_DOWNLINK_DATA_NOTIFICATION_DELAY,
			Value: nl.AttrU8(*req.DownlinkDataNotificationDelay),
		})
	}
	// SuggestedBufferingPacketsCount
	if req.SuggestedBufferingPacketsCount != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.BAR_DOWNLINK_DATA_NOTIFICATION_DELAY,
			Value: nl.AttrU8(*req.SuggestedBufferingPacketsCount),
		})
	}

	oid := gtp5gnl.OID{lSeid, uint64(req.BARID)}
	return gtp5gnl.CreateBAROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) UpdateBAR(lSeid uint64, req *ie.UpdateBARSessionModificationRequest) error {
	var attrs []nl.Attr

	// DownlinkDataNotificationDelay
	if req.DownlinkDataNotificationDelay != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.BAR_DOWNLINK_DATA_NOTIFICATION_DELAY,
			Value: nl.AttrU8(*req.DownlinkDataNotificationDelay),
		})
	}
	// SuggestedBufferingPacketsCount
	if req.SuggestedBufferingPacketsCount != nil {
		attrs = append(attrs, nl.Attr{
			Type:  gtp5gnl.BAR_DOWNLINK_DATA_NOTIFICATION_DELAY,
			Value: nl.AttrU8(*req.SuggestedBufferingPacketsCount),
		})
	}

	oid := gtp5gnl.OID{lSeid, uint64(req.BARID)}
	return gtp5gnl.UpdateBAROID(g.client, g.link.link, oid, attrs)
}

func (g *Gtp5g) RemoveBAR(lSeid uint64, req *ie.RemoveBAR) error {
	oid := gtp5gnl.OID{lSeid, uint64(req.BARID)}
	return gtp5gnl.RemoveBAROID(g.client, g.link.link, oid)
}

func (g *Gtp5g) QueryURR(lSeid uint64, urrid uint32) ([]report.USAReport, error) {
	return g.queryURR(lSeid, urrid, false)
}

func (g *Gtp5g) psQueryURR(lSeidUrridsMap map[uint64][]uint32) (map[uint64][]report.USAReport, error) {
	return g.queryMultiURR(lSeidUrridsMap, true)
}

func (g *Gtp5g) queryURR(lSeid uint64, urrid uint32, ps bool) ([]report.USAReport, error) {
	var usars []report.USAReport

	oid := gtp5gnl.OID{lSeid, uint64(urrid)}
	c := g.client
	if ps {
		c = g.psClient
	}
	rs, err := gtp5gnl.GetReportOID(c, g.link.link, oid)
	if err != nil {
		return nil, fmt.Errorf("queryURR[%#x:%#x]: %+v", lSeid, urrid, err)
	}

	if rs == nil {
		return nil, nil
	}

	for _, r := range rs {
		usar := report.USAReport{
			URRID:       r.URRID,
			QueryUrrRef: r.QueryUrrRef,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
		}

		usar.VolumMeasure = report.VolumeMeasure{
			TotalVolume:    r.VolMeasurement.TotalVolume,
			UplinkVolume:   r.VolMeasurement.UplinkVolume,
			DownlinkVolume: r.VolMeasurement.DownlinkVolume,
			TotalPktNum:    r.VolMeasurement.TotalPktNum,
			UplinkPktNum:   r.VolMeasurement.UplinkPktNum,
			DownlinkPktNum: r.VolMeasurement.DownlinkPktNum,
		}

		usars = append(usars, usar)
	}

	g.Tracef("queryURR: %+v", usars)

	return usars, nil
}

func (g *Gtp5g) QueryMultiURR(lSeidUrridsMap map[uint64][]uint32) (map[uint64][]report.USAReport, error) {
	return g.queryMultiURR(lSeidUrridsMap, false)
}

func (g *Gtp5g) queryMultiURR(lSeidUrridsMap map[uint64][]uint32, ps bool) (map[uint64][]report.USAReport, error) {
	var oids []gtp5gnl.OID
	var reports []gtp5gnl.USAReport

	c := g.client
	if ps {
		c = g.psClient
	}

	// Note: the max size of netlink msg is 16k,
	//       the number of reports from gtp5g is limited
	//       depending on the size of report
	queryNum := 0
	queryNumOnce := gtp5gnl.MaxNetlinkUsageReportNum()
	for seid, urrIds := range lSeidUrridsMap {
		for _, urrId := range urrIds {
			oids = append(oids, gtp5gnl.OID{seid, uint64(urrId)})
			queryNum++

			if queryNum >= queryNumOnce {
				rs, err := gtp5gnl.GetMultiReportsOID(c, g.link.link, oids)
				if err != nil {
					return nil, fmt.Errorf("queryMultiURR[%+v]: %+v", lSeidUrridsMap, err)
				}

				g.Tracef("Reports number in one netlink request: %+v", len(rs))
				reports = append(reports, rs...)
				oids = oids[:0]
				queryNum = 0
			}
		}
	}

	if len(oids) > 0 {
		rs, err := gtp5gnl.GetMultiReportsOID(c, g.link.link, oids)
		if err != nil {
			return nil, fmt.Errorf("queryMultiURR[%+v]: %+v", lSeidUrridsMap, err)
		}

		g.Tracef("Reports number in one netlink request: %+v", len(rs))
		reports = append(reports, rs...)
	}

	if reports == nil {
		return nil, nil
	}

	usars := make(map[uint64][]report.USAReport)
	for _, r := range reports {
		usar := report.USAReport{
			URRID:       r.URRID,
			QueryUrrRef: r.QueryUrrRef,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
		}

		usar.VolumMeasure = report.VolumeMeasure{
			TotalVolume:    r.VolMeasurement.TotalVolume,
			UplinkVolume:   r.VolMeasurement.UplinkVolume,
			DownlinkVolume: r.VolMeasurement.DownlinkVolume,
			TotalPktNum:    r.VolMeasurement.TotalPktNum,
			UplinkPktNum:   r.VolMeasurement.UplinkPktNum,
			DownlinkPktNum: r.VolMeasurement.DownlinkPktNum,
		}
		usars[r.SEID] = append(usars[r.SEID], usar)
	}

	g.Tracef("queryMultiURR: %+v", usars)

	return usars, nil
}

func (g *Gtp5g) HandleReport(handler report.Handler) {
	g.bsnl.Handle(handler)
	g.ps.Handle(handler, g.psQueryURR)
}

func (g *Gtp5g) applyAction(lSeid uint64, farid int, action ie.ApplyAction) {
	oid := gtp5gnl.OID{lSeid, uint64(farid)}
	far, err := gtp5gnl.GetFAROID(g.client, g.link.link, oid)
	if err != nil {
		g.Errorf("applyAction err: %+v", err)
		return
	}
	if far.Action&ie.APPLY_ACT_BUFF == 0 {
		return
	}
	switch {
	case action.DROP():
		// BUFF -> DROP
		for _, pdrid := range far.PDRIDs {
			for {
				_, ok := g.bsnl.Pop(lSeid, pdrid)
				if !ok {
					break
				}
			}
		}
	case action.FORW():
		// BUFF -> FORW
		for _, pdrid := range far.PDRIDs {
			oid := gtp5gnl.OID{lSeid, uint64(pdrid)}
			pdr, err := gtp5gnl.GetPDROID(g.client, g.link.link, oid)
			if err != nil {
				g.Warnf("applyAction GetPDROID err: %+v", err)
				continue
			}
			var qer *gtp5gnl.QER
			for _, qerId := range pdr.QERID {
				oid := gtp5gnl.OID{lSeid, uint64(qerId)}
				q, err := gtp5gnl.GetQEROID(g.client, g.link.link, oid)
				if err != nil {
					g.Warnf("applyAction GetQEROID err: %+v", err)
					continue
				}
				if q.QFI != 0 {
					qer = q
					break
				}
			}
			for {
				pkt, ok := g.bsnl.Pop(lSeid, pdrid)
				if !ok {
					break
				}
				err := g.WritePacket(far, qer, pkt)
				if err != nil {
					g.Warnf("applyAction WritePacket err: %+v", err)
					continue
				}
			}
		}
	}
}

func (g *Gtp5g) WritePacket(far *gtp5gnl.FAR, qer *gtp5gnl.QER, pkt []byte) error {
	if far.Param == nil || far.Param.Creation == nil {
		return fmt.Errorf("far param not found")
	}
	hc := far.Param.Creation
	addr := &net.UDPAddr{
		IP:   hc.PeerAddr,
		Port: int(hc.Port),
	}
	msg := gtpv1.Message{
		Flags:   0x34,
		Type:    gtpv1.MsgTypeTPDU,
		TEID:    hc.TEID,
		Payload: pkt,
	}
	if qer != nil {
		msg.Exts = []gtpv1.Encoder{
			gtpv1.PDUSessionContainer{
				PDUType:   0,
				QoSFlowID: qer.QFI,
			},
		}
	}
	n := msg.Len()
	b := make([]byte, n)
	_, err := msg.Encode(b)
	if err != nil {
		return err
	}
	_, err = g.link.WriteTo(b, addr)
	return err
}
