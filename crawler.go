package crawly

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/rubpy/crawly/clog"
	"github.com/rubpy/crawly/csync"
)

//////////////////////////////////////////////////

type AnyCrawler interface {
	Logger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Log(ctx context.Context, params clog.Params)

	Tracked() (handles []Handle)
	IsTracked(handle Handle) bool
	Track(ctx context.Context, handle Handle) (tracked bool, err error)
	Untrack(ctx context.Context, handle Handle) (tracked bool, err error)
	UntrackAll(ctx context.Context) (untracked int, err error)

	Paused() bool
	Pause(ctx context.Context)
	Resume(ctx context.Context)
	Immediate(ctx context.Context, in time.Duration) (ok bool, err error)

	Active() bool
	Start(ctx context.Context, sessionSettings SessionSettings) error
	Stop(ctx context.Context) (ok bool, err error)
	Listen() csync.Listener[*Result]
}

type Crawler struct {
	logger   *slog.Logger
	settings csync.Value[CrawlerSettings]
	session  csync.Session[*Result]

	orders   csync.Map[Handle, Order]
	entities csync.Map[Handle, Entity]

	handlers csync.Value[CrawlerHandlers]
}

type CrawlerHandlers struct {
	Order  OrderHandler
	Entity EntityHandler
}

//////////////////////////////////////////////////

func (cr *Crawler) Logger() *slog.Logger {
	return cr.logger
}

func (cr *Crawler) SetLogger(logger *slog.Logger) {
	cr.logger = logger
}

func (cr *Crawler) Log(ctx context.Context, params clog.Params) {
	if cr.logger == nil {
		return
	}

	clog.WithParams(cr.logger, ctx, params)
}

func (cr *Crawler) Active() bool {
	return cr.session.Active()
}

func (cr *Crawler) Paused() bool {
	return cr.session.Paused()
}

func (cr *Crawler) Pause(ctx context.Context) {
	cr.session.Pause(ctx)
}

func (cr *Crawler) Resume(ctx context.Context) {
	cr.session.Resume(ctx)
}

func (cr *Crawler) Immediate(ctx context.Context, in time.Duration) (ok bool, err error) {
	return cr.session.Immediate(ctx, in)
}

func (cr *Crawler) Start(ctx context.Context, sessionSettings SessionSettings) error {
	return cr.session.Start(ctx, cr.sessionHandler, csync.SessionSettings(sessionSettings))
}

func (cr *Crawler) Stop(ctx context.Context) (ok bool, err error) {
	return cr.session.Stop(ctx)
}

func (cr *Crawler) Listen() csync.Listener[*Result] {
	return cr.session.Listen()
}

//////////////////////////////////////////////////

var NilHandler = errors.New("handler is nil")

func LoadCrawlerHandlers(cr *Crawler) (handlers CrawlerHandlers) {
	if cr == nil {
		return
	}

	return cr.loadHandlers()
}

func SetCrawlerHandlers(cr *Crawler, handlers CrawlerHandlers) {
	if cr == nil {
		return
	}

	cr.setHandlers(handlers)
}

func (cr *Crawler) loadHandlers() CrawlerHandlers {
	return cr.handlers.Load()
}

func (cr *Crawler) setHandlers(handlers CrawlerHandlers) {
	cr.handlers.Store(handlers)
}

//////////////////////////////////////////////////

func (cr *Crawler) sessionHandler(ctx context.Context, sess *csync.Session[*Result]) (result *Result) {
	result = &Result{
		Valid: true,

		SessionID: sess.ID(),
		Pass:      sess.Pass(),

		Orders:   make(map[Handle]TrackingResult),
		Entities: make(map[Handle]TrackingResult),
	}
	defer func() {
		result.Timestamp = time.Now()
	}()

	if result.Err == nil {
		cr.orders.Range(func(handle Handle, order Order) bool {
			result.Idle = false

			var tr TrackingResult
			if err := cr.processOrder(ctx, &order, &tr); err != nil {
				result.Err = err
				return false
			}

			cr.commitTrackingResult(&tr)
			result.Orders[handle] = tr

			return true
		})
	}

	if result.Err == nil {
		cr.entities.Range(func(handle Handle, entity Entity) bool {
			result.Idle = false

			var tr TrackingResult
			if err := cr.processEntity(ctx, &entity, &tr); err != nil {
				result.Err = err
				return false
			}

			cr.commitTrackingResult(&tr)
			result.Entities[handle] = tr

			return true
		})
	}

	return
}
