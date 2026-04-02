package uecontext

import (
	"etrib5gc/nfs/amf/sm"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"sync"
)

const (
	MAX_PDU_SESSIONS uint8 = 16
)

// report of session synchronization between UE-AMF-SMFs during
// registration request/service request handling
type SyncPduSessionReport struct {
	StatusList *[MAX_PDU_SESSIONS]bool //list of currently active sessions at core on current access
	ErrPduList []uint8                 //fail to establish pdu sessions
	ErrCauses  []uint8                 //and their causes
	ReactList  *[MAX_PDU_SESSIONS]bool //list of reactivated pdu session

	N2SmInfoList []models.N2SmInfoDownlinkContent //list of pdu session information to send to RAN
	N1MsgList    [][]byte                         //N1Sm messages received from SMFs during synchronization
	N1Msg        []byte                           //N1Msg from pending N1N2
}

func isValidPduSessionId(id uint8) bool {
	return id >= 1 && id <= 15
}

type PduSessions struct {
	ueCtx    *UeContext
	sessions [MAX_PDU_SESSIONS]*sm.SmContext
	mutex    sync.RWMutex
}

func newPduSessions(ueCtx *UeContext) *PduSessions {
	return &PduSessions{
		ueCtx: ueCtx,
	}
}

func (m *PduSessions) getSessionsForAccess(isGpp bool) (sessions []*sm.SmContext) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, smCtx := range m.sessions {
		if smCtx != nil && smCtx.IsGpp() == isGpp {
			sessions = append(sessions, smCtx)
		}
	}
	return
}

func (m *PduSessions) listAll() (sessions []*sm.SmContext) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, smCtx := range m.sessions {
		if smCtx != nil {
			sessions = append(sessions, smCtx)
		}
	}
	return
}

func (m *PduSessions) getNum() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	cnt := 0
	for _, s := range m.sessions {
		if s != nil {
			cnt++
		}
	}
	return cnt
}

func (m *PduSessions) find(id uint8) *sm.SmContext {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if isValidPduSessionId(id) {
		return m.sessions[id]
	}
	return nil
}

func (m *PduSessions) add(smCtx *sm.SmContext) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	id := smCtx.GetId()
	if isValidPduSessionId(id) {
		m.sessions[id] = smCtx
	}
}

func (m *PduSessions) remove(id uint8) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if isValidPduSessionId(id) {
		m.sessions[id] = nil
		return true
	}
	return false
}

func (m *PduSessions) sync(isGpp bool, uplink *nas.UplinkDataStatus, status *nas.PduSessionStatus, allowed *nas.AllowedPduSessionStatus) *SyncPduSessionReport {
	ueCtx := m.ueCtx
	ueCtx.Infof("Start sesstion synchronization")
	tasks := []func(){}

	//create a report
	report := new(SyncPduSessionReport)

	allowedReEstablishment := m.ueCtx.isReEstablishPduSessionAllowed()

	//process UplinkDataStatus

	if uplink != nil {
		// UplinkDataStatus (PDU sessionsto be activated, only associated with the
		// related access). If service request is for signaling, the list should not
		// be included by UE. Alway-on sessions should be in the list event without
		// pending data
		uplinkStatus := uplink.Get()
		ueCtx.Tracef("UplinkStatus=%v", uplinkStatus)

		for id, smCtx := range m.sessions {
			if uplinkStatus[id] {
				if !allowedReEstablishment {
					report.ErrPduList = append(report.ErrPduList, uint8(id))
					report.ErrCauses = append(report.ErrCauses, nas.Cause5GMMRestrictedServiceArea)
				} else {
					if smCtx != nil && smCtx.IsGpp() == isGpp {
						if report.ReactList == nil {
							report.ReactList = new([MAX_PDU_SESSIONS]bool)
						}
						tasks = append(tasks, func() {
							ueCtx.reactivateSession(smCtx, report)
						})
					} else {
						//Session of the same access does not exist
						report.ErrPduList = append(report.ErrPduList, uint8(id))
						report.ErrCauses = append(report.ErrCauses, nas.Cause5GSMProtocolErrorUnspecified)
					}
				}
			}
		}

	}

	if status != nil {
		report.StatusList = new([MAX_PDU_SESSIONS]bool)
		sessionStatus := status.Get()
		ueCtx.Tracef("PduStatus=%v", sessionStatus)
		// PDUSessionStatus indicates the sessions available in the UE (so basically
		// any other sessions existing in the network with same access type should be released
		for id, smCtx := range m.sessions {
			if smCtx != nil && smCtx.IsGpp() == isGpp {
				if sessionStatus[id] { //active session
					(*report.StatusList)[id] = true
				} else { //not active at UE, need to release
					tasks = append(tasks, func() {
						ueCtx.releaseSmContext(smCtx, models.CAUSE_PDU_SESSION_STATUS_MISMATCH)
						report.StatusList[smCtx.GetId()] = false
					})
				}
			}
		}
	}

	if allowed != nil && isGpp {
		allowedStatus := allowed.Get()
		ueCtx.Tracef("AllowedStatus=%v", allowedStatus)
		//create reactivation list
		if report.ReactList == nil {
			report.ReactList = new([MAX_PDU_SESSIONS]bool)
		}

		// AllowedPduSessionStatus - a list of PDU session provided by UE as a response
		// to a paging/notification (non-3gpp) from network; identifying pdu sessions
		// to be transferred to 3GPP access (from non-3gpp)
		for id, smCtx := range m.sessions {
			if allowedStatus[id] { //request to change to 3GPP access
				if smCtx != nil && !smCtx.IsGpp() {
					//change AN type to 3GPP access
					tasks = append(tasks, func() {
						ueCtx.changeSessionAccess(smCtx, report)
					})
				} else {
					report.ReactList[id] = false
				}
			}
		}

	}
	//execute session synchronization with SMFs
	ueCtx.Tracef("Sending synchronization task")
	executeTasks(tasks)
	ueCtx.Infof("Synchronization task completed")
	return report
}
