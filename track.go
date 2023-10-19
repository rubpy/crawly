package crawly

import (
	"context"
	"errors"
)

//////////////////////////////////////////////////

func (cr *Crawler) Tracked() (handles []Handle) {
	handles = []Handle{}

	cr.entities.Range(func(handle Handle, _ Entity) bool {
		handles = append(handles, handle)

		return true
	})

	return
}

func (cr *Crawler) IsTracked(handle Handle) bool {
	return cr.entities.Has(handle)
}

func (cr *Crawler) Track(ctx context.Context, handle Handle) (tracked bool, err error) {
	return cr.order(ctx, handle, TrackingCommandStart, false)
}

func (cr *Crawler) Untrack(ctx context.Context, handle Handle) (tracked bool, err error) {
	return cr.order(ctx, handle, TrackingCommandStop, false)
}

func (cr *Crawler) UntrackAll(ctx context.Context) (untracked int, err error) {
	cr.entities.Range(func(handle Handle, _ Entity) bool {
		cr.order(ctx, handle, TrackingCommandStop, false)

		untracked++
		return true
	})

	return
}

func (cr *Crawler) order(ctx context.Context, handle Handle, command TrackingCommand, quiet bool) (tracked bool, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !handle.Valid() {
		err = InvalidHandle
		return
	}

	if cr.IsTracked(handle) {
		tracked = true
	}

	switch command {
	case TrackingCommandStart:
		if tracked {
			return
		}

	case TrackingCommandStop:
		if !tracked {
			return
		}

	default:
		err = InvalidTrackingCommand
		return
	}

	cr.orders.Store(handle, Order{
		Command: command,
		Handle:  handle,
	})

	if !quiet && cr.session.PauseIdle() {
		if cr.session.Paused() {
			cr.session.Resume(ctx)
		}
	}

	return
}

//////////////////////////////////////////////////

var (
	InvalidTrackingCommand = errors.New("invalid tracking command")

	ExceededTrackingOrderTimeout = errors.New("exceeded tracking order timeout")
	ExceededTrackingTimeout      = errors.New("exceeded tracking timeout")
)

type TrackingResult struct {
	Order  actionableResult[Order]  `json:"order"`
	Entity actionableResult[Entity] `json:"entity"`
}

type actionableResult[T any] struct {
	Action TrackingAction

	Value T
	Err   error
}

func (cr *Crawler) commitTrackingResult(tr *TrackingResult) {
	if tr == nil {
		return
	}

	h := tr.Order.Value.Handle
	if h != nil && h.Valid() {
		switch tr.Order.Action {
		case TrackingActionRemove:
			cr.orders.Delete(h)
		case TrackingActionUpdate:
			cr.orders.Store(h, tr.Order.Value)
		}
	}

	h = tr.Entity.Value.Handle
	if h != nil && h.Valid() {
		switch tr.Entity.Action {
		case TrackingActionRemove:
			cr.entities.Delete(h)
		case TrackingActionUpdate:
			cr.entities.Store(h, tr.Entity.Value)
		}
	}
}

//////////////////////////////////////////////////

type TrackingCommand uint

const (
	TrackingCommandNone TrackingCommand = iota
	TrackingCommandStart
	TrackingCommandStop
)

func (cmd TrackingCommand) String() string {
	switch cmd {
	case TrackingCommandStart:
		return "start"
	case TrackingCommandStop:
		return "stop"
	}

	return "none"
}

//////////////////////////////////////////////////

type TrackingAction uint

const (
	TrackingActionNone TrackingAction = iota
	TrackingActionUpdate
	TrackingActionRemove
)

func (act TrackingAction) String() string {
	switch act {
	case TrackingActionUpdate:
		return "update"
	case TrackingActionRemove:
		return "remove"
	}

	return "none"
}
