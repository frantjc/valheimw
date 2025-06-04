package logutil_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/frantjc/sindri/internal/logutil"
	"github.com/spf13/pflag"
)

func TestSlogConfigAddFlags(t *testing.T) {
	var (
		slogConfig = new(logutil.SlogConfig)
		flagSet    = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	if err := os.Unsetenv("DEBUG"); err != nil {
		t.Fatalf("failed to unset DEBUG environment variable: %v", err)
	}

	slogConfig.AddFlags(flagSet)

	if slogConfig.Level() != slog.LevelError {
		t.Fatalf("expected level %v, got %v", slog.LevelError, slogConfig.Level())
	}

	if err := flagSet.Parse([]string{"--debug"}); err != nil {
		t.Fatalf("failed to set debug flag: %v", err)
	}

	if slogConfig.Level() != slog.LevelDebug {
		t.Fatalf("expected level %v, got %v", slog.LevelDebug, slogConfig.Level())
	}

	if err := flagSet.Parse([]string{"--quiet"}); err != nil {
		t.Fatalf("failed to set quiet flag: %v", err)
	}

	if slogConfig.Level() != slog.LevelError {
		t.Fatalf("expected level %v, got %v", slog.LevelError, slogConfig.Level())
	}

	if err := flagSet.Parse([]string{"-v"}); err != nil {
		t.Fatalf("failed to set V flag: %v", err)
	}

	if slogConfig.Level() != slog.LevelWarn {
		t.Fatalf("expected level %v, got %v", slog.LevelWarn, slogConfig.Level())
	}
}
