package types

import (
	"github.com/reogac/sbi/models"
)

type RanState struct {
	Registered bool
	Info       *RanUeInfo
}

type UeContextInfo struct {
	Supi         string
	Suci         string
	Pei          string
	Tmsi         uint32
	MMState      string
	GppAccess    RanState
	NonGppAccess RanState
	PlmnId       models.PlmnId
	SmContexts   []SmContextInfo
}

type UeContextShortInfo struct {
	Supi        string
	Suci        string
	Pei         string
	Tmsi        uint32
	MMState     string
	NumSessions int
}

type RanUeInfo struct {
	Access                   models.AccessType
	RanUeId                  models.RanUeId
	AllowedSlices            []models.AllowedSnssai
	LocalId                  int64
	CreatedTime              string
	LastRegistrationTime     string
	LastRegistrationDuration int64
}

type SmContextInfo struct {
	SmfInfo      string
	Dnn          string
	Slice        string
	Is3Gpp       bool
	PduSessionId int
	SmCtxRef     string
}

type WorkerInfo struct {
	NumWorkers        int64
	NumSubmittedTasks uint64
	NumWaitingTasks   uint64
	NumDroppedTasks   uint64
	NumCompletedTasks uint64
}
