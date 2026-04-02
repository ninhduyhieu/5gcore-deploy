package fsm

import (
	"time"
)

type EventType uint8

const (
	EntryEvent EventType = iota
	ExitEvent
	EventIndexStart
)

type EventData struct {
	evType      EventType
	payload     any
	createdTime time.Time
}

func NewEventData[T any](evType EventType, value T) *EventData {
	return &EventData{evType: evType, payload: value, createdTime: time.Now()}
}
func NewEmptyEventData(evType EventType) *EventData {
	return &EventData{evType: evType, createdTime: time.Now()}
}

func GetEventData[T any](e *EventData) (T, bool) {
	if v, ok := e.payload.(T); ok {
		return v, true
	} else {
		var zero T
		return zero, false
	}
}

func (e *EventData) clone(evType EventType) *EventData {
	return &EventData{evType: evType, payload: e.payload, createdTime: time.Now()}
}
