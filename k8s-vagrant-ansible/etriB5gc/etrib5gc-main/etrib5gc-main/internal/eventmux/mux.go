// Package eventmux provides a keyed executor that processes events strictly
// FIFO per key using a shared worker pool (ants).
//
// Usage overview:
//   exec, _ := eventmux.NewExecutor[MyState](256, 4096, eventmux.Options[MyState]{ ... })
//   slot := exec.MakeSlot[MyState](state)
//   _ = exec.Send(ctx, slot, FooPayload{...})
package eventmux

import (
	"context"
	"errors"
	//	"log"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

const (
	MAX_BATCH   int           = 128
	RETRY_DELAY time.Duration = 5 * time.Second
)

// Kind distinguishes your event types, compact as a byte.
type Kind uint8

// Event carries only the kind + an arbitrary payload.
type Event struct {
	ctx     context.Context
	payload any
}

// Handler runs sequentially per key and may mutate *state in-place.
type Handler[S any] func(ctx context.Context, state *S, payload any) error

// Options configure the executor.
type Options[S any] struct {
	Handler    Handler[S]
	MaxBatch   int           // optional: fairness cap (default 128)
	RetryDelay time.Duration // optional: retry delay if pool is full (default 5ms)
}

// Executor guarantees FIFO event handling per key using a shared worker pool.
type Executor[S any] struct {
	pool *ants.Pool
	opts Options[S]

	//	reg sync.Map // map[string]*Slot[S]
}

// NewExecutor builds an Executor with ants worker pool.
func NewExecutor[S any](workers, queueSize int, opts Options[S]) (*Executor[S], error) {
	if opts.Handler == nil {
		return nil, errors.New("eventmux: Options.Handler is required")
	}
	if opts.MaxBatch <= 0 {
		opts.MaxBatch = MAX_BATCH
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = RETRY_DELAY
	}

	pool, err := ants.NewPool(
		workers,
		ants.WithNonblocking(true),
		ants.WithPreAlloc(true),
		ants.WithMaxBlockingTasks(queueSize),
	)
	if err != nil {
		return nil, err
	}
	return &Executor[S]{pool: pool, opts: opts}, nil
}

// Stop releases the worker pool. Call at shutdown.
func (x *Executor[S]) Stop() { x.pool.Release() }

func (x *Executor[S]) MakeSlot(state *S) *Slot[S] {
	return &Slot[S]{state: state}
}

func (x *Executor[S]) Send(ctx context.Context, s *Slot[S], msg any) error {
	start := false
	s.mu.Lock()
	s.q = append(s.q, &Event{
		ctx:     ctx,
		payload: msg,
	})
	if !s.processing {
		s.processing = true
		start = true
	}
	s.mu.Unlock()

	if start {
		if err := x.pool.Submit(func() { x.drain(s) }); err != nil {
			// Pool temporarily full — retry so this key doesn't stall.
			time.AfterFunc(x.opts.RetryDelay, func() {
				_ = x.pool.Submit(func() { x.drain(s) })
			})
		}
	}
	return nil
}

type Slot[S any] struct {
	mu         sync.Mutex
	q          []*Event
	processing bool
	state      *S
}

func (x *Executor[S]) drain(s *Slot[S]) {
	/*
		defer func() {
			if r := recover(); r != nil {
				log.Printf("eventmux: drain panic: %v", r)
				s.mu.Lock()
				s.processing = false
				s.mu.Unlock()
			}
		}()
	*/
	processed := 0
	for {
		var ev *Event
		s.mu.Lock()
		if len(s.q) == 0 {
			s.processing = false
			s.mu.Unlock()
			return
		}
		ev = s.q[0]
		s.q = s.q[1:]
		state := s.state
		s.mu.Unlock()

		if err := x.opts.Handler(ev.ctx, state, ev.payload); err != nil {
			_ = err // wire to your logger/metrics if desired
		}

		processed++
		if processed >= x.opts.MaxBatch {
			// Yield the worker for fairness; continue later.
			_ = x.pool.Submit(func() { x.drain(s) })
			return
		}
	}
}
