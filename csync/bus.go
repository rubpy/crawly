package csync

import (
	"sync"
	"sync/atomic"
	"time"
)

//////////////////////////////////////////////////

type Bus[T any] struct {
	sync.RWMutex
	ready atomic.Bool

	broadcast Broadcaster[T]

	results   chan T
	stop      chan struct{}
	stopped   chan struct{}
	immediate chan time.Duration
	paused    chan struct{}
}

func (bus *Bus[T]) Ready() bool {
	return bus.ready.Load()
}

func (bus *Bus[T]) Setup() {
	bus.Lock()
	defer bus.Unlock()

	defer bus.ready.Store(true)

	if bus.broadcast == nil || bus.broadcast.Closed() {
		bus.broadcast = NewBroadcaster[T](0)
	}

	if bus.results == nil {
		bus.results = make(chan T)
	}
	if bus.stop == nil {
		bus.stop = make(chan struct{})
	}
	if bus.stopped == nil {
		bus.stopped = make(chan struct{})
	}
	if bus.immediate == nil {
		bus.immediate = make(chan time.Duration)
	}
	if bus.paused == nil {
		bus.paused = make(chan struct{})
	}
}

func (bus *Bus[T]) Reset() {
	if !bus.ready.Swap(false) {
		return
	}

	bus.Lock()
	defer bus.Unlock()

	{
		if bus.broadcast != nil && !bus.broadcast.Closed() {
			bus.broadcast.Discard()
		}
		bus.broadcast = nil

		bus.results = nil
		bus.stop = nil
		bus.stopped = nil
		bus.immediate = nil
		bus.paused = nil
	}
}

func (bus *Bus[T]) Broadcast() Broadcaster[T] {
	if !bus.ready.Load() {
		bus.Setup()
	}

	bus.RLock()
	defer bus.RUnlock()

	return bus.broadcast
}

func (bus *Bus[T]) Channels() (results chan T, stop chan struct{}, stopped chan struct{}, immediate chan time.Duration, paused chan struct{}) {
	if !bus.ready.Load() {
		bus.Setup()
	}

	bus.RLock()
	defer bus.RUnlock()

	return bus.results, bus.stop, bus.stopped, bus.immediate, bus.paused
}
