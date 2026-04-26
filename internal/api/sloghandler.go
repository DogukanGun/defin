package api

import (
	"context"
	"log/slog"
	"strings"
)

// HubHandler wraps another slog.Handler and forwards every record to a Hub.
type HubHandler struct {
	inner slog.Handler
	hub   *Hub
}

func NewHubHandler(inner slog.Handler, hub *Hub) *HubHandler {
	return &HubHandler{inner: inner, hub: hub}
}

func (h *HubHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *HubHandler) Handle(ctx context.Context, r slog.Record) error {
	levelStr := r.Level.String()

	// Build message with key=value attrs appended
	msg := r.Message
	r.Attrs(func(a slog.Attr) bool {
		msg += " " + a.Key + "=" + a.Value.String()
		return true
	})

	h.hub.Send(Event{
		Type:    EventLog,
		Level:   strings.ToUpper(levelStr),
		Message: msg,
	})

	return h.inner.Handle(ctx, r)
}

func (h *HubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &HubHandler{inner: h.inner.WithAttrs(attrs), hub: h.hub}
}

func (h *HubHandler) WithGroup(name string) slog.Handler {
	return &HubHandler{inner: h.inner.WithGroup(name), hub: h.hub}
}
