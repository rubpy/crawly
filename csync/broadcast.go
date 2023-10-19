package csync

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

//////////////////////////////////////////////////

type BroadcasterReport[V any] interface {
	Status() (okCount int, failCount int)
	Ok() []Listener[V]
	Fail() []Listener[V]
}

type broadcasterReport[V any] struct {
	ok   []Listener[V]
	fail []Listener[V]
}

func (br *broadcasterReport[V]) pushOk(l Listener[V]) {
	br.ok = append(br.ok, l)
}

func (br *broadcasterReport[V]) pushFail(l Listener[V]) {
	br.fail = append(br.fail, l)
}

func (br *broadcasterReport[V]) Status() (okCount int, failCount int) {
	if br.ok == nil && br.fail == nil {
		return -1, -1
	}

	return len(br.ok), len(br.fail)
}

func (br *broadcasterReport[V]) Ok() []Listener[V]   { return br.ok }
func (br *broadcasterReport[V]) Fail() []Listener[V] { return br.fail }

//////////////////////////////////////////////////

var (
	ClosedBroadcastChannel       = errors.New("closed broadcast channel")
	ExceededBroadcastSendTimeout = errors.New("exceeded broadcast send timeout")
)

type Broadcaster[V any] interface {
	Discard()
	Closed() bool

	Listen() Listener[V]
	SendWithTimeout(pctx context.Context, value V, timeout time.Duration, report bool) (BroadcasterReport[V], error)
	Send(ctx context.Context, value V, report bool) (r BroadcasterReport[V], err error)
	DiscardListener(l Listener[V])
}

type broadcaster[V any] struct {
	listeners Map[Listener[V], chan<- V]
	capacity  int
	closed    atomic.Bool
}

func NewBroadcaster[V any](capacity int) Broadcaster[V] {
	return &broadcaster[V]{
		capacity: capacity,
	}
}

func (bc *broadcaster[V]) SendWithTimeout(pctx context.Context, value V, timeout time.Duration, report bool) (BroadcasterReport[V], error) {
	if bc.closed.Load() {
		return nil, ClosedBroadcastChannel
	}

	if err := pctx.Err(); err != nil {
		return nil, err
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if timeout == 0 {
		ctx, cancel = context.WithCancel(pctx)
	} else {
		ctx, cancel = context.WithTimeoutCause(pctx, timeout, ExceededBroadcastSendTimeout)
	}
	defer cancel()

	r := &broadcasterReport[V]{}

	var cb func(l Listener[V], ch chan<- V) bool
	if timeout == 0 {
		cb = func(l Listener[V], ch chan<- V) bool {
			select {
			case ch <- value:
				if report {
					r.pushOk(l)
				}

			case <-ctx.Done():
				if report {
					r.pushFail(l)
				}

			default:
				if report {
					r.pushFail(l)
				}
			}

			return true
		}
	} else {
		cb = func(l Listener[V], ch chan<- V) bool {
			select {
			case ch <- value:
				if report {
					r.pushOk(l)
				}

			case <-ctx.Done():
				if report {
					r.pushFail(l)
				}
			}

			return true
		}
	}

	bc.listeners.Range(cb)

	return r, nil
}

func (bc *broadcaster[V]) Send(ctx context.Context, value V, report bool) (r BroadcasterReport[V], err error) {
	return bc.SendWithTimeout(ctx, value, 0, report)
}

func (bc *broadcaster[V]) Discard() {
	if bc.closed.Swap(true) {
		return
	}

	bc.listeners.Range(func(l Listener[V], _ chan<- V) bool {
		bc.DiscardListener(l)

		return true
	})
}

func (bc *broadcaster[V]) DiscardListener(l Listener[V]) {
	if l == nil {
		return
	}

	ch, ok := bc.listeners.LoadAndDelete(l)
	if !ok {
		return
	}

	close(ch)
}

func (bc *broadcaster[V]) Listen() Listener[V] {
	ch := make(chan V, bc.capacity)

	l := &listener[V]{
		ch: ch,
		bc: bc,
	}
	if bc.closed.Load() {
		// Returning a 'dummy' listener (i.e., with a closed channel).

		l.closed.Store(true)
		close(ch)
	}

	bc.listeners.Store(l, ch)

	return l
}

func (bc *broadcaster[V]) Closed() bool {
	return bc.closed.Load()
}

//////////////////////////////////////////////////

type Listener[V any] interface {
	Channel() <-chan V
	Discard()
	Redirect(ctx context.Context, destination Broadcaster[V])

	Closed() bool
}

type listener[V any] struct {
	ch     <-chan V
	bc     Broadcaster[V]
	closed atomic.Bool
}

func (l *listener[V]) Closed() bool {
	return l.closed.Load()
}

func (l *listener[V]) Discard() {
	if l.closed.Swap(true) {
		return
	}

	l.bc.DiscardListener(l)
}

func (l *listener[V]) Channel() <-chan V {
	return l.ch
}

func (l *listener[V]) Redirect(ctx context.Context, destination Broadcaster[V]) {
	if l.closed.Load() {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}
	defer l.Discard()

	ch := l.ch
	for {
		select {
		case <-ctx.Done():
			return

		case res, ok := <-ch:
			if !ok {
				return
			}

			_, _ = destination.Send(ctx, res, false)
		}
	}
}
