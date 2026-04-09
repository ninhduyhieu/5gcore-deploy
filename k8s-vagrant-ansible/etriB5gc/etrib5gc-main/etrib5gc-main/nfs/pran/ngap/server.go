package ngap

import (
	"etrib5gc/logctx"
	"etrib5gc/nfs/pran/ran"
	"etrib5gc/sctp"
	"github.com/reogac/utils"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
	"syscall"
)

const PPID uint32 = 0x3c000000

type Server struct {
	*logrus.Entry
	connections sync.Map
	listener    *sctp.SCTPListener
	wg          sync.WaitGroup
	ipList      []string
	port        int
	done        chan bool
}

const readBufSize uint32 = 8192

// set default read timeout to 2 seconds
var readTimeout syscall.Timeval = syscall.Timeval{Sec: 2, Usec: 0}

var sctpConfig sctp.SocketConfig = sctp.SocketConfig{
	InitMsg:   sctp.InitMsg{NumOstreams: 3, MaxInstreams: 5, MaxAttempts: 2, MaxInitTimeout: 2},
	RtoInfo:   &sctp.RtoInfo{SrtoAssocID: 0, SrtoInitial: 500, SrtoMax: 1500, StroMin: 100},
	AssocInfo: &sctp.AssocInfo{AsocMaxRxt: 4},
}

func NewServer(ipList []string, port int) *Server {
	return &Server{
		Entry: logctx.Entry(logrus.Fields{
			"mod": "ngap",
		}),
		done:   make(chan bool),
		ipList: ipList,
		port:   port,
	}
}

func (s *Server) Run() (err error) {
	if s.listener != nil {
		return nil
	}

	ips := []net.IPAddr{}
	var netAddr *net.IPAddr
	for _, addr := range s.ipList {
		if netAddr, err = net.ResolveIPAddr("ip", addr); err != nil {
			return utils.WrapError("Resolving binding address", err)
		} else {
			s.Debugf("Resolved address '%s' to %s\n", addr, netAddr)
			ips = append(ips, *netAddr)
		}
	}

	addr := &sctp.SCTPAddr{
		IPAddrs: ips,
		Port:    s.port,
	}

	if s.listener, err = sctpConfig.Listen("sctp", addr); err != nil {
		return utils.WrapError("Server start listening", err)
	}
	s.Infof("NGAP server listening on %s", s.listener.Addr())
	go s.loop()
	return
}

func (s *Server) loop() {
	s.wg.Add(1)
	defer s.wg.Done()
	s.Infof("Waiting for Ngap connections")
	for {
		select {
		case <-s.done:
			return
		default:
			newConn, err := s.listener.AcceptSCTP()
			if err != nil {
				s.Errorf("Receive an error while waiting for new connections: %+v", err)
				switch err {
				case syscall.EINTR, syscall.EAGAIN:
					s.Debugf("EINTR or EAGAIN error type")
				default:
				}
				continue
			} else if newConn != nil {
				s.Debugf("New connection from %s:%s", newConn.RemoteAddr().Network(), newConn.RemoteAddr().String())
				var info *sctp.SndRcvInfo
				if infoTmp, err := newConn.GetDefaultSentParam(); err != nil {
					s.Errorf("Get default sent param error: %+v, accept failed", err)
					if err = newConn.Close(); err != nil {
						s.Errorf("Close error: %+v", err)
					}
					continue
				} else {
					info = infoTmp
					s.Tracef("Get default sent param[value: %+v]", info)
				}

				info.PPID = PPID
				if err := newConn.SetDefaultSentParam(info); err != nil {
					s.Errorf("Set default sent param error: %+v, accept failed", err)
					if err = newConn.Close(); err != nil {
						s.Errorf("Close error: %+v", err)
					}
					continue
				} else {
					s.Tracef("Set default sent param[value: %+v]", info)
				}

				events := sctp.SCTP_EVENT_DATA_IO | sctp.SCTP_EVENT_SHUTDOWN | sctp.SCTP_EVENT_ASSOCIATION
				if err := newConn.SubscribeEvents(events); err != nil {
					s.Errorf("Failed to subscribe events: %+v", err)
					if err = newConn.Close(); err != nil {
						s.Errorf("Close error: %+v", err)
					}
					continue
				} else {
					s.Tracef("Subscribe SCTP event[DATA_IO, SHUTDOWN_EVENT, ASSOCIATION_CHANGE]")
				}

				if err := newConn.SetReadBuffer(int(readBufSize)); err != nil {
					s.Errorf("Set read buffer error: %+v, accept failed", err)
					if err = newConn.Close(); err != nil {
						s.Errorf("Close error: %+v", err)
					}
					continue
				} else {
					s.Tracef("Set read buffer to %d bytes", readBufSize)
				}

				if err := newConn.SetReadTimeout(readTimeout); err != nil {
					s.Errorf("Set read timeout error: %+v, accept failed", err)
					if err = newConn.Close(); err != nil {
						s.Errorf("Close error: %+v", err)
					}
					continue
				} else {
					s.Tracef("Set read timeout: %+v", readTimeout)
				}

				s.Infof("SCTP Accept from: %s", newConn.RemoteAddr().String())
				s.connections.Store(newConn, newConn)

				go s.handleConnection(newConn, readBufSize)
			} else {
				s.Tracef("Listener timeouted")
			}
		}
	}
}

