package crawly

import (
	"context"
	"log/slog"
	"time"

	"github.com/rubpy/crawly/clog"
)

//////////////////////////////////////////////////

type Entity struct {
	Attempt        int       `json:"attempt"`
	LastProcessing time.Time `json:"last_processing"`

	Handle Handle `json:"handle"`
	Data   any    `json:"data"`
}

type EntityHandler func(ctx context.Context, entity *Entity, result *TrackingResult) error

//////////////////////////////////////////////////

func (cr *Crawler) processEntity(parentCtx context.Context, entity *Entity, result *TrackingResult) (err error) {
	if err = parentCtx.Err(); err != nil {
		return
	}

	result.Entity.Value = *entity

	var ctx context.Context
	var cancel context.CancelFunc

	settings := cr.loadSettings()

	timeout := settings.TrackingTimeout
	if timeout > 0 {
		ctx, cancel = context.WithTimeoutCause(parentCtx, timeout, ExceededTrackingTimeout)
	} else {
		ctx, cancel = context.WithCancel(parentCtx)
	}
	defer cancel()

	if !result.Entity.Value.LastProcessing.IsZero() {
		elapsed := time.Now().Sub(result.Entity.Value.LastProcessing)

		if elapsed < settings.MinimumTrackingDelay {
			return
		}
	}

	maxAttempts := settings.MaximumTrackingAttempts
	if maxAttempts < 1 {
		maxAttempts = -1
	}

	handlers := cr.loadHandlers()
	if handlers.Entity != nil {
		result.Entity.Err = handlers.Entity(ctx, &result.Entity.Value, result)
	} else {
		result.Entity.Err = NilHandler
	}

	if result.Entity.Err != nil {
		result.Entity.Value.Attempt++
	} else {
		result.Entity.Value.Attempt = 0
	}

	if result.Entity.Err != nil {
		if result.Entity.Err == InvalidHandle || (maxAttempts > 0 && result.Entity.Value.Attempt >= maxAttempts) {
			result.Entity.Action = TrackingActionRemove
		} else if result.Entity.Action == TrackingActionNone {
			result.Entity.Action = TrackingActionUpdate
		}
	} else {
		if result.Entity.Action == TrackingActionNone {
			result.Entity.Action = TrackingActionUpdate
		}
	}

	result.Entity.Value.LastProcessing = time.Now()

	{
		lp := clog.Params{
			Message: "process:entity",
			Level:   slog.LevelInfo,
			Err:     result.Entity.Err,
		}

		{
			g := clog.ParamGroup{
				"handle": result.Entity.Value.Handle,
			}
			if result.Entity.Value.Attempt > 0 {
				g.Set("attempt", clog.ParamGroup{
					"current": result.Entity.Value.Attempt,
					"limit":   maxAttempts,
				})

				if result.Entity.Action == TrackingActionRemove {
					g.Set("action", result.Entity.Action)
				}
			}

			lp.Set("entity", g)
		}

		cr.Log(ctx, lp)
	}

	return
}
