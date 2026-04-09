package eventmux

import (
	//	"context"
	//	"fmt"
	"sync"
	//	"time"
)

//AsyncTask carry a chanel signal to wait for a task to complete
//You should attach an AsyncTask object to your real task.
//When finish your long task you call Finalize to notify and call <-Wait() to receive
//the completion signal. Finalize is idempotent, so you may want to call it when a
//Context is canceled due to time out in order to clean up resource of your
//running task.

/* EXAMPLE
type MyTask struct {
	*AsyncTask
}

func newMyTask() *MyTask {
	return &MyTask{
		AsyncTask: newAsyncTask(),
	}
}

func test() {
	task := newMyTask()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go runTask(task) //perform the task
	select {
	case <-ctx.Done():
		task.Finalize(ctx.Err())
	case <-task.Wait():
	}
}

func runTask(t *MyTask) {
	t.SetFinalizer(func(fn func(err error)) func(error) {
		return func(err error) {
			fmt.Printf("Finalize me")
			fn(err) //call oritnal finalizer to signal completion
		}
	})
	time.Sleep(10 * time.Second)
	t.Finalize(nil)
}
*/

type AsyncTask struct {
	done        chan error  //to send completion signal
	mu          sync.Mutex  //protect doFinalize
	isFinalized bool        //make finalize idempotem
	doFinalize  func(error) //doing finalize
}

func NewAsyncTask() *AsyncTask {
	t := &AsyncTask{
		done: make(chan error, 1),
	}
	t.doFinalize = func(err error) {
		t.done <- err
	}
	return t
}

func (t *AsyncTask) Wait() chan error {
	return t.done
}

func (t *AsyncTask) Finalize(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.isFinalized {
		return
	}
	if t.doFinalize != nil {
		t.isFinalized = true
		t.doFinalize(err)
	}
}
func (t *AsyncTask) SetFinalizer(setter func(func(error)) func(error)) {
	t.doFinalize = setter(t.doFinalize)
}

type EventData struct {
	evType uint8
	evDat  any
}

func NewEventData[T any](evType uint8, value *T) *EventData {
	return &EventData{
		evType: evType,
		evDat:  value,
	}
}

func NewEmptyEventData(evType uint8) *EventData {
	return &EventData{
		evType: evType,
		evDat:  nil,
	}
}

func (e *EventData) Type() uint8 {
	return e.evType
}

func GetEventData[T any](e *EventData) *T {
	if e.evDat == nil {
		return nil
	}
	return e.evDat.(*T)
}
