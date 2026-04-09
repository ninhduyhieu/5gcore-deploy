package common

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var log *logrus.Entry

func init() {
	log = logrus.WithFields(logrus.Fields{"mod": "common"})
}

type Timer interface {
	Stop()
	Start() //Start or Reset timer
}

type timerImpl struct {
	d     time.Duration
	e     Worker
	abort chan bool
	_t    *time.Timer
	fn    func() //callback
	wg    sync.WaitGroup
	mutex sync.Mutex
}

// create an idling timer
func NewTimer(d time.Duration, fn func(), e Worker) Timer {
	return &timerImpl{
		d:  d,
		fn: fn,
		e:  e,
	}
}

// start/restart the timer
func (t *timerImpl) Start() {
	//make sure the timer is stopped
	t.Stop()

	t.abort = make(chan bool)
	t._t = time.NewTimer(t.d) //always create a new one

	t.wg.Add(1)
	if t.e != nil { //submit to workerpool
		t.e.Submit(t.run)
	} else {
		go t.run() //create new go routine
	}

}

func (t *timerImpl) run() {
	defer t.wg.Done()
	select {
	case <-t.abort:
		//just exit
		t._t.Stop()
	case <-t._t.C:
		t.mutex.Lock()
		t.abort = nil
		t.mutex.Unlock()
		//execute callback function
		if t.fn != nil {
			t.fn()
		}
	}
}

func (t *timerImpl) Stop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.abort != nil { //make the call idempotem
		close(t.abort) //trigger canceling
		t.wg.Wait()    //make sure the running goroutine to complete
		t.abort = nil  //make the call idempotem
	}
}
