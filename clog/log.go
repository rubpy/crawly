package clog

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

//////////////////////////////////////////////////

func WithParams(logger *slog.Logger, ctx context.Context, params Params) {
	if logger == nil {
		return
	}

	attrs, level := params.Serialize()
	WithAttrs(logger, ctx, level, params.Message, attrs...)
}

func WithAttrs(logger *slog.Logger, ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	if logger == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if !logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])

	r := slog.NewRecord(
		time.Now(),
		level,
		msg,
		pcs[0],
	)
	r.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}
