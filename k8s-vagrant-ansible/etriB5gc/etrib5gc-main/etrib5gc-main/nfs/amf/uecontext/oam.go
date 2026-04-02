package uecontext

import (
	"context"
	"etrib5gc/internal/fsm"
	"etrib5gc/nfs/amf/types"
	"github.com/alitto/pond/v2"
)

func (m *PduSessions) GetSessionInfos() (infos []types.SmContextInfo) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, s := range m.sessions {
		if s != nil {
			infos = append(infos, s.WriteInfo())
		}
	}
	return
}

func (ue *UeContext) writeShortInfo() (info types.UeContextShortInfo) {
	info = types.UeContextShortInfo{
		Supi:        ue.supi,
		Suci:        ue.suci,
		Pei:         ue.pei,
		Tmsi:        ue.tmsi,
		MMState:     ue.getMMStateString(),
		NumSessions: ue.sessions.getNum(),
	}

	return
}

func (ue *UeContext) getMMStateString() string {
	switch ue.state.CurrentState() {
	case MM_IDLE:
		return "registered"
	case MM_REGISTERING:
		return "registering"
	case MM_DEREGISTERING:
		return "deregistering"
	default:
	}
	return "unknown"
}

func (ue *UeContext) WriteInfo() (info types.UeContextInfo) {
	info = types.UeContextInfo{
		Supi:       ue.supi,
		Suci:       ue.suci,
		Pei:        ue.pei,
		Tmsi:       ue.tmsi,
		MMState:    ue.getMMStateString(),
		SmContexts: ue.sessions.GetSessionInfos(),
	}
	if ue.plmnId != nil {
		info.PlmnId = *ue.plmnId
	}
	info.GppAccess.Registered = ue.gpp.registered
	info.NonGppAccess.Registered = ue.nonGpp.registered

	if ranUe := ue.gpp.ranUe; ranUe != nil {
		info.GppAccess.Info = ranUe.WriteInfo()
		info.GppAccess.Info.AllowedSlices = ue.getAllowedSlices(true)
	}

	if ranUe := ue.nonGpp.ranUe; ranUe != nil {
		info.NonGppAccess.Info = ranUe.WriteInfo()
		info.NonGppAccess.Info.AllowedSlices = ue.getAllowedSlices(false)
	}
	return
}

func OamGetNumUes() int {
	_uePool.mutex.RLock()
	_uePool.mutex.RUnlock()
	return len(_uePool.byTmsi)
}

func OamGetUeList() (ueList []types.UeContextShortInfo) {
	_uePool.mutex.RLock()
	_uePool.mutex.RUnlock()
	for _, ue := range _uePool.byTmsi {
		ueList = append(ueList, ue.writeShortInfo())
	}
	return
}

func OamGetUeInfo(supi string) (info *types.UeContextInfo) {
	if ueCtx := _uePool.findUeBySupi(supi); ueCtx != nil {
		tmp := ueCtx.WriteInfo()
		info = &tmp
	}
	return
}

//deregister UE, return true if event is trigger
func OamRelease(ueCtx *UeContext, isGpp bool) bool {
	ueCtx.sendEvent(context.TODO(), fsm.NewEventData(ReleaseContextEvent, isGpp))
	return true
}

//deregister UE, return true if event is trigger
func OamDeregister(ueCtx *UeContext, cause uint8) bool {
	doneCh := make(chan struct{}, 1)
	trigger := &DeregistrationTrigger{
		doneCh:    doneCh,
		targetRan: TARGET_BOTH,
		retry:     false,
	}
	ueCtx.sendEvent(context.TODO(), fsm.NewEventData(DeregisterEvent, trigger))
	//NOTE: no need to wait for deregistration completion
	return true

}

//deregister UE, return true if event is trigger
func OamDeregisterUe(supi string, cause uint8) bool {
	//NOTE: for now we do not use `cause`
	if ueCtx := _uePool.findUeBySupi(supi); ueCtx != nil {
		return OamDeregister(ueCtx, cause)
	}
	return false
}

func OamGetFsmStats() *fsm.FsmInfo {
	return _sm.Info()
}

func getWorkerStats(wp pond.Pool) types.WorkerInfo {
	return types.WorkerInfo{
		NumWorkers:        wp.RunningWorkers(),
		NumSubmittedTasks: wp.SubmittedTasks(),
		NumWaitingTasks:   wp.WaitingTasks(),
		NumDroppedTasks:   wp.DroppedTasks(),
		NumCompletedTasks: wp.SuccessfulTasks(),
	}
}

func OamGetFsmWorkerStats() types.WorkerInfo {
	//deprecated
	return types.WorkerInfo{} //getWorkerStats(_uePool.ueWorkers)
}

func OamGetPublicWorkerStats() types.WorkerInfo {
	return getWorkerStats(_uePool.pubWorkers)
}
