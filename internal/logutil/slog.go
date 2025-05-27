package logutil

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SloggerFrom returns a *slog.Logger from the logr.Logger in the context.
func SloggerFrom(ctx context.Context) *slog.Logger {
	return slog.New(logr.ToSlogHandler(logr.FromContextOrDiscard(ctx)))
}

// Slogger returns a *slog.Logger. Prefer SloggerFrom.
func Slogger() *slog.Logger {
	return slog.New(logr.ToSlogHandler(log.Log))
}
