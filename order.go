package crawly

import (
	"context"
	"log/slog"
	"time"

	"github.com/rubpy/crawly/clog"
)

//////////////////////////////////////////////////

type Order struct {
	Command        TrackingCommand `json:"command"`
	Attempt        int             `json:"attempt"`
	LastProcessing time.Time       `json:"last_processing"`

	Handle Handle `json:"handle"`
	Data   any    `json:"data"`
}

type OrderHandler func(ctx context.Context, order *Order, result *TrackingResult) error

//////////////////////////////////////////////////

func (cr *Crawler) processOrder(parentCtx context.Context, order *Order, result *TrackingResult) (err error) {
	if err = parentCtx.Err(); err != nil {
		return
	}

	result.Order.Value = *order
	result.Entity.Value.Handle = order.Handle

	var ctx context.Context
	var cancel context.CancelFunc

	settings := cr.loadSettings()

	timeout := settings.TrackingOrderTimeout
	if timeout > 0 {
		ctx, cancel = context.WithTimeoutCause(parentCtx, timeout, ExceededTrackingOrderTimeout)
	} else {
		ctx, cancel = context.WithCancel(parentCtx)
	}
	defer cancel()

	if !result.Order.Value.LastProcessing.IsZero() {
		elapsed := time.Now().Sub(result.Order.Value.LastProcessing)

		if elapsed < settings.MinimumTrackingOrderDelay {
			return
		}
	}

	maxAttempts := settings.MaximumTrackingOrderAttempts
	if maxAttempts < 1 {
		maxAttempts = -1
	}

	switch result.Order.Value.Command {
	case TrackingCommandStart:
		{
			handlers := cr.loadHandlers()
			if handlers.Order != nil {
				result.Order.Err = handlers.Order(ctx, &result.Order.Value, result)
			} else {
				result.Order.Err = NilHandler
			}

			if result.Order.Err != nil {
				result.Order.Value.Attempt++
			} else {
				result.Order.Value.Attempt = 0
			}

			if result.Order.Err != nil {
				result.Entity.Action = TrackingActionNone

				if result.Order.Err == InvalidHandle || (maxAttempts > 0 && result.Order.Value.Attempt >= maxAttempts) {
					result.Order.Action = TrackingActionRemove
					result.Entity.Action = TrackingActionNone
				} else if result.Order.Action == TrackingActionNone {
					result.Order.Action = TrackingActionUpdate
				}
			} else {
				if result.Order.Action == TrackingActionNone {
					result.Order.Action = TrackingActionRemove
					result.Entity.Action = TrackingActionUpdate
				}
			}
		}

	case TrackingCommandStop:
		result.Order.Action = TrackingActionRemove
		result.Entity.Action = TrackingActionRemove

	default:
		result.Order.Err = InvalidTrackingCommand
		result.Order.Action = TrackingActionRemove
	}

	result.Order.Value.LastProcessing = time.Now()

	{
		lp := clog.Params{
			Message: "process:order",
			Level:   slog.LevelInfo,
			Err:     result.Order.Err,
		}

		{
			g := clog.ParamGroup{
				"handle": result.Order.Value.Handle,
			}
			if result.Order.Value.Attempt > 0 {
				g.Set("attempt", clog.ParamGroup{
					"current": result.Order.Value.Attempt,
					"limit":   maxAttempts,
				})

				if result.Order.Action == TrackingActionRemove {
					g.Set("action", result.Order.Action)
				}
			}

			lp.Set("order", g)
		}
		{
			g := clog.ParamGroup{}
			if result.Entity.Action == TrackingActionRemove {
				g.Set("action", result.Entity.Action)
			}

			if g.Count() > 0 {
				lp.Set("entity", g)
			}
		}

		cr.Log(ctx, lp)
	}

	return
}
