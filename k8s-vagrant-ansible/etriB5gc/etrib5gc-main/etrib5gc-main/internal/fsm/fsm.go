// Package fsm implements a finite state machine backed by a keyed executor.
package fsm

import (
	"context"
	"etrib5gc/internal/eventmux"
	"fmt"
	//	"log"
	"time"
)

type StateEventTuple struct {
	state StateType
	event EventType
}

func Tuple(state StateType, event EventType) StateEventTuple {
	return StateEventTuple{state: state, event: event}
}

type Transitions map[StateEventTuple]StateType
type CallbackFn func(context.Context, *State, EventType, any)
type Callbacks map[StateType]CallbackFn

type Fsm struct {
	transitions   Transitions
	callbacks     Callbacks
	commonEvents  map[EventType]bool
	commonHandler CallbackFn
	metrics       FsmMetrics
	exec          *eventmux.Executor[State]
}

type Options struct {
	Transitions    Transitions
	Callbacks      Callbacks
	CommonCallback CallbackFn
	CommonEvents   []EventType
}

func NewFsm(opts Options, workers, queue int) *Fsm {
	f := &Fsm{
		transitions:   opts.Transitions,
		callbacks:     opts.Callbacks,
		commonEvents:  map[EventType]bool{},
		commonHandler: opts.CommonCallback,
		metrics:       newFsmMetrics(),
	}

	valid := map[EventType]bool{}
	for _, ev := range opts.CommonEvents {
		f.commonEvents[ev] = true
		valid[ev] = true
	}
	// Collect all event types that appear in transitions
	for k := range opts.Transitions {
		valid[k.event] = true
	}

	// One common handler
	handler := func(ctx context.Context, st *State, payload any) error {
		//	log.Printf("Receive message")
		q := payload.(*queued)
		ev := q.ev
		t0 := time.Now()
		f.metrics.onTriggered()

		if f.commonEvents[ev.evType] {
			/*
				if q.ack != nil {
					//log.Printf("common event %d is handled", ev.evType)
					q.ack <- nil
				}
			*/
			f.executeCallback(f.commonHandler, ctx, st, ev)
		} else {
			f.transit(ctx, st, ev, q.ack)
		}

		f.metrics.onCompleted(ev.evType, t0)
		f.processNextEvent(ctx, st)
		return nil
	}

	ex, err := eventmux.NewExecutor[State](workers, queue,
		eventmux.Options[State]{
			Handler: handler,
		},
	)
	if err != nil {
		panic(err)
	}
	f.exec = ex
	return f
}
func (fsm *Fsm) Stop() { fsm.exec.Stop() }

type queued struct {
	ev  *EventData
	ack chan error
}

func (fsm *Fsm) SendEvent(ctx context.Context, st *State, ev *EventData) chan error {
	ch := make(chan error, 1)
	q := &queued{ev: ev, ack: ch}
	if err := fsm.exec.Send(ctx, st.slot, q); err != nil {
		ch <- err
		return ch
	}
	fsm.metrics.onSubmitted()

	if fsm.commonEvents[ev.evType] {
		ch <- nil //common events are always accepted
	}
	return ch
}

func (fsm *Fsm) processNextEvent(ctx context.Context, state *State) {
	for state.nextEv != nil {
		t := time.Now()
		fsm.metrics.onSubmitted()
		fsm.metrics.onTriggered()
		nextEv := state.nextEv
		state.nextEv = nil
		if fsm.commonEvents[nextEv.evType] {
			fsm.executeCallback(fsm.commonHandler, ctx, state, nextEv)
		} else {
			fsm.transit(ctx, state, nextEv, nil)
		}
		fsm.metrics.onCompleted(nextEv.evType, t)
	}
}

func (fsm *Fsm) executeCallback(callback CallbackFn, ctx context.Context, state *State, event *EventData) {
	if callback == nil {
		return
	}
	state.nextEvSetter = fsm.setNextEventSetter(state.current, event.evType)
	callback(ctx, state, event.evType, event.payload)
	state.nextEvSetter = nil
}

func (fsm *Fsm) setNextEventSetter(current StateType, evType EventType) func(*State, *EventData) {
	return func(state *State, ev *EventData) {
		if evType == ExitEvent {
			panic("SetNextEvent in an ExitEvent callback is not allowed")
		}
		if fsm.isTransited(current, evType) {
			panic("SetNextEvent right after a transit is not allowed")
		}
		if state.nextEv != nil {
			panic("Multiple SetNextEvent is called")
		}
		state.nextEv = ev
	}
}

func (fsm *Fsm) isTransited(current StateType, evType EventType) bool {
	if !fsm.commonEvents[evType] {
		if nextState, ok := fsm.transitions[StateEventTuple{state: current, event: evType}]; ok {
			return nextState != current
		}
	}
	return false
}

func (fsm *Fsm) transit(ctx context.Context, state *State, event *EventData, errCh chan error) {
	current := state.CurrentState()
	if nextState, ok := fsm.transitions[StateEventTuple{state: current, event: event.evType}]; ok {
		if errCh != nil {
			//log.Printf("transit event %d is handled", event.evType)
			errCh <- nil
		}
		curCallback := fsm.callbacks[current]
		nextCallback := fsm.callbacks[nextState]
		fsm.executeCallback(curCallback, ctx, state, event)
		if current != nextState {
			fsm.executeCallback(curCallback, ctx, state, event.clone(ExitEvent))
			state.setState(nextState)
			fsm.executeCallback(nextCallback, ctx, state, event.clone(EntryEvent))
		}
	} else {
		if errCh != nil {
			errCh <- fmt.Errorf("Unknown transition from state %v with event %v", current, event)
		}
	}
}

func (fsm *Fsm) Info() *FsmInfo {
	return fsm.metrics.getInfo()
}