func (s *Server) Stop() {
	if s.listener == nil {
		s.Warnf("SCTP server not running")
		return
	}

	s.Debugf("Close SCTP listener...")
	if err := s.listener.Close(); err != nil {
		s.Errorf("SCTP listener may not close normally: %+v", err)
	}
	close(s.done)
	s.connections.Range(func(key, value interface{}) bool {
		conn := value.(net.Conn)
		if err := conn.Close(); err != nil {
			s.Error("Close connection returns error: %+v", err)
		}
		return true
	})
	s.wg.Wait()
	s.Infof("SCTP server closed")
}

func (s *Server) handleConnection(conn *sctp.SCTPConn, bufsize uint32) {
	s.wg.Add(1)
	defer func() {
		// in case Pran calling Stop(), conn.Close() will return EBADF because
		// conn has been already closed
		if err := conn.Close(); err != nil && err != syscall.EBADF {
			s.Errorf("close connection error: %+v", err)
		}
		s.connections.Delete(conn)
		s.wg.Done()
	}()

	gnb := ran.FindRanWithConn(conn)
	remoteaddr := conn.RemoteAddr().String()
	if gnb == nil {
		s.Infof("Create a GnB context for connection from %s", remoteaddr)
		//NOTE: We should not add a new ran to the pool. Only add
		//it after handling of a Setup request message
		gnb = ran.NewRan(conn)
	}

	buf := make([]byte, bufsize)
	for {
		n, info, notification, err := conn.SCTPRead(buf)
		if err != nil {
			switch err {
			case io.EOF, io.ErrUnexpectedEOF:
				s.Errorf("Connection %s closed", remoteaddr) //either PRAN or remote terminates the connection
				ran.RemoveRan(gnb)
				return
			case syscall.EAGAIN:
				s.Trace("SCTP read timeout")
				continue
			case syscall.EINTR:
				s.Debugf("SCTPRead: %+v", err)
				continue
			default:
				s.Errorf("Handle connection[addr: %+v] error: %+v", conn.RemoteAddr(), err)
				return
			}
		}

		if notification != nil {
			s.handleSCTPNotification(gnb, notification)
		} else {
			if info == nil || info.PPID != PPID {
				s.Warnln("Received SCTP PPID != 60, discard this packet")
				continue
			}

			//s.Tracef("Read %d bytes", n)
			//s.Tracef("Packet content:\n%+v", hex.Dump(buf[:n]))
			s.handleMessage(gnb, buf[:n])
		}
	}
}
