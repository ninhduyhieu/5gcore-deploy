package forwarder

import (
	"etrib5gc/logctx"
	"net"
	"os"
	"syscall"

	"github.com/khirono/go-nl"
	"github.com/khirono/go-rtnllink"
	"github.com/khirono/go-rtnlroute"
	"github.com/reogac/utils"

	"github.com/free5gc/go-gtp5gnl"
	"github.com/sirupsen/logrus"
)

const (
	UPFGTP_DEV_NAME string = "upfgtp"
)

type Gtp5gLink struct {
	*logrus.Entry
	mux     *nl.Mux
	rtconn  *nl.Conn
	client  *nl.Client
	link    *gtp5gnl.Link
	conn    *net.UDPConn
	f       *os.File
	devName string
}

func OpenGtp5gLink(mux *nl.Mux, devName string, addr string, mtu uint32) (g *Gtp5gLink, err error) {

	g = &Gtp5gLink{
		Entry: logctx.Entry(
			logrus.Fields{
				"mod": "forwarder",
			}),
		mux: mux,
	}

	g.devName = UPFGTP_DEV_NAME
	if len(devName) > 0 {
		g.devName = devName
	}

	if g.rtconn, err = nl.Open(syscall.NETLINK_ROUTE); err != nil {
		err = utils.WrapError("open netlink", err)
		return
	}
	g.client = nl.NewClient(g.rtconn, g.mux)

	var laddr *net.UDPAddr
	if laddr, err = net.ResolveUDPAddr("udp4", addr); err != nil {
		g.Close()
		err = utils.WrapError("resolve addr", err)
		return
	}
	if g.conn, err = net.ListenUDP("udp4", laddr); err != nil {
		g.conn = nil
		g.Close()
		err = utils.WrapError("gtp udp server listening", err)
		return
	}
	g.Infof("Listen to %s", addr)
	if g.f, err = g.conn.File(); err != nil {
		g.f = nil
		g.Close()
		err = utils.WrapError("get file failed", err)
		return
	}

	linkinfo := &nl.Attr{
		Type: syscall.IFLA_LINKINFO,
		Value: nl.AttrList{
			{
				Type:  rtnllink.IFLA_INFO_KIND,
				Value: nl.AttrString("gtp5g"),
			},
			{
				Type: rtnllink.IFLA_INFO_DATA,
				Value: nl.AttrList{
					{
						Type:  gtp5gnl.IFLA_FD1,
						Value: nl.AttrU32(g.f.Fd()),
					},
					{
						Type:  gtp5gnl.IFLA_HASHSIZE,
						Value: nl.AttrU32(131072),
					},
				},
			},
		},
	}
	attrs := []*nl.Attr{linkinfo}

	if mtu != 0 {
		attrs = append(attrs, &nl.Attr{
			Type:  syscall.IFLA_MTU,
			Value: nl.AttrU32(mtu),
		})
	}

	if err = rtnllink.Create(g.client, g.devName, attrs...); err != nil {
		g.Close()
		err = utils.WrapError("create rtnllink", err)
		return
	}

	g.Infof("Create link device %s", g.devName)

	if err = rtnllink.Up(g.client, g.devName); err != nil {
		g.Close()
		err = utils.WrapError("up rtnllink", err)
		return
	}
	if g.link, err = gtp5gnl.GetLink(g.devName); err != nil {
		g.Close()
		err = utils.WrapError("get link", err)
	}

	return
}

func (g *Gtp5gLink) Close() {
	if g.f != nil {
		err := g.f.Close()
		if err != nil {
			g.Warnf("file close err: %+v", err)
		}
	}
	if g.conn != nil {
		err := g.conn.Close()
		if err != nil {
			g.Warnf("conn close err: %+v", err)
		}
	}
	if g.link != nil {
		err := rtnllink.Remove(g.client, g.devName)
		if err != nil {
			g.Warnf("rtnllink remove err: %+v", err)
		}
	}
	if g.rtconn != nil {
		g.rtconn.Close()
	}
}

func (g *Gtp5gLink) RouteAdd(dst *net.IPNet) error {
	r := &rtnlroute.Request{
		Header: rtnlroute.Header{
			Table:    syscall.RT_TABLE_MAIN,
			Scope:    syscall.RT_SCOPE_UNIVERSE,
			Protocol: syscall.RTPROT_STATIC,
			Type:     syscall.RTN_UNICAST,
		},
	}
	err := r.AddDst(dst)
	if err != nil {
		return err
	}
	err = r.AddIfName(g.link.Name)
	if err != nil {
		return err
	}
	return rtnlroute.Create(g.client, r)
}

func (g *Gtp5gLink) WriteTo(b []byte, addr net.Addr) (int, error) {
	return g.conn.WriteTo(b, addr)
}
