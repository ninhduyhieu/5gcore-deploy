package ngap

import (
	"etrib5gc/nfs/pran/ran"
	"etrib5gc/sctp"
	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/ies"
)

func (s *Server) handleMessage(gnb *ran.Ran, pdu []byte) {
	if len(pdu) == 0 {
		s.Error("Receive an empty NGAP Pdu")
		return
	}

	ngapMsg, err, _ := ngap.NgapDecode(pdu)
	if err != nil {
		s.Errorf("Fail to decode an NGAP message: %+v", err)
		return
	}
	//TODO: handle diagnostic information
	switch ngapMsg.Present {
	case ies.NgapPduInitiatingMessage:
		gnb.HandleInitiatingMsg(&ngapMsg.Message)
	case ies.NgapPduSuccessfulOutcome:
		gnb.HandleSuccessfulMsg(&ngapMsg.Message)

	case ies.NgapPduUnsuccessfulOutcome:
		gnb.HandleUnsuccessfulMsg(&ngapMsg.Message)

	default:
		s.Warnf("Unknown NGAP message type: %d", ngapMsg.Present)
	}

}

func (s *Server) handleSCTPNotification(gnb *ran.Ran, notification sctp.Notification) {
	s.Tracef("Handle SCTP Notification[addr: %+v]", gnb.Conn().RemoteAddr())
	switch notification.Type() {
	case sctp.SCTP_ASSOC_CHANGE:
		s.Tracef("SCTP_ASSOC_CHANGE notification")
		event := notification.(*sctp.SCTPAssocChangeEvent)
		switch event.State() {
		case sctp.SCTP_COMM_LOST:
			s.Infof("SCTP state is SCTP_COMM_LOST, close the connection")
			ran.RemoveRan(gnb)
		case sctp.SCTP_SHUTDOWN_COMP:
			s.Infof("SCTP state is SCTP_SHUTDOWN_COMP, close the connection")
			ran.RemoveRan(gnb)
		default:
			s.Warnf("SCTP state[%+v] is not handled", event.State())
		}
	case sctp.SCTP_SHUTDOWN_EVENT:
		s.Infof("SCTP_SHUTDOWN_EVENT notification, close the connection")
		ran.RemoveRan(gnb)
	default:
		s.Warnf("Non handled notification type: 0x%x", notification.Type())
	}
}
