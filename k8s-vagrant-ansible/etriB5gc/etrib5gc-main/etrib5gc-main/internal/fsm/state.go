package fsm

import (
	"etrib5gc/internal/eventmux"
	"sync"
)

type StateType int
type State struct {
	current      StateType
	nextEv       *EventData
	mutex        sync.RWMutex
	slot         *eventmux.Slot[State]
	owner        any
	nextEvSetter func(*State, *EventData)
}

func CreateState(fsm *Fsm, state StateType, owner any) *State {
	s := &State{
		current: state,
		owner:   owner,
	}
	s.slot = fsm.exec.MakeSlot(s)
	return s
}

func GetStateOwner[T any](state *State) (T, bool) {
	if v, ok := state.owner.(T); ok {
		return v, true
	} else {
		var zero T
		return zero, false
	}
}

func (s *State) setState(now StateType) {
	s.mutex.Lock()
	s.current = now
	s.mutex.Unlock()
}

func (s *State) CurrentState() StateType {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.current
}

func (s *State) SetNextEvent(ev *EventData) {
	if s.nextEvSetter == nil {
		panic("SetNextEvent must be called within a FSM callback function")
	}
	s.nextEvSetter(s, ev)
}
