package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// TextHandler formats log records as: [HH:MM:SS] step: message [key=value ...]
//
// If the "step" attribute is present (via WithAttrs), it is displayed before the
// message. Other attributes are appended after the message.
type TextHandler struct {
	w        io.Writer
	minLevel slog.Level
	preAttrs []slog.Attr
	mu       *sync.Mutex
}

// NewTextHandler creates a TextHandler that writes to w at the given minimum level.
func NewTextHandler(w io.Writer, minLevel slog.Level) *TextHandler {
	return &TextHandler{
		w:        w,
		minLevel: minLevel,
		mu:       &sync.Mutex{},
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *TextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle formats and writes the log record.
func (h *TextHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Level < h.minLevel {
		return nil
	}

	allAttrs := make([]slog.Attr, 0, len(h.preAttrs)+r.NumAttrs())
	allAttrs = append(allAttrs, h.preAttrs...)
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	var step string
	var otherAttrs []slog.Attr
	for _, a := range allAttrs {
		if a.Key == "step" {
			step = a.Value.String()
		} else {
			otherAttrs = append(otherAttrs, a)
		}
	}

	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(r.Time.Format("15:04:05"))
	sb.WriteString("] ")
	if step != "" {
		sb.WriteString(step)
		sb.WriteString(": ")
	}
	sb.WriteString(r.Message)
	for _, a := range otherAttrs {
		fmt.Fprintf(&sb, " %s=%v", a.Key, a.Value.Any())
	}
	sb.WriteString("\n")

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, sb.String())
	return err
}

// WithAttrs returns a new handler with the given attributes pre-set.
func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.preAttrs)+len(attrs))
	copy(newAttrs, h.preAttrs)
	copy(newAttrs[len(h.preAttrs):], attrs)
	return &TextHandler{
		w:        h.w,
		minLevel: h.minLevel,
		preAttrs: newAttrs,
		mu:       h.mu,
	}
}

// WithGroup returns the handler unchanged (groups are not supported).
func (h *TextHandler) WithGroup(_ string) slog.Handler {
	return h
}

// FanOutHandler routes log records to multiple handlers.
type FanOutHandler struct {
	handlers []slog.Handler
}

// NewFanOutHandler creates a FanOutHandler that routes to all provided handlers.
func NewFanOutHandler(handlers ...slog.Handler) *FanOutHandler {
	return &FanOutHandler{handlers: handlers}
}

// Enabled reports whether any handler handles the given level.
func (h *FanOutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle dispatches the record to all handlers that accept its level.
func (h *FanOutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// WithAttrs returns a new FanOutHandler where each sub-handler has the given attributes.
func (h *FanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &FanOutHandler{handlers: handlers}
}

// WithGroup returns a new FanOutHandler where each sub-handler uses the given group.
func (h *FanOutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &FanOutHandler{handlers: handlers}
}
