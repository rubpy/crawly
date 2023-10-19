package csync

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

//////////////////////////////////////////////////

var (
	SessionAlreadyActive   = errors.New("session is already active")
	SessionInvalidInterval = errors.New("invalid session interval")

	ExceededSessionPassTimeout = errors.New("exceeded session pass timeout")
)

var (
	MinimumSessionInterval = 1 * time.Second
)

type Session[T any] struct {
	active    atomic.Bool
	paused    atomic.Bool
	pauseIdle atomic.Bool

	id   string
	pass uint64

	bus Bus[T]
}

type Handler[T any] func(ctx context.Context, sess *Session[T]) T

type SessionSettings struct {
	Interval          time.Duration `json:"interval"`
	SinglePassTimeout time.Duration `json:"single_pass_timeout"`

	Paused    bool `json:"paused"`
	PauseIdle bool `json:"pause_idle"`
}

//////////////////////////////////////////////////

func (sess *Session[T]) Active() bool {
	return sess.active.Load()
}

func (sess *Session[T]) ID() string {
	return sess.id
}

func (sess *Session[T]) Pass() uint64 {
	return atomic.LoadUint64(&sess.pass)
}

func (sess *Session[T]) IncrementPass() uint64 {
	return atomic.AddUint64(&sess.pass, 1)
}

func (sess *Session[T]) Listen() (listener Listener[T]) {
	broadcast := sess.bus.Broadcast()

	return broadcast.Listen()
}

func (sess *Session[T]) halt(ctx context.Context) {
	if ctx != nil && ctx.Err() != nil {
		return
	}

	sess.active.Store(false)
	sess.bus.Reset()
}

func (sess *Session[T]) PauseIdle() bool {
	return sess.pauseIdle.Load()
}

func (sess *Session[T]) SetPauseIdle(ctx context.Context, pauseIdle bool) {
	sess.pauseIdle.Swap(pauseIdle)
}

func (sess *Session[T]) Paused() bool {
	return sess.paused.Load()
}

func (sess *Session[T]) SetPaused(ctx context.Context, paused bool) {
	if sess.paused.Swap(paused) != paused {
		_, _, _, _, ch := sess.bus.Channels()

		if ch != nil {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
}

func (sess *Session[T]) Pause(ctx context.Context) {
	sess.SetPaused(ctx, true)
}

func (sess *Session[T]) Resume(ctx context.Context) {
	sess.SetPaused(ctx, false)
}

func (sess *Session[T]) Immediate(parentCtx context.Context, in time.Duration) (ok bool, err error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	} else if err = parentCtx.Err(); err != nil {
		return
	}

	if !sess.active.Load() {
		return
	}

	_, _, _, immediate, _ := sess.bus.Channels()
	if immediate == nil {
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	select {
	case immediate <- in:

	case <-ctx.Done():
		err = ctx.Err()
		return
	}

	ok = true
	return
}

func (sess *Session[T]) Stop(parentCtx context.Context) (ok bool, err error) {
	if parentCtx == nil {
		parentCtx = context.Background()
	} else if err = parentCtx.Err(); err != nil {
		return
	}

	if !sess.active.Load() {
		return
	}

	_, stop, stopped, _, _ := sess.bus.Channels()
	if stop == nil || stopped == nil {
		// NOTE: this should not happen.
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	select {
	case stop <- struct{}{}:

	case <-ctx.Done():
		err = ctx.Err()
		return
	}

	select {
	case <-stopped:

	case <-ctx.Done():
		err = ctx.Err()
		return
	}

	ok = true
	return
}

func (sess *Session[T]) Start(ctx context.Context, handler Handler[T], settings SessionSettings) error {
	if ctx == nil {
		ctx = context.Background()
	} else if err := ctx.Err(); err != nil {
		return err
	}

	if settings.Interval < MinimumSessionInterval {
		return SessionInvalidInterval
	}

	if sess.active.Swap(true) {
		return SessionAlreadyActive
	}
	sess.paused.Store(settings.Paused)
	sess.pauseIdle.Store(settings.PauseIdle)

	sess.id = uniqueHex()
	sess.pass = 0

	go sess.run(ctx, handler, settings.Interval, settings.SinglePassTimeout)

	return nil
}

func (sess *Session[T]) run(parentCtx context.Context, handler Handler[T], interval time.Duration, singlePassTimeout time.Duration) {
	var cooldown <-chan time.Time

	broadcast := sess.bus.Broadcast()
	results, stop, stopped, immediate, paused := sess.bus.Channels()

handleLoop:
	for {
		if err := parentCtx.Err(); err != nil {
			break handleLoop
		}

		if !sess.paused.Load() {
			if cooldown == nil {
				go func(ctx context.Context, sess *Session[T], handler Handler[T], results chan<- T, singlePassTimeout time.Duration) {
					var handlerCtx context.Context
					var cancel context.CancelFunc

					if singlePassTimeout > 0 {
						handlerCtx, cancel = context.WithTimeoutCause(ctx, singlePassTimeout, ExceededSessionPassTimeout)
					} else {
						handlerCtx, cancel = context.WithCancel(ctx)
					}
					defer cancel()

					results <- handler(handlerCtx, sess)
				}(parentCtx, sess, handler, results, singlePassTimeout)
			}
		}

		select {
		case <-stop:
			break handleLoop

		case <-cooldown:
			cooldown = nil
			continue

		case t := <-immediate:
			if t <= 0 {
				cooldown = nil
			} else {
				cooldown = time.After(t)
			}
			continue

		case <-paused:
			cooldown = nil
			continue

		case <-parentCtx.Done():

		case result := <-results:
			valid := true
			if v, ok := any(result).(interface{ IsValid() bool }); ok {
				valid = v.IsValid()
			}

			if valid {
				if broadcast != nil {
					broadcast.Send(parentCtx, result, false)
				}

				if sess.pauseIdle.Load() {
					idle := false
					if v, ok := any(result).(interface{ IsIdle() bool }); ok {
						idle = v.IsIdle()
					}

					if idle {
						sess.paused.Store(true)
					}
				}

				sess.IncrementPass()
			}
		}

		if !sess.paused.Load() {
			cooldown = time.After(interval)
		} else {
			cooldown = nil
		}
	}

	select {
	case stopped <- struct{}{}:
		sess.halt(parentCtx)
	default:
	}

	return
}
