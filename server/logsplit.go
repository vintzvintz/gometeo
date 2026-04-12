package server

import (
	"context"
	"io"
	"log/slog"
)

// levelSplitHandler routes log records to different writers based on level.
// Records at slog.LevelWarn and above go to errW; the rest go to outW.
type levelSplitHandler struct {
	out slog.Handler
	err slog.Handler
}

func newLevelSplitHandler(outW, errW io.Writer, opts *slog.HandlerOptions) *levelSplitHandler {
	return &levelSplitHandler{
		out: slog.NewTextHandler(outW, opts),
		err: slog.NewTextHandler(errW, opts),
	}
}

func (h *levelSplitHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.out.Enabled(ctx, level) || h.err.Enabled(ctx, level)
}

func (h *levelSplitHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		return h.err.Handle(ctx, r)
	}
	return h.out.Handle(ctx, r)
}

func (h *levelSplitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelSplitHandler{
		out: h.out.WithAttrs(attrs),
		err: h.err.WithAttrs(attrs),
	}
}

func (h *levelSplitHandler) WithGroup(name string) slog.Handler {
	return &levelSplitHandler{
		out: h.out.WithGroup(name),
		err: h.err.WithGroup(name),
	}
}
