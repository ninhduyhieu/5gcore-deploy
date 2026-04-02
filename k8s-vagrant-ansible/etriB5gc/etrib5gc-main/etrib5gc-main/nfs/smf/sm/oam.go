package sm

import (
	"context"
	"etrib5gc/internal/fsm"
	"github.com/reogac/sbi/models"
)

type SessionShortInfo struct {
	Ref   string
	Supi  string
	Id    uint32
	Slice string
	Dnn   string
	UeIp  string
}

type SessionInfo struct {
	Ref        string
	Supi       string
	Id         uint32
	Slice      string
	Dnn        string
	UeIp       string
	Access     string
	HomePlmnId string
}

func (smCtx *SmContext) WriteInfo() (info SessionInfo) {
	info.Supi = smCtx.supi
	info.Ref = smCtx.idString()
	info.Id = smCtx.pduSessionId
	info.Slice = models.SnssaiToString(*smCtx.snssai)
	info.Dnn = smCtx.dnn
	info.Access = string(smCtx.access)
	info.HomePlmnId = models.PlmnIdToString(*smCtx.homePlmnId)
	if smCtx.ueIp != nil {
		info.UeIp = smCtx.ueIp.IP().String()
	}
	return
}

func (smCtx *SmContext) WriteShortInfo() (info SessionShortInfo) {
	info.Supi = smCtx.supi
	info.Ref = smCtx.idString()
	info.Id = smCtx.pduSessionId
	info.Slice = models.SnssaiToString(*smCtx.snssai)
	info.Dnn = smCtx.dnn
	if smCtx.ueIp != nil {
		info.UeIp = smCtx.ueIp.IP().String()
	}
	return
}

func OamGetSessions() (sessions []SessionShortInfo) {
	for _, s := range _pool.getSmContexts() {
		sessions = append(sessions, s.WriteShortInfo())
	}
	return
}
func OamGetSession(ref string) *SessionInfo {
	if s := FindSmContext(ref); s != nil {
		info := s.WriteInfo()
		return &info
	}
	return nil
}

func OamReleaseSession(ref string) bool {

	if s := FindSmContext(ref); s != nil {
		s.oamRelease()
		return true
	}
	return false
}

func (smCtx *SmContext) oamRelease() {
	doneCh := make(chan error, 1)
	event := fsm.NewEventData(ReleaseTriggerEvent, &EventData{
		evType: RELEASE_FORCE,
		dat:    &doneCh,
	})
	smCtx.sendEvent(context.Background(), event)
	//no need to wait for release complete
}

func OamReleaseSessions() bool {
	for _, s := range _pool.getSmContexts() {
		go s.oamRelease()
	}
	return true
}
