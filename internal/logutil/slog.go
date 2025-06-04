package logutil

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type contextKey struct{}

// SloggerInto returns a new context with a *slog.Logger stored in it.
func SloggerInto(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

// SloggerFrom returns a *slog.Logger from the logr.Logger in the context.
func SloggerFrom(ctx context.Context) *slog.Logger {
	v := ctx.Value(contextKey{})
	if v == nil {
		return slog.New(slog.DiscardHandler)
	}

	switch v := v.(type) {
	case *slog.Logger:
		return v
	default:
		return slog.New(slog.DiscardHandler)
	}
}

// Slogger returns a *slog.Logger. Prefer SloggerFrom.
func Slogger() *slog.Logger {
	return slog.New(logr.ToSlogHandler(log.Log))
}

type SlogConfig struct {
	level *slog.Level
}

func (s *SlogConfig) init() {
	if s.level == nil {
		l := slog.LevelError
		if os.Getenv("DEBUG") != "" {
			l = slog.LevelDebug
		}
		s.level = &l
	}
}

// Level implements slog.Leveler.
func (s *SlogConfig) Level() slog.Level {
	s.init()
	return *s.level
}

func (s *SlogConfig) AddFlags(flags *pflag.FlagSet) {
	s.init()
	flags.AddFlag(&pflag.Flag{
		Name:      "debug",
		Shorthand: "d",
		Value: &genericBool[slog.Level]{
			Value: s.level,
			IfSet: slog.LevelDebug,
		},
		NoOptDefVal: "true",
		Usage:       "Print debug logs",
	})
	flags.AddFlag(&pflag.Flag{
		Name:      "quiet",
		Shorthand: "q",
		Value: &genericBool[slog.Level]{
			Value: s.level,
			IfSet: slog.LevelError,
		},
		NoOptDefVal: "true",
		Usage:       "Minimize logs",
	})
	flags.AddFlag(&pflag.Flag{
		Name:      "verbose",
		Shorthand: "v",
		Value: &incrementalCount[slog.Level]{
			Value:     s.level,
			Increment: slog.LevelWarn - slog.LevelError,
		},
		NoOptDefVal: "+1",
		Usage:       "More vebose logging",
	})
}
