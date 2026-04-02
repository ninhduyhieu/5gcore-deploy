package common

import (
	"sync"
)

// hold a stack of functions to be executed
type fnStack struct {
	list   []func()
	maxlen int
	cur    int
	mux    sync.Mutex
}

func newFnStack(maxlen int) fnStack {
	return fnStack{
		maxlen: maxlen,
		list:   make([]func(), maxlen, maxlen),
		cur:    -1,
	}
}

// false if the queue is full
func (s *fnStack) Add(fn func()) bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.cur >= s.maxlen-1 { //the stack is full
		return false
	}
	s.list[s.cur+1] = fn
	s.cur++
	return true
}

func (s *fnStack) Pop() (fn func()) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.cur >= 0 {
		fn = s.list[s.cur]
		s.cur--
	}
	return
}

// a go routine worker to execute functions
type Worker interface {
	Submit(func())
}

// a very simple implmentation of a worker
// note: there are available worker pool implementations
type executer struct {
	jobs fnStack
	quit chan struct{}
}

// create then run
// maxjobsize is the max size of job queue
func NewExecuter(maxjobsize int) Worker {
	e := &executer{
		jobs: newFnStack(maxjobsize),
		quit: make(chan struct{}),
	}
	go e.loop()
	return e
}

// terminate once, don't be stupid
func (e *executer) Terminate() {
	e.quit <- struct{}{}
}

// add job to the executer
// return false of the job queue is full
func (e *executer) Submit(fn func()) {
	e.jobs.Add(fn)
}

// inner loop, exits when Terminate method is called
func (e *executer) loop() {

LOOP:
	for {
		select {
		case <-e.quit:
			break LOOP
		default:
			if fn := e.jobs.Pop(); fn != nil {
				fn()
			}
		}
	}
}
